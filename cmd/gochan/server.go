package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
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
	var extension string
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
	} else {
		//the file exists, and is not a folder
		extension = strings.ToLower(gcutil.GetFileExtension(request.URL.Path))
		switch extension {
		case "png":
			writer.Header().Add("Content-Type", "image/png")
			writer.Header().Add("Cache-Control", "max-age=86400")
		case "gif":
			writer.Header().Add("Content-Type", "image/gif")
			writer.Header().Add("Cache-Control", "max-age=86400")
		case "jpg":
			fallthrough
		case "jpeg":
			writer.Header().Add("Content-Type", "image/jpeg")
			writer.Header().Add("Cache-Control", "max-age=86400")
		case "css":
			writer.Header().Add("Content-Type", "text/css")
			writer.Header().Add("Cache-Control", "max-age=43200")
		case "js":
			writer.Header().Add("Content-Type", "text/javascript")
			writer.Header().Add("Cache-Control", "max-age=43200")
		case "json":
			writer.Header().Add("Content-Type", "application/json")
			writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
		case "webm":
			writer.Header().Add("Content-Type", "video/webm")
			writer.Header().Add("Cache-Control", "max-age=86400")
		case "htm":
			fallthrough
		case "html":
			writer.Header().Add("Content-Type", "text/html")
			writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
		}
		gclog.Printf(gclog.LAccessLog, "Success: 200 from %s @ %s", gcutil.GetRealIP(request), request.URL.Path)
	}

	// serve the index page
	writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	fileBytes, _ = ioutil.ReadFile(filePath)
	writer.Header().Add("Cache-Control", "max-age=86400")
	writer.Write(fileBytes)
}

func (s gochanServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	systemCritical := config.GetSystemCriticalConfig()
	for name, namespaceFunction := range s.namespaces {
		if request.URL.Path == systemCritical.WebRoot+name {
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
		gclog.Printf(gclog.LErrorLog|gclog.LStdLog|gclog.LFatal,
			"Failed listening on %s:%d: %s", systemCritical.ListenIP, systemCritical.Port, err.Error())
	}
	server = new(gochanServer)
	server.namespaces = make(map[string]func(http.ResponseWriter, *http.Request))

	// Check if Akismet API key is usable at startup.
	err = serverutil.CheckAkismetAPIKey(siteConfig.AkismetAPIKey)
	if err == serverutil.ErrBlankAkismetKey {
		gclog.Print(gclog.LStdLog, err.Error(), ". Akismet spam protection won't be used.")
	} else if err != nil {
		gclog.Print(gclog.LErrorLog|gclog.LStdLog, ". Akismet spam protection will be disabled.")
		siteConfig.AkismetAPIKey = ""
	}

	server.namespaces["banned"] = posting.BanHandler
	server.namespaces["captcha"] = posting.ServeCaptcha
	server.namespaces["manage"] = manage.CallManageFunction
	server.namespaces["post"] = posting.MakePost
	server.namespaces["util"] = utilHandler
	server.namespaces["example"] = func(writer http.ResponseWriter, request *http.Request) {
		if writer != nil {
			http.Redirect(writer, request, "https://www.youtube.com/watch?v=dQw4w9WgXcQ", http.StatusFound)
		}
	}
	// Eventually plugins will be able to register new namespaces or they will be restricted to something
	// like /plugin

	if systemCritical.UseFastCGI {
		err = fcgi.Serve(listener, server)
	} else {
		err = http.Serve(listener, server)
	}

	if err != nil {
		gclog.Print(gclog.LErrorLog|gclog.LStdLog|gclog.LFatal,
			"Error initializing server: ", err.Error())
	}
}

// handles requests to /util
func utilHandler(writer http.ResponseWriter, request *http.Request) {
	action := request.FormValue("action")
	password := request.FormValue("password")
	board := request.FormValue("board")
	boardid := request.FormValue("boardid")
	fileOnly := request.FormValue("fileonly") == "on"
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	editBtn := request.PostFormValue("edit_btn")
	doEdit := request.PostFormValue("doedit")
	systemCritical := config.GetSystemCriticalConfig()

	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit" && doEdit != "1" {
		gclog.Printf(gclog.LAccessLog, "Received invalid /util request from %q", request.Host)
		http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "/"), http.StatusFound)
		return
	}
	var postsArr []string
	for key := range request.PostForm {
		if strings.Index(key, "check") == 0 {
			postsArr = append(postsArr, key[5:])
		}
	}
	var err error
	if reportBtn == "Report" {
		// submitted request appears to be a report
		if err = posting.HandleReport(request); err != nil {
			gclog.Printf(gclog.LErrorLog|gclog.LStdLog, "Error from HandleReport: %v\n", err)
		}
		return
	}

	if editBtn == "Edit" {
		var err error
		if len(postsArr) == 0 {
			serverutil.ServeErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(postsArr) > 1 {
			serverutil.ServeErrorPage(writer, "You can only edit one post at a time.")
			return
		} else {
			rank := manage.GetStaffRank(request)
			if password == "" && rank == 0 {
				serverutil.ServeErrorPage(writer, "Password required for post editing")
				return
			}
			passwordMD5 := gcutil.Md5Sum(password)

			var post gcsql.Post
			postid, _ := strconv.Atoi(postsArr[0])
			post, err = gcsql.GetSpecificPost(postid, true)
			if err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Error getting post information: ", err.Error()))
				return
			}

			if post.Password != passwordMD5 && rank == 0 {
				serverutil.ServeErrorPage(writer, "Wrong password")
				return
			}

			if err = gctemplates.PostEdit.Execute(writer, map[string]interface{}{
				"systemCritical": config.GetSystemCriticalConfig(),
				"siteConfig":     config.GetSiteConfig(),
				"boardConfig":    config.GetBoardConfig(""),
				"post":           post,
				"referrer":       request.Referer(),
			}); err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Error executing edit post template: ", err.Error()))
				return
			}
		}
	}
	if doEdit == "1" {
		var password string
		postid, err := strconv.Atoi(request.FormValue("postid"))
		if err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Invalid form data: ", err.Error()))
			return
		}
		boardid, err := strconv.Atoi(request.FormValue("boardid"))
		if err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Invalid form data: ", err.Error()))
			return
		}
		password, err = gcsql.GetPostPassword(postid)
		if err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Invalid form data: ", err.Error()))
			return
		}

		rank := manage.GetStaffRank(request)
		if request.FormValue("password") != password && rank == 0 {
			serverutil.ServeErrorPage(writer, "Wrong password")
			return
		}

		var board gcsql.Board
		if err = board.PopulateData(boardid); err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Invalid form data: ", err.Error()))
			return
		}

		if err = gcsql.UpdatePost(postid, request.FormValue("editemail"), request.FormValue("editsubject"),
			posting.FormatMessage(request.FormValue("editmsg")), request.FormValue("editmsg")); err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
				"Unable to edit post: ", err.Error()))
			return
		}

		building.BuildBoards(false, boardid)
		if request.FormValue("parentid") == "0" {
			http.Redirect(writer, request, "/"+board.Dir+"/res/"+strconv.Itoa(postid)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, "/"+board.Dir+"/res/"+request.FormValue("parentid")+".html#"+strconv.Itoa(postid), http.StatusFound)
		}

		return
	}

	if deleteBtn == "Delete" {
		// Delete a post or thread
		writer.Header().Add("Content-Type", "text/plain")
		passwordMD5 := gcutil.Md5Sum(password)
		rank := manage.GetStaffRank(request)

		if passwordMD5 == "" && rank == 0 {
			serverutil.ServeErrorPage(writer, "Password required for post deletion")
			return
		}

		for _, checkedPostID := range postsArr {
			var post gcsql.Post
			var err error
			post.ID, _ = strconv.Atoi(checkedPostID)
			post.BoardID, _ = strconv.Atoi(boardid)

			if post, err = gcsql.GetSpecificPost(post.ID, true); err == sql.ErrNoRows {
				//the post has already been deleted
				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d has already been deleted or is a post in a deleted thread.\n", post.ID)
				continue
			} else if err != nil {
				serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
					"Error deleting post: ", err.Error()))
				return
			}

			if passwordMD5 != post.Password && rank == 0 {
				fmt.Fprintf(writer, "Incorrect password for %d\n", post.ID)
				continue
			}

			if fileOnly {
				fileName := post.Filename
				if fileName != "" && fileName != "deleted" {
					if err = gcsql.DeleteFilesFromPost(post.ID); err != nil {
						serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
							"Error deleting files from post: ", err.Error()))
						return
					}
				}
				_board, _ := gcsql.GetBoardFromID(post.BoardID)
				building.BuildBoardPages(&_board)
				postBoard, _ := gcsql.GetSpecificPost(post.ID, true)
				building.BuildThreadPages(&postBoard)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "Attached image from %d deleted successfully\n", post.ID)
			} else {
				// delete the post
				if err = gcsql.DeletePost(post.ID, true); err != nil {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Error deleting post: ", err.Error()))
				}
				if post.ParentID == 0 {
					os.Remove(path.Join(
						systemCritical.DocumentRoot, board, "/res/"+strconv.Itoa(post.ID)+".html"))
				} else {
					_board, _ := gcsql.GetBoardFromID(post.BoardID)
					building.BuildBoardPages(&_board)
				}
				building.BuildBoards(false, post.BoardID)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d deleted successfully\n", post.ID)
			}
		}
	}
}
