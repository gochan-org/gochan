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
	cookies       []*http.Cookie
	writer        http.ResponseWriter
	request       http.Request
	server        *GochanServer
	referrerRegex *regexp.Regexp
)

type GochanServer struct {
	/* writer     http.ResponseWriter
	request    http.Request */
	namespaces map[string]func(http.ResponseWriter, *http.Request, interface{})
}

func (s GochanServer) AddNamespace(basePath string, namespaceFunction func(http.ResponseWriter, *http.Request, interface{})) {
	s.namespaces[basePath] = namespaceFunction
}

func (s GochanServer) getFileData(writer http.ResponseWriter, url string) (fileBytes []byte) {
	filePath := path.Join(config.DocumentRoot, url)
	results, err := os.Stat(filePath)
	if err != nil {
		// the requested path isn't a file or directory, 404
		fileBytes = nil
	} else {
		//the file exists, or there is a folder here
		if results.IsDir() {
			//check to see if one of the specified index pages exists
			for _, value := range config.FirstPage {
				newPath := path.Join(filePath, value)
				_, err := os.Stat(newPath)
				if err == nil {
					// serve the index page
					writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
					fileBytes, _ := ioutil.ReadFile(newPath)
					return fileBytes
				}
			}
		} else {
			//the file exists, and is not a folder
			fileBytes, _ = ioutil.ReadFile(filePath)
			extension := getFileExtension(url)
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
			accessLog.Print("Success: 200 from " + getRealIP(&request) + " @ " + request.RequestURI)
		}
	}
	return
}

func serveNotFound(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(404)
	errorPage, err := ioutil.ReadFile(config.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and 404 error page not found"))
	} else {
		writer.Write(errorPage)

	}
	errorLog.Print("Error: 404 Not Found from " + getRealIP(request) + " @ " + request.RequestURI)
}

func serveErrorPage(writer http.ResponseWriter, err string) {
	errorPageBytes, _ := ioutil.ReadFile("templates/error.html")
	errorPage := strings.Replace(string(errorPageBytes), "{ERRORTEXT}", err, -1)
	_, _ = writer.Write([]byte(errorPage))
}

func (s GochanServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	for name, namespaceFunction := range s.namespaces {
		if request.URL.Path == "/"+name {
			namespaceFunction(writer, request, nil)
			return
		}
	}
	fb := s.getFileData(writer, request.URL.Path)
	writer.Header().Add("Cache-Control", "max-age=86400")
	if fb == nil {
		serveNotFound(writer, request)
		return
	}
	writer.Write(fb)
}

func initServer() {
	listener, err := net.Listen("tcp", config.ListenIP+":"+strconv.Itoa(config.Port))
	if err != nil {
		handleError(0, "Failed listening on %s:%d: %s", config.ListenIP, config.Port, customError(err))
		os.Exit(2)
	}
	server = new(GochanServer)
	server.namespaces = make(map[string]func(http.ResponseWriter, *http.Request, interface{}))

	// Check if Akismet API key is usable at startup.
	if config.AkismetAPIKey != "" {
		checkAkismetAPIKey()
	}

	// Compile regex for checking referrers.
	referrerRegex = regexp.MustCompile(config.DomainRegex)

	testfunc := func(writer http.ResponseWriter, response *http.Request, data interface{}) {
		if writer != nil {
			writer.Write([]byte("hahahaha"))
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

func getRealIP(r *http.Request) string {
	// HTTP_CF_CONNECTING_IP > X-Forwarded-For > RemoteAddr
	if r.Header.Get("HTTP_CF_CONNECTING_IP") != "" {
		return r.Header.Get("HTTP_CF_CONNECTING_IP")
	}
	if r.Header.Get("X-Forwarded-For") != "" {
		return r.Header.Get("X-Forwarded-For")
	}
	return r.RemoteAddr
}

func validReferrer(request http.Request) bool {
	return referrerRegex.MatchString(request.Referer())
}

// register /util handler
func utilHandler(writer http.ResponseWriter, request *http.Request, data interface{}) {
	//writer.Header().Add("Content-Type", "text/css")
	action := request.FormValue("action")
	password := request.FormValue("password")
	board := request.FormValue("board")
	boardid := request.FormValue("boardid")
	fileOnly := request.FormValue("fileonly") == "on"
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	editBtn := request.PostFormValue("edit_btn")

	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" && editBtn != "Edit" {
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
		if len(postsArr) == 0 {
			serveErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(postsArr) > 1 {
			serveErrorPage(writer, "You can only edit one post at a time.")
			return
		} else {
			passwordMD5 := md5Sum(password)
			rank := getStaffRank()
			if passwordMD5 == "" && rank == 0 {
				serveErrorPage(writer, "Password required for post editing")
				return
			}
			var post PostTable
			post.ID, _ = strconv.Atoi(postsArr[0])
			post.BoardID, _ = strconv.Atoi(boardid)
			stmt, err := db.Prepare("SELECT `parentid`,` password`,`message_raw` FROM `" + config.DBprefix + "posts` WHERE `id` = ? AND `deleted_timestamp` = ?")
			if err != nil {
				serveErrorPage(writer, handleError(1, err.Error()+"\n"))
			}
			defer closeStatement(stmt)
			/* var post_edit_buffer bytes.Buffer
			if err = renderTemplate(post_edit_tmpl, "post_edit", post_edit_buffer,
				&Wrapper{IName: "boards_", Data: all_boards},
				&Wrapper{IName: "sections_w", Data: all_sections},
				&Wrapper{IName: "posts_w", Data: []interface{}{
					PostTable{BoardID: board.ID},
				}},
				&Wrapper{IName: "op", Data: []interface{}{PostTable{}}},
				&Wrapper{IName: "board", Data: []interface{}{board}},
			); err != nil {
				html += handleError(1, fmt.Sprintf("Failed building /%s/res/%d.html: %s", board.Dir, 0, err.Error())) + "<br />"
				return
			} */
		}
	}

	if deleteBtn == "Delete" {
		// Delete a post or thread
		passwordMD5 := md5Sum(password)
		rank := getStaffRank()

		if passwordMD5 == "" && rank == 0 {
			serveErrorPage(writer, "Password required for post deletion")
			return
		}

		for _, checkedPostID := range postsArr {
			var fileType string
			var thumbType string
			var post PostTable
			post.ID, _ = strconv.Atoi(checkedPostID)
			post.BoardID, _ = strconv.Atoi(boardid)

			stmt, err := db.Prepare("SELECT `parentid`, `filename`, `password` FROM `" + config.DBprefix + "posts` WHERE `id` = ? AND `boardid` = ? AND `deleted_timestamp` = ?")
			if err != nil {
				serveErrorPage(writer, handleError(1, err.Error()+"\n"))
			}
			defer closeStatement(stmt)

			err = stmt.QueryRow(&post.ID, &post.BoardID, nilTimestamp).Scan(&post.ParentID, &post.Filename, &post.Password)
			if err == sql.ErrNoRows {
				//the post has already been deleted
				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d has already been deleted or is a post in a deleted thread.\n<br />", post.ID)
				continue
			}
			if err != nil {
				serveErrorPage(writer, err.Error())
				return
			}

			err = db.QueryRow("SELECT `id` FROM `" + config.DBprefix + "boards` WHERE `dir` = '" + board + "'").Scan(&post.BoardID)
			if err != nil {
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

					_, err = db.Exec("UPDATE `" + config.DBprefix + "posts` SET `filename` = 'deleted' WHERE `id` = " + strconv.Itoa(post.ID) + " AND `boardid` = " + strconv.Itoa(post.BoardID))
					if err != nil {
						serveErrorPage(writer, err.Error())
						return
					}
				}
				_board, _ := getBoardArr(map[string]interface{}{"id": post.BoardID}, "")
				buildBoardPages(&_board[0])
				_post, _ := getPostArr(map[string]interface{}{"id": post.ID, "boardid": post.BoardID}, "")
				postBoard := _post[0]
				// postBoard := _post[0].(PostTable)
				buildThreadPages(&postBoard)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "Attached image from %d deleted successfully<br />\n<meta http-equiv=\"refresh\" content=\"1;url=/"+board+"/\">", post.ID)
			} else {
				// delete the post
				_, err = db.Exec("UPDATE `" + config.DBprefix + "posts` SET `deleted_timestamp` = '" + getSQLDateTime() + "' WHERE `id` = " + strconv.Itoa(post.ID))
				if post.ParentID == 0 {
					os.Remove(path.Join(config.DocumentRoot, board, "/res/"+strconv.Itoa(post.ID)+".html"))
				} else {
					_board, _ := getBoardArr(map[string]interface{}{"id": post.BoardID}, "") // getBoardArr("`id` = " + strconv.Itoa(boardid))
					buildBoardPages(&_board[0])
				}

				// if the deleted post is actually a thread, delete its posts
				_, err = db.Exec("UPDATE `" + config.DBprefix + "posts` SET `deleted_timestamp` = '" + getSQLDateTime() + "' WHERE `parentID` = " + strconv.Itoa(post.ID))
				if err != nil {
					serveErrorPage(writer, err.Error())
					return
				}

				// delete the file
				var deletedFilename string
				err = db.QueryRow("SELECT `filename` FROM `" + config.DBprefix + "posts` WHERE `id` = " + strconv.Itoa(post.ID) + " AND `filename` != ''").Scan(&deletedFilename)
				if err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				err = db.QueryRow("SELECT `filename` FROM `" + config.DBprefix + "posts` WHERE `parentID` = " + strconv.Itoa(post.ID) + " AND `filename` != ''").Scan(&deletedFilename)
				if err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				buildBoards(false, post.BoardID)

				writer.Header().Add("refresh", "4;url="+request.Referer())
				fmt.Fprintf(writer, "%d deleted successfully\n<br />", post.ID)
			}
		}
	}
}
