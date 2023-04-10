package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/fcgi"
	"path"
	"strconv"

	"github.com/uptrace/bunrouter"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func initServer() {
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()

	listener, err := net.Listen("tcp", systemCritical.ListenIP+":"+strconv.Itoa(systemCritical.Port))
	if err != nil {
		if !systemCritical.DebugMode {
			fmt.Printf("Failed listening on %s:%d: %s", systemCritical.ListenIP, systemCritical.Port, err.Error())
		}
		gcutil.Logger().Fatal().Caller().
			Err(err).
			Str("ListenIP", systemCritical.ListenIP).
			Int("Port", systemCritical.Port).Send()
	}

	// Check if Akismet API key is usable at startup.
	err = serverutil.CheckAkismetAPIKey(siteConfig.AkismetAPIKey)
	if err != nil && err != serverutil.ErrBlankAkismetKey {
		if !systemCritical.DebugMode {
			fmt.Println("Got error when initializing Akismet spam protection, it will be disabled:", err)
		}
		gcutil.Logger().Fatal().Caller().
			Err(err).
			Msg("Akismet spam protection will be disabled")
	}
	router := server.GetRouter()
	router.GET(config.WebPath("/captcha"), bunrouter.HTTPHandlerFunc(posting.ServeCaptcha))
	router.POST(config.WebPath("/captcha"), bunrouter.HTTPHandlerFunc(posting.ServeCaptcha))
	router.GET(config.WebPath("/manage"), bunrouter.HTTPHandlerFunc(manage.CallManageFunction))
	router.GET(config.WebPath("/manage/:action"), bunrouter.HTTPHandlerFunc(manage.CallManageFunction))
	router.POST(config.WebPath("/manage/:action"), bunrouter.HTTPHandlerFunc(manage.CallManageFunction))
	router.GET(config.WebPath("/post"), bunrouter.HTTPHandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, config.WebPath("/"), http.StatusFound)
	}))
	router.POST(config.WebPath("/post"), bunrouter.HTTPHandlerFunc(posting.MakePost))
	router.GET(config.WebPath("/util"), bunrouter.HTTPHandlerFunc(utilHandler))
	router.POST(config.WebPath("/util"), bunrouter.HTTPHandlerFunc(utilHandler))
	router.GET(config.WebPath("/util/banner"), bunrouter.HTTPHandlerFunc(randomBanner))
	// Eventually plugins might be able to register new namespaces or they might be restricted to something
	// like /plugin

	if systemCritical.UseFastCGI {
		err = fcgi.Serve(listener, router)
	} else {
		err = http.Serve(listener, router)
	}

	if err != nil {
		if !systemCritical.DebugMode {
			fmt.Println("Error initializing server:", err.Error())
		}
		gcutil.Logger().Fatal().
			Err(err).
			Msg("Error initializing server")
	}
}

func randomBanner(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	boardDir := request.FormValue("board")
	boardCfg := config.GetBoardConfig(boardDir)
	var banner *config.PageBanner
	if len(boardCfg.Banners) > 1 {
		banner = &boardCfg.Banners[rand.Intn(len(boardCfg.Banners)-1)]
	} else if len(boardCfg.Banners) == 1 {
		banner = &boardCfg.Banners[0]
	}
	err := json.NewEncoder(writer).Encode(banner)
	if err != nil {
		gcutil.LogError(err).Caller().Str("board", boardDir).Send()
		server.ServeError(writer, err.Error(), true, map[string]any{
			"board":  boardDir,
			"banner": banner,
		})
		return
	}
	if banner != nil {
		gcutil.LogAccess(request).Str("board", boardDir).Str("banner", banner.Filename).Send()
	}
}

// handles requests to /util
func utilHandler(writer http.ResponseWriter, request *http.Request) {
	action := request.FormValue("action")
	board := request.FormValue("board")
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	editBtn := request.PostFormValue("edit_btn")
	doEdit := request.PostFormValue("doedit")
	moveBtn := request.PostFormValue("move_btn")
	doMove := request.PostFormValue("domove")
	systemCritical := config.GetSystemCriticalConfig()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if wantsJSON {
		writer.Header().Set("Content-Type", "application/json")
	}
	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit post" && doEdit != "post" && doEdit != "upload" && moveBtn != "Move thread" && doMove != "1" {
		gcutil.LogAccess(request).
			Int("status", http.StatusBadRequest).
			Msg("received invalid /util request")
		if wantsJSON {
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeJSON(writer, map[string]any{"error": "Invalid /util request"})
		} else {
			http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "/"), http.StatusBadRequest)
		}
		return
	}

	var err error
	var id int
	var checkedPosts []int
	for key, val := range request.Form {
		// get checked posts into an array
		if _, err = fmt.Sscanf(key, "check%d", &id); err != nil || val[0] != "on" {
			err = nil
			continue
		}
		checkedPosts = append(checkedPosts, id)
	}

	if reportBtn == "Report" {
		// submitted request appears to be a report
		if err = posting.HandleReport(request); err != nil {
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Ints("posts", checkedPosts).
				Str("board", board).
				Msg("Error submitting report")
			server.ServeError(writer, err.Error(), wantsJSON, map[string]any{
				"posts": checkedPosts,
				"board": board,
			})
			return
		}
		gcutil.LogWarning().
			Ints("reportedPosts", checkedPosts).
			Str("board", board).
			Str("IP", gcutil.GetRealIP(request)).Send()

		redirectTo := request.Referer()
		if redirectTo == "" {
			// request doesn't have a referer for some reason, redirect to board
			redirectTo = config.WebPath(board)
		}
		http.Redirect(writer, request, redirectTo, http.StatusFound)
		return
	}

	if editBtn != "" || doEdit == "post" || doEdit == "upload" {
		editPost(checkedPosts, editBtn, doEdit, writer, request)
		return
	}

	if moveBtn != "" || doMove == "1" {
		moveThread(checkedPosts, moveBtn, doMove, writer, request)
		return
	}

	if deleteBtn == "Delete" {
		deletePosts(checkedPosts, writer, request)
		return
	}
}
