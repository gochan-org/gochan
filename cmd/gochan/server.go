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
	wantsJSON := request.PostFormValue("json") == "1"
	if wantsJSON {
		writer.Header().Set("Content-Type", "application/json")
	}

	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit" && doEdit != "1" {
		gclog.Printf(gclog.LAccessLog, "Received invalid /util request from %q", request.Host)
		if wantsJSON {
			serverutil.ServeJSON(writer, map[string]interface{}{"error": "Invalid /util request"})
		} else {
			http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "/"), http.StatusFound)
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
		err = posting.HandleReport(request)
		if wantsJSON {
			serverutil.ServeJSON(writer, map[string]interface{}{
				"error": err,
				"posts": checkedPosts,
				"board": board,
			})
			return
		}
		if err != nil {
			serverutil.ServeErrorPage(writer, gclog.Println(gclog.LErrorLog,
				"Error submitting report:", err.Error()))
			return
		}
		redirectTo := request.Referer()
		if redirectTo == "" {
			// request doesn't have a referer for some reason, redirect to board
			redirectTo = path.Join(systemCritical.WebRoot, board)
		}
		http.Redirect(writer, request, redirectTo, http.StatusFound)
		return
	}

	if editBtn == "Edit" {
		var err error
		if len(checkedPosts) == 0 {
			serverutil.ServeErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(checkedPosts) > 1 {
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
			post, err = gcsql.GetSpecificPost(checkedPosts[0], true)
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
				gclog.Print(gclog.LErrorLog,
					"Error executing edit post template: ", err.Error())
				if wantsJSON {
					serverutil.ServeJSON(writer, map[string]interface{}{
						"error": "Error executing edit post template",
					})
				} else {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Error executing edit post template: ", err.Error()))
				}
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

		if password == "" && rank == 0 {
			serverutil.ServeErrorPage(writer, "Password required for post deletion")
			return
		}

		for _, checkedPostID := range checkedPosts {
			var post gcsql.Post
			var err error
			post.ID = checkedPostID
			post.BoardID, err = strconv.Atoi(boardid)
			if err != nil {
				gclog.Printf(gclog.LErrorLog, "Invalid board ID in deletion request")
				if wantsJSON {
					serverutil.ServeJSON(writer, map[string]interface{}{
						"error":   "invalid boardid string",
						"boardid": boardid,
					})
				} else {
					serverutil.ServeErrorPage(writer,
						fmt.Sprintf("Invalid boardid '%s' in request (got error '%s')", boardid, err))
				}
				return
			}

			post, err = gcsql.GetSpecificPost(post.ID, true)
			if err == sql.ErrNoRows {
				if wantsJSON {
					serverutil.ServeJSON(writer, map[string]interface{}{
						"error":   "Post does not exist",
						"postid":  post.ID,
						"boardid": post.BoardID,
					})
					return
				} else {
					serverutil.ServeErrorPage(writer, fmt.Sprintf(
						"Post #%d has already been deleted or is a post in a deleted thread", post.ID))
					return
				}
			} else if err != nil {
				if wantsJSON {
					serverutil.ServeJSON(writer, map[string]interface{}{
						"error":   err,
						"postid":  post.ID,
						"boardid": post.BoardID,
					})
					return
				} else {
					serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
						"Error deleting post: ", err.Error()))
					return
				}
			}

			if passwordMD5 != post.Password && rank == 0 {
				if wantsJSON {
					serverutil.ServeJSON(writer, map[string]interface{}{
						"error":   "incorrect password",
						"postid":  post.ID,
						"boardid": post.BoardID,
					})
				} else {
					serverutil.ServeErrorPage(writer,
						fmt.Sprintf("Incorrect password for #%d", post.ID))
				}
				return
			}

			if fileOnly {
				fileName := post.Filename
				if fileName != "" && fileName != "deleted" {
					if err = gcsql.DeleteFilesFromPost(post.ID); err != nil {
						if wantsJSON {
							serverutil.ServeJSON(writer, map[string]interface{}{
								"error":  err,
								"postid": post.ID,
							})
						} else {
							serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
								"Error deleting files from post: ", err.Error()))
						}
					}
				}
				_board, _ := gcsql.GetBoardFromID(post.BoardID)
				building.BuildBoardPages(&_board)
				postBoard, _ := gcsql.GetSpecificPost(post.ID, true)
				building.BuildThreadPages(&postBoard)
			} else {
				// delete the post
				if err = gcsql.DeletePost(post.ID, true); err != nil {
					if wantsJSON {
						serverutil.ServeJSON(writer, map[string]interface{}{
							"error":  err,
							"postid": post.ID,
						})
					} else {
						serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
							"Error deleting post: ", err.Error()))
					}
				}
				if post.ParentID == 0 {
					os.Remove(path.Join(
						systemCritical.DocumentRoot, board, "/res/"+strconv.Itoa(post.ID)+".html"))
				} else {
					_board, _ := gcsql.GetBoardFromID(post.BoardID)
					building.BuildBoardPages(&_board)
				}
				building.BuildBoards(false, post.BoardID)
			}
			gclog.Printf(gclog.LAccessLog,
				"Post #%d on boardid %d deleted by %s, file only: %t",
				post.ID, post.BoardID, post.IP, fileOnly)
			if !wantsJSON {
				http.Redirect(writer, request, request.Referer(), http.StatusFound)
			}
		}
	}
}
