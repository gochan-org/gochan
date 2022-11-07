package main

import (
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

var (
	server *gochanServer
)

type gochanServer struct {
	namespaces map[string]func(http.ResponseWriter, *http.Request)
}

func (s gochanServer) serveFile(writer http.ResponseWriter, request *http.Request) {
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()

	filePath := path.Join(systemCritical.DocumentRoot, request.URL.Path)
	var fileBytes []byte
	results, err := os.Stat(filePath)
	if err != nil {
		// the requested path isn't a file or directory, 404
		serverutil.ServeNotFound(writer, request)
		return
	}

	//the file exists, or there is a folder here
	if results.IsDir() {
		//check to see if one of the specified index pages exists
		var found bool
		for _, value := range siteConfig.FirstPage {
			newPath := path.Join(filePath, value)
			_, err := os.Stat(newPath)
			if err == nil {
				filePath = newPath
				found = true
				break
			}
		}
		if !found {
			serverutil.ServeNotFound(writer, request)
			return
		}
	}
	s.setFileHeaders(filePath, writer)

	// serve the requested file
	fileBytes, _ = os.ReadFile(filePath)
	gcutil.LogAccess(request).Int("status", 200).Send()
	writer.Write(fileBytes)
}

// set mime type/cache headers according to the file's extension
func (*gochanServer) setFileHeaders(filename string, writer http.ResponseWriter) {
	extension := strings.ToLower(path.Ext(filename))
	switch extension {
	case ".png":
		writer.Header().Set("Content-Type", "image/png")
		writer.Header().Set("Cache-Control", "max-age=86400")
	case ".gif":
		writer.Header().Set("Content-Type", "image/gif")
		writer.Header().Set("Cache-Control", "max-age=86400")
	case ".jpg":
		fallthrough
	case ".jpeg":
		writer.Header().Set("Content-Type", "image/jpeg")
		writer.Header().Set("Cache-Control", "max-age=86400")
	case ".css":
		writer.Header().Set("Content-Type", "text/css")
		writer.Header().Set("Cache-Control", "max-age=43200")
	case ".js":
		writer.Header().Set("Content-Type", "text/javascript")
		writer.Header().Set("Cache-Control", "max-age=43200")
	case ".json":
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("Cache-Control", "max-age=5, must-revalidate")
	case ".webm":
		writer.Header().Set("Content-Type", "video/webm")
		writer.Header().Set("Cache-Control", "max-age=86400")
	case ".htm":
		fallthrough
	case ".html":
		writer.Header().Set("Content-Type", "text/html")
		writer.Header().Set("Cache-Control", "max-age=5, must-revalidate")
	default:
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.Header().Set("Cache-Control", "max-age=86400")
	}
}

func (s gochanServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	for name, namespaceFunction := range s.namespaces {
		if request.URL.Path == config.WebPath(name) {
			namespaceFunction(writer, request)
			return
		}
	}
	s.serveFile(writer, request)
}

func initServer() {
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()

	listener, err := net.Listen("tcp", systemCritical.ListenIP+":"+strconv.Itoa(systemCritical.Port))
	if err != nil {
		gcutil.Logger().Fatal().
			Err(err).
			Str("ListenIP", systemCritical.ListenIP).
			Int("Port", systemCritical.Port).Send()
		fmt.Printf("Failed listening on %s:%d: %s", systemCritical.ListenIP, systemCritical.Port, err.Error())
	}
	server = new(gochanServer)
	server.namespaces = make(map[string]func(http.ResponseWriter, *http.Request))

	// Check if Akismet API key is usable at startup.
	err = serverutil.CheckAkismetAPIKey(siteConfig.AkismetAPIKey)
	if err != nil && err != serverutil.ErrBlankAkismetKey {
		gcutil.Logger().Err(err).
			Msg("Akismet spam protection will be disabled")
		fmt.Println("Got error when initializing Akismet spam protection, it will be disabled:", err)
	}

	// server.namespaces["banned"] = posting.BanHandler
	server.namespaces["captcha"] = posting.ServeCaptcha
	server.namespaces["manage"] = manage.CallManageFunction
	server.namespaces["post"] = posting.MakePost
	server.namespaces["util"] = utilHandler
	server.namespaces["example"] = func(writer http.ResponseWriter, request *http.Request) {
		if writer != nil {
			http.Redirect(writer, request, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", http.StatusFound)
		}
	}
	// Eventually plugins might be able to register new namespaces or they might be restricted to something
	// like /plugin

	if systemCritical.UseFastCGI {
		err = fcgi.Serve(listener, server)
	} else {
		err = http.Serve(listener, server)
	}

	if err != nil {
		gcutil.Logger().Fatal().
			Err(err).
			Msg("Error initializing server")
		fmt.Println("Error initializing server:", err.Error())
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
	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit post" && doEdit != "1" && moveBtn != "Move thread" && doMove != "1" {
		gcutil.LogAccess(request).Int("status", 400).Msg("received invalid /util request")
		if wantsJSON {
			writer.WriteHeader(http.StatusBadRequest)
			serverutil.ServeJSON(writer, map[string]interface{}{"error": "Invalid /util request"})
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
			serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
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
			redirectTo = path.Join(systemCritical.WebRoot, board)
		}
		http.Redirect(writer, request, redirectTo, http.StatusFound)
		return
	}

	if editBtn != "" || doEdit == "1" {
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
