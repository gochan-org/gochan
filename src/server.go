package main 

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/fcgi"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
)

var (
	form url.Values
	header http.Header
	cookies []*http.Cookie
	writer http.ResponseWriter
	request http.Request
	exit_error bool
	server *GochanServer
)

type GochanServer struct{
	writer http.ResponseWriter
	request http.Request
	namespaces map[string]func(http.ResponseWriter, *http.Request, interface{})
}

func (s GochanServer) AddNamespace(base_path string, namespace_function func(http.ResponseWriter, *http.Request, interface{})) {
	s.namespaces[base_path] = namespace_function
}

func (s GochanServer) getFileData(writer http.ResponseWriter, url string) ([]byte, bool) {
	var file_bytes []byte
	filepath := path.Join(config.DocumentRoot, url)
	results,err := os.Stat(filepath)
	if err != nil {
		fmt.Println("404 at ", filepath)
		// the requested path isn't a file or directory, 404
		return file_bytes, false
	} else {
		//the file exists, or there is a folder here
		if results.IsDir() {
			found_index := false
			newpath := ""

			//check to see if one of the specified index pages exists
			for i := 0; i < len(config.FirstPage); i++ {
				newpath = path.Join(filepath,config.FirstPage[i])
				_,err := os.Stat(newpath)
				if err == nil {
					// serve the index page
					writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
					fmt.Println("found index at ", newpath)
					file_bytes,err  = ioutil.ReadFile(newpath)
					return file_bytes, true
					found_index = true
					break
				}
			}

			if !found_index {
				// none of the index pages specified in config.cfg exist
				return file_bytes, false
			}
		} else {
			//the file exists, and is not a folder
			file_bytes, err = ioutil.ReadFile(filepath)
			extension := getFileExtension(url)
			switch {
				case extension == "png":
					writer.Header().Add("Content-Type", "image/png")
					writer.Header().Add("Cache-Control", "max-age=86400")
				case extension == "gif":
					writer.Header().Add("Content-Type", "image/gif")
					writer.Header().Add("Cache-Control", "max-age=86400")
				case extension == "jpg":
					writer.Header().Add("Content-Type", "image/jpeg")
					writer.Header().Add("Cache-Control", "max-age=86400")
				case extension == "css":
					writer.Header().Add("Content-Type", "text/css")
					writer.Header().Add("Cache-Control", "max-age=43200")
				case extension == "js":
					writer.Header().Add("Content-Type", "text/javascript")
					writer.Header().Add("Cache-Control", "max-age=43200")
			}
			if extension  == "html" || extension == "htm" {
				writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
			}
			//http.ServeFile(writer, request, filepath)
			access_log.Print("Success: 200 from " + request.RemoteAddr + " @ " + request.RequestURI)
			return file_bytes, true
		}
	}
	return file_bytes, false
}

func (s GochanServer) Redirect(location string) {
	http.Redirect(writer,&request,location,http.StatusFound)
}

func (s GochanServer) serve404(writer http.ResponseWriter, request *http.Request) {
	error_page, err := ioutil.ReadFile(config.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and 404 error page not found"))
	} else {
		writer.Write(error_page)
	}
	error_log.Print("Error: 404 Not Found from " + request.RemoteAddr + " @ " + request.RequestURI)
}

func (s GochanServer) ServeErrorPage(writer http.ResponseWriter, err string) {
	error_page_bytes,_ := ioutil.ReadFile("templates/error.html")
	error_page := string(error_page_bytes)
	error_page = strings.Replace(error_page,"{ERRORTEXT}", err,-1)
	writer.Write([]byte(error_page))
	exit_error = true
}

func (s GochanServer) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	for name, namespace_function := range s.namespaces {
		//if len(request.URL)
		if request.URL.Path == "/" + name {
			namespace_function(writer, request, nil)
			return
		}
	}
	fb,found := s.getFileData(writer, request.URL.Path)
	writer.Header().Add("Cache-Control", "max-age=86400")
	if !found {
		s.serve404(writer, request)
		return
	}
	writer.Write(fb)
}

func initServer() {
	listener,err := net.Listen("tcp", config.Domain+":"+strconv.Itoa(config.Port))
	if(err != nil) {
		fmt.Printf("Failed listening on "+config.Domain+":%d, see log for details",config.Port)
		error_log.Fatal(err.Error())
	}
	server = new(GochanServer)
	server.namespaces = make(map[string]func(http.ResponseWriter, *http.Request, interface{}))

	testfunc := func(writer http.ResponseWriter, response *http.Request, data interface{}) {
		if writer != nil {
			writer.Write([]byte("hahahaha"))
		}
	}
	server.AddNamespace("example", testfunc)
	server.AddNamespace("manage", callManageFunction)
	server.AddNamespace("post", makePost)
	server.AddNamespace("util", utilHandler)
	if config.UseFastCGI {
		fcgi.Serve(listener,server)
	} else {
		http.Serve(listener, server)
	}
}

func validReferrer(request http.Request) (valid bool) {
	if request.Referer() == "" || request.Referer()[7:len(config.SiteDomain)+7] != config.SiteDomain {
	// if request.Referer() == "" || request.Referer()[7:len(config.Domain)+7] != config.Domain {
		valid = false
	} else {
		valid = true
	}
	return
}

func utilHandler(writer http.ResponseWriter, request *http.Request, data interface{}) {
	action := request.FormValue("action")
	board := request.FormValue("board")
	if action == "" && request.PostFormValue("delete_btn") != "Delete" && request.PostFormValue("report_btn") != "Report" {
		http.Redirect(writer,request,path.Join(config.SiteWebfolder,"/"),http.StatusFound)
		return
	}
	var posts_arr []string
	for key,_ := range request.PostForm {
		if strings.Index(key,"check") == 0 {
			posts_arr = append(posts_arr,key[5:])
		}
	}
	if request.PostFormValue("delete_btn") == "Delete" {
		file_only := request.FormValue("fileonly") == "on"
		password := md5_sum(request.FormValue("password"))
		rank := getStaffRank()

		if request.FormValue("password") == ""  && rank == 0 {
			server.ServeErrorPage(writer, "Password required for post deletion")
			return
		}

		for _,post := range posts_arr {
			var parent_id int
			var filename string
			var filetype string
			var password_checksum string
			var board_id int
			post_int,err := strconv.Atoi(post)

			err = db.QueryRow("SELECT `parentid`,`filename`,`password` FROM `"+config.DBprefix+"posts` WHERE `id` = "+post).Scan(&parent_id,&filename,&password_checksum)
			if err == sql.ErrNoRows {
				//the post has already been deleted
				fmt.Fprintf(writer, "%s has already been deleted\n",post)
				continue
			}
			if err != nil {
				server.ServeErrorPage(writer,err.Error())
				return
			}

			err = db.QueryRow("SELECT `id` FROM `"+config.DBprefix+"boards` WHERE `dir` = '"+board+"'").Scan(&board_id)
			if err != nil {
				server.ServeErrorPage(writer,err.Error())
				return
			}

			if password != password_checksum && rank == 0 {
				fmt.Fprintf(writer, "Incorrect password for %s\n", post)
				continue
			}

			if file_only {
				if filename != "" {
					filetype = filename[strings.Index(filename,".")+1:]
					filename = filename[:strings.Index(filename,".")]
					err := os.Remove(path.Join(config.DocumentRoot,board,"/src/"+filename+"."+filetype))
					if err != nil {
						server.ServeErrorPage(writer,err.Error())
						return
					}
					err = os.Remove(path.Join(config.DocumentRoot,board,"/thumb/"+filename+"t."+filetype))
					if err != nil {
						server.ServeErrorPage(writer,err.Error())
						return
					}
					_,err = db.Exec("UPDATE `"+config.DBprefix+"posts` SET `filename` = 'deleted' WHERE `id` = "+post)
					if err != nil {
						server.ServeErrorPage(writer,err.Error())
						return
					}
				}
				fmt.Fprintf(writer, "Attached image from %s deleted successfully<br />\n<meta http-equiv=\"refresh\" content=\"1;url=http://lunachan.net/test/\">", post)
			} else {
				if parent_id > 0 {
					os.Remove(path.Join(config.DocumentRoot,board,"/res/index.html"))
				}
				_,err = db.Exec("DELETE FROM `"+config.DBprefix+"posts` WHERE `id` = "+post)
				if parent_id == 0 {
					err = buildThread(post_int, board_id)
				} else {
					err = buildThread(parent_id,board_id)
				}

				if err != nil {
					server.ServeErrorPage(writer,err.Error())
					return
				}
				fmt.Fprintf(writer, "%s deleted successfully\n", post)
				writer.Header().Add("refresh", "5;url="+request.Referer())
			}
		}
	}
}
