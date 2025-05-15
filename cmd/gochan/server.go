package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/fcgi"
	"strconv"
	"time"

	"github.com/uptrace/bunrouter"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func initServer() {
	var listener net.Listener
	var err error
	systemCritical := config.GetSystemCriticalConfig()
	listenAddr := net.JoinHostPort(systemCritical.ListenAddress, strconv.Itoa(systemCritical.Port))
	fatalEv := gcutil.LogFatal().
		Str("listenAddress", systemCritical.ListenAddress).
		Bool("useFastCGI", systemCritical.UseFastCGI).
		Int("port", systemCritical.Port)
	defer fatalEv.Discard()

	router := server.GetRouter()
	router.GET(config.WebPath("/captcha"), bunrouter.HTTPHandlerFunc(posting.ServeCaptcha))
	router.POST(config.WebPath("/captcha"), bunrouter.HTTPHandlerFunc(posting.ServeCaptcha))
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
		listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			fatalEv.Err(err).Caller().Msg("Failed listening on address/port")
		}
		err = fcgi.Serve(listener, router)
	} else {
		httpServer := &http.Server{
			Addr:              listenAddr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		}
		err = httpServer.ListenAndServe()
	}

	if err != nil {
		fatalEv.Err(err).Caller().Msg("Error initializing server")
	}
}

func randomBanner(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	banners := config.GetBoardConfig("").Banners // get global banners
	boardDir := request.FormValue("board")
	if boardDir != "" {
		banners = append(banners, config.GetBoardConfig(boardDir).Banners...)
	}
	var banner config.PageBanner
	if len(banners) > 0 {
		banner = banners[rand.Intn(len(banners))] // skipcq: GSC-G404
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
	if banner.Filename != "" {
		gcutil.LogAccess(request).Str("board", boardDir).Str("banner", banner.Filename).Send()
	}
}

// handles requests to /util
func utilHandler(writer http.ResponseWriter, request *http.Request) {
	board := request.FormValue("board")
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	editBtn := request.PostFormValue("edit_btn")
	doEdit := request.PostFormValue("doedit")
	moveBtn := request.PostFormValue("move_btn")
	doMove := request.PostFormValue("domove")
	wantsJSON := serverutil.IsRequestingJSON(request)
	redirectTo := request.Referer()
	if wantsJSON {
		writer.Header().Set("Content-Type", "application/json")
	}
	if redirectTo == "" || (deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit post" && doEdit != "post" && doEdit != "upload" && moveBtn != "Move thread" && doMove != "1") {
		accessEv := gcutil.LogAccess(request).
			Int("status", http.StatusBadRequest).
			Str("redirect", redirectTo)

		if redirectTo == "" {
			accessEv.Msg("received /util request with no referer")
		} else {
			accessEv.Msg("received invalid /util request")
		}
		accessEv.Discard()
		server.ServeError(writer, server.NewServerError("bad /util request", http.StatusBadRequest), wantsJSON, nil)
		return
	}

	var err error
	var id int
	var checkedPosts []int
	for key, val := range request.PostForm {
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
