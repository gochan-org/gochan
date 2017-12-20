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
	writer     http.ResponseWriter
	request    http.Request
	namespaces map[string]func(http.ResponseWriter, *http.Request, interface{})
}

func (s GochanServer) AddNamespace(basePath string, namespaceFunction func(http.ResponseWriter, *http.Request, interface{})) {
	s.namespaces[basePath] = namespaceFunction
}

func (s GochanServer) getFileData(writer http.ResponseWriter, url string) []byte {
	filePath := path.Join(config.DocumentRoot, url)
	results, err := os.Stat(filePath)
	if err != nil {
		// the requested path isn't a file or directory, 404
		return nil
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
			fileBytes, _ := ioutil.ReadFile(filePath)
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
			}
			if strings.HasPrefix(extension, "htm") {
				writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
			}

			access_log.Print("Success: 200 from " + getRealIP(&request) + " @ " + request.RequestURI)
			return fileBytes
		}
	}
	return nil
}

func serveNotFound(writer http.ResponseWriter, request *http.Request) {
	errorPage, err := ioutil.ReadFile(config.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and 404 error page not found"))
	} else {
		writer.Write(errorPage)
	}
	error_log.Print("Error: 404 Not Found from " + getRealIP(request) + " @ " + request.RequestURI)
}

func serveErrorPage(writer http.ResponseWriter, err string) {
	errorPageBytes, _ := ioutil.ReadFile("templates/error.html")
	errorPage := strings.Replace(string(errorPageBytes), "{ERRORTEXT}", err, -1)
	writer.Write([]byte(errorPage))
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
		fmt.Printf("Failed listening on %s:%d, see log for details", config.ListenIP, config.Port)
		error_log.Fatal(err.Error())
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
		fcgi.Serve(listener, server)
	} else {
		http.Serve(listener, server)
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
	writer.Header().Add("Content-Type", "text/css")
	action := request.FormValue("action")
	password := request.FormValue("password")
	board := request.FormValue("board")
	fileOnly := request.FormValue("fileonly") == "on"
	deleteBtn := request.PostFormValue("delete_btn")
	reportBtn := request.PostFormValue("report_btn")
	if action == "" && deleteBtn != "Delete" && reportBtn != "Report" {
		http.Redirect(writer, request, path.Join(config.SiteWebfolder, "/"), http.StatusFound)
		return
	}
	var postsArr []string
	for key := range request.PostForm {
		if strings.Index(key, "check") == 0 {
			postsArr = append(postsArr, key[5:])
		}
	}
	if reportBtn == "Delete" {
		// Delete a post or thread
		password = md5Sum(password)
		rank := getStaffRank()

		if password == "" && rank == 0 {
			serveErrorPage(writer, "Password required for post deletion")
			return
		}

		for _, post := range postsArr {
			var parentId int
			var fileName string
			var fileType string
			var passwordChecksum string
			var boardId int
			stmt, err := db.Prepare("SELECT `parentid, `filename`, `password` FROM " + config.DBprefix + "posts WHERE `id` = ? AND `deleted_timestamp`  = ?")
			defer func() {
				if stmt != nil {
					stmt.Close()
				}
			}()

			if err != nil {
				error_log.Print(err.Error())
				println(1, err.Error())
				serveErrorPage(writer, err.Error())
			}
			err = stmt.QueryRow(&post, nil_timestamp).Scan(&parentId, &fileName, &passwordChecksum)

			if err == sql.ErrNoRows {
				//the post has already been deleted
				writer.Header().Add("refresh", "3;url="+request.Referer())
				fmt.Fprintf(writer, "%s has already been deleted or is a post in a deleted thread.\n<br />", post)
				continue
			}
			if err != nil {
				serveErrorPage(writer, err.Error())
				return
			}

			err = db.QueryRow(`SELECT "id" FROM "` + config.DBprefix + `boards" WHERE "dir" = "` + board + `"`).Scan(&boardId)
			if err != nil {
				serveErrorPage(writer, err.Error())
				return
			}

			if password != passwordChecksum && rank == 0 {
				fmt.Fprintf(writer, "Incorrect password for %s\n", post)
				continue
			}

			if fileOnly {
				if fileName != "" && fileName != "deleted" {
					fileType = fileName[strings.Index(fileName, ".")+1:]
					fileName = fileName[:strings.Index(fileName, ".")]
					err := os.Remove(path.Join(config.DocumentRoot, board, "/src/"+fileName+"."+fileType))
					if err != nil {
						serveErrorPage(writer, err.Error())
						return
					}
					err = os.Remove(path.Join(config.DocumentRoot, board, "/thumb/"+fileName+"t."+fileType))
					if err != nil {
						serveErrorPage(writer, err.Error())
						return
					}
					_, err = db.Exec(`UPDATE "` + config.DBprefix + `posts" SET "filename" = "deleted" WHERE "id" = ` + post)
					if err != nil {
						serveErrorPage(writer, err.Error())
						return
					}
				}
				writer.Header().Add("refresh", "3;url="+request.Referer())
				fmt.Fprintf(writer, "Attached image from %s deleted successfully<br />\n<meta http-equiv=\"refresh\" content=\"1;url="+config.DocumentRoot+"/"+board+"/\">", post)
			} else {

				// delete the post
				_, err = db.Exec(`UPDATE "` + config.DBprefix + `posts" SET "deleted_timestamp" = "` + getSQLDateTime() + `" WHERE "id" = ` + post)
				if parentId == 0 {
					err = os.Remove(path.Join(config.DocumentRoot, board, "/res/"+post+".html"))
				} else {
					_board, _ := getBoardArr(map[string]interface{}{"id": boardId}, "") // getBoardArr("`id` = " + strconv.Itoa(boardId))
					buildBoardPages(&_board[0])
				}

				// if the deleted post is actually a thread, delete its posts
				_, err = db.Exec(`UPDATE "` + config.DBprefix + `posts" SET "deleted_timestamp" = "` + getSQLDateTime() + `" WHERE "parentid" = ` + post)
				if err != nil {
					serveErrorPage(writer, err.Error())
					return
				}

				// delete the file
				var deletedFilename string
				err = db.QueryRow(`SELECT "filename" FROM "` + config.DBprefix + `posts" WHERE "id" = ` + post + ` AND "filename" != ""`).Scan(&deletedFilename)
				if err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				err = db.QueryRow(`SELECT "filename" FROM "` + config.DBprefix + `posts" WHERE "parentid" = ` + post + ` AND "filename" != ""`).Scan(&deletedFilename)
				if err == nil {
					os.Remove(path.Join(config.DocumentRoot, board, "/src/", deletedFilename))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "t.", -1)))
					os.Remove(path.Join(config.DocumentRoot, board, "/thumb/", strings.Replace(deletedFilename, ".", "c.", -1)))
				}

				buildBoards(false, boardId)

				writer.Header().Add("refresh", "3;url="+request.Referer())
				fmt.Fprintf(writer, "%s deleted successfully\n<br />", post)
			}
		}
	}
}
