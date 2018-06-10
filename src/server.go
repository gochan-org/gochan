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
	"regexp"
	"strconv"
	"strings"
)

var (
	server        *GochanServer
	referrerRegex *regexp.Regexp
)

type GochanServer struct {
	namespaces map[string]func(http.ResponseWriter, *http.Request)
}

func (s GochanServer) AddNamespace(basePath string, namespaceFunction func(http.ResponseWriter, *http.Request)) {
	s.namespaces[basePath] = namespaceFunction
}

func (s GochanServer) serveFile(writer http.ResponseWriter, request *http.Request) {
	filePath := path.Join(config.DocumentRoot, request.URL.Path)
	var fileBytes []byte
	results, err := os.Stat(filePath)
	if err != nil {
		// the requested path isn't a file or directory, 404
		writer.WriteHeader(404)
		serveNotFound(writer, request)
		return
	} else {
		//the file exists, or there is a folder here
		if results.IsDir() {
			//check to see if one of the specified index pages exists
			for _, value := range config.FirstPage {
				newPath := path.Join(filePath, value)
				_, err := os.Stat(newPath)
				if err == nil {
					filePath = newPath
					break
				}
			}
		} else {
			//the file exists, and is not a folder
			extension := getFileExtension(request.URL.Path)
			switch extension {
			case "png":
				writer.Header().Add("Content-Type", "image/png")
				writer.Header().Add("Cache-Control", "max-age=86400")
			case "gif":
				writer.Header().Add("Content-Type", "image/gif")
				writer.Header().Add("Cache-Control", "max-age=86400")
			case "jpg":
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
			}
			if strings.HasPrefix(extension, "htm") {
				writer.Header().Add("Content-Type", "text/html")
				writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
			}
			accessLog.Print("Success: 200 from " + getRealIP(request) + " @ " + request.URL.Path)
		}
	}
	// serve the index page
	writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	fileBytes, _ = ioutil.ReadFile(filePath)
	writer.Header().Add("Cache-Control", "max-age=86400")
	_, _ = writer.Write(fileBytes)
}

func serveNotFound(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(404)
	errorPage, err := ioutil.ReadFile(config.DocumentRoot + "/error/404.html")
	if err != nil {
		_, _ = writer.Write([]byte("Requested page not found, and 404 error page not found"))
	} else {
		_, _ = writer.Write(errorPage)
	}
	errorLog.Print("Error: 404 Not Found from " + getRealIP(request) + " @ " + request.URL.Path)
}

func serveErrorPage(writer http.ResponseWriter, err string) {
	errorpage_tmpl.Execute(writer, map[string]interface{}{
		"config":      config,
		"ErrorTitle":  "Error :c",
		"ErrorImage":  "/error/lol 404.gif",
		"ErrorHeader": "Error",
		"ErrorText":   err,
	})
}

func (s GochanServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	for name, namespaceFunction := range s.namespaces {
		if request.URL.Path == "/"+name {
			// writer.WriteHeader(200)
			namespaceFunction(writer, request)
			return
		}
	}
	s.serveFile(writer, request)
}

func initServer() {
	listener, err := net.Listen("tcp", config.ListenIP+":"+strconv.Itoa(config.Port))
	if err != nil {
		handleError(0, "Failed listening on %s:%d: %s", config.ListenIP, config.Port, customError(err))
		os.Exit(2)
	}
	server = new(GochanServer)
	server.namespaces = make(map[string]func(http.ResponseWriter, *http.Request))

	// Check if Akismet API key is usable at startup.
	if config.AkismetAPIKey != "" {
		checkAkismetAPIKey()
	}

	// Compile regex for checking referrers.
	referrerRegex = regexp.MustCompile(config.DomainRegex)

	testfunc := func(writer http.ResponseWriter, request *http.Request) {
		if writer != nil {
			_, _ = writer.Write([]byte("hahahaha"))
		}
	}

	server.AddNamespace("example", testfunc)
	server.AddNamespace("manage", callManageFunction)
	server.AddNamespace("post", makePost)
	server.AddNamespace("util", utilHandler)
	// eventually plugins will be able to register new namespaces. Or they will be restricted to something like /plugin

	if config.UseFastCGI {
		err = fcgi.Serve(listener, server)
	} else {
		err = http.Serve(listener, server)
	}

	if err != nil {
		handleError(0, customError(err))
		os.Exit(2)
	}
}

func getRealIP(request *http.Request) string {
	// HTTP_CF_CONNECTING_IP > X-Forwarded-For > RemoteAddr
	if request.Header.Get("HTTP_CF_CONNECTING_IP") != "" {
		return request.Header.Get("HTTP_CF_CONNECTING_IP")
	}
	if request.Header.Get("X-Forwarded-For") != "" {
		return request.Header.Get("X-Forwarded-For")
	}
	return request.RemoteAddr
}

func validReferrer(request *http.Request) bool {
	return referrerRegex.MatchString(request.Referer())
}

// register /util handler
func utilHandler(writer http.ResponseWriter, request *http.Request) {
	//writer.Header().Add("Content-Type", "text/css")
	action := request.FormValue("action")
	password := request.FormValue("password")
	board := request.FormValue("board")
	boardid := request.FormValue("boardid")
	fileOnly := request.FormValue("fileonly") == "on"
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	editBtn := request.PostFormValue("edit_btn")
	doEdit := request.PostFormValue("doedit")

	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit" && doEdit != "1" {
		http.Redirect(writer, request, path.Join(config.SiteWebfolder, "/"), http.StatusFound)
		return
	}
	var postsArr []string
	for key := range request.PostForm {
		if strings.Index(key, "check") == 0 {
			postsArr = append(postsArr, key[5:])
		}
	}

	if editBtn == "Edit" {
		var err error
		if len(postsArr) == 0 {
			serveErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(postsArr) > 1 {
			serveErrorPage(writer, "You can only edit one post at a time.")
			return
		} else {
			rank := getStaffRank(request)
			if password == "" && rank == 0 {
				serveErrorPage(writer, "Password required for post editing")
				return
			}
			passwordMD5 := md5Sum(password)

			var post PostTable
			post.ID, _ = strconv.Atoi(postsArr[0])
			post.BoardID, _ = strconv.Atoi(boardid)
			if err = queryRowSQL("SELECT `parentid`,`name`,`tripcode`,`email`,`subject`,`password`,`message_raw` FROM `"+config.DBprefix+"posts` WHERE `id` = ? AND `boardid` = ? AND `deleted_timestamp` = ?",
				[]interface{}{post.ID, post.BoardID, nilTimestamp},
				[]interface{}{
					&post.ParentID, &post.Name, &post.Tripcode, &post.Email, &post.Subject,
					&post.Password, &post.MessageText},
			); err != nil {
				serveErrorPage(writer, handleError(0, err.Error()))
				return
			}

			if post.Password != passwordMD5 && rank == 0 {
				serveErrorPage(writer, "Wrong password")
				return
			}

			if err = post_edit_tmpl.Execute(writer, map[string]interface{}{
				"config":   config,
				"post":     post,
				"referrer": request.Referer(),
			}); err != nil {
				serveErrorPage(writer, handleError(0, err.Error()))
				return
			}

		}
	}
	if doEdit == "1" {
		var postPassword string

		postid, err := strconv.Atoi(request.FormValue("postid"))
		if err != nil {
			serveErrorPage(writer, handleError(0, "Invalid form data: %s", err.Error()))
			return
		}
		boardid, err := strconv.Atoi(request.FormValue("boardid"))
		if err != nil {
			serveErrorPage(writer, handleError(0, "Invalid form data: %s", err.Error()))
			return
		}

		if err = queryRowSQL("SELECT `password` FROM `"+config.DBprefix+"posts` WHERE `id` = ? AND `boardid` = ?",
			[]interface{}{postid, boardid},
			[]interface{}{&postPassword},
		); err != nil {
			serveErrorPage(writer, handleError(0, "Invalid form data: %s", err.Error()))
		}

		rank := getStaffRank(request)
		if request.FormValue("password") != password && rank == 0 {
			serveErrorPage(writer, "Wrong password")
			return
		}

		board, err := getBoardFromID(boardid)
		if err != nil {
			serveErrorPage(writer, handleError(0, "Invalid form data: %s", err.Error()))
			return
		}

		if _, err = execSQL("UPDATE `"+config.DBprefix+"posts` SET "+
			"`email` = ?, `subject` = ?, `message` = ?, `message_raw` = ? WHERE `id` = ? AND `boardid` = ?",
			request.FormValue("editemail"), request.FormValue("editsubject"), formatMessage(request.FormValue("editmsg")), request.FormValue("editmsg"),
			postid, boardid,
		); err != nil {
			serveErrorPage(writer, handleError(0, "editing post: %s", err.Error()))
			return
		}

		buildBoards(false, boardid)
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
		passwordMD5 := md5Sum(password)
		rank := getStaffRank(request)

		if passwordMD5 == "" && rank == 0 {
			serveErrorPage(writer, "Password required for post deletion")
			return
		}

		for _, checkedPostID := range postsArr {
			var fileType string
			var thumbType string
			var post PostTable
			var err error
			post.ID, _ = strconv.Atoi(checkedPostID)
			post.BoardID, _ = strconv.Atoi(boardid)

			if err = queryRowSQL(
				"SELECT `parentid`, `filename`, `password` FROM `"+config.DBprefix+"posts` WHERE `id` = ? AND `boardid` = ? AND `deleted_timestamp` = ?",
				[]interface{}{&post.ID, &post.BoardID, nilTimestamp},
				[]interface{}{&post.ParentID, &post.Filename, &post.Password},
			); err == sql.ErrNoRows {
				//the post has already been deleted
				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d has already been deleted or is a post in a deleted thread.\n", post.ID)
				continue
			} else if err != nil {
				serveErrorPage(writer, handleError(1, err.Error()+"\n"))
				return
			}

			if err = queryRowSQL(
				"SELECT `id` FROM `"+config.DBprefix+"boards` WHERE `dir` = ?",
				[]interface{}{board},
				[]interface{}{&post.BoardID},
			); err != nil {
				serveErrorPage(writer, err.Error())
				return
			}

			if passwordMD5 != post.Password && rank == 0 {
				fmt.Fprintf(writer, "Incorrect password for %d\n", post.ID)
				continue
			}

			if fileOnly {
				fileName := post.Filename
				if fileName != "" && fileName != "deleted" {
					fileName = fileName[:strings.Index(fileName, ".")]
					fileType = fileName[strings.Index(fileName, ".")+1:]
					if fileType == "gif" || fileType == "webm" {
						thumbType = "jpg"
					}

					os.Remove(path.Join(config.DocumentRoot, board, "/src/"+fileName+"."+fileType))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/"+fileName+"t."+thumbType))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/"+fileName+"c."+thumbType))

					if _, err = execSQL(
						"UPDATE `"+config.DBprefix+"posts` SET `filename` = 'deleted' WHERE `id` = ? AND `boardid` = ?",
						post.ID, post.BoardID,
					); err != nil {
						serveErrorPage(writer, err.Error())
						return
					}
				}
				_board, _ := getBoardArr(map[string]interface{}{"id": post.BoardID}, "")
				buildBoardPages(&_board[0])
				_post, _ := getPostArr(map[string]interface{}{"id": post.ID, "boardid": post.BoardID}, "")
				postBoard := _post[0]
				buildThreadPages(&postBoard)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "Attached image from %d deleted successfully\n", post.ID) //<br />\n<meta http-equiv=\"refresh\" content=\"1;url=/"+board+"/\">", post.ID)
			} else {
				// delete the post
				if _, err = execSQL(
					"UPDATE `"+config.DBprefix+"posts` SET `deleted_timestamp` = ? WHERE `id` = ?",
					getSQLDateTime(), post.ID,
				); err != nil {
					serveErrorPage(writer, err.Error())
				}
				if post.ParentID == 0 {
					os.Remove(path.Join(config.DocumentRoot, board, "/res/"+strconv.Itoa(post.ID)+".html"))
				} else {
					_board, _ := getBoardArr(map[string]interface{}{"id": post.BoardID}, "") // getBoardArr("`id` = " + strconv.Itoa(boardid))
					buildBoardPages(&_board[0])
				}

				// if the deleted post is actually a thread, delete its posts
				if _, err = execSQL("UPDATE `"+config.DBprefix+"posts` SET `deleted_timestamp` = ? WHERE `parentID` = ?",
					getSQLDateTime(), post.ID,
				); err != nil {
					serveErrorPage(writer, err.Error())
					return
				}

				// delete the file
				var deletedFilename string
				if err = queryRowSQL(
					"SELECT `filename` FROM `"+config.DBprefix+"posts` WHERE `id` = ? AND `filename` != ''",
					[]interface{}{post.ID},
					[]interface{}{&deletedFilename},
				); err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				if err = queryRowSQL(
					"SELECT `filename` FROM `"+config.DBprefix+"posts` WHERE `parentID` = ? AND `filename` != ''",
					[]interface{}{post.ID},
					[]interface{}{&deletedFilename},
				); err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				buildBoards(false, post.BoardID)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d deleted successfully\n", post.ID)
			}
		}
	}
}
