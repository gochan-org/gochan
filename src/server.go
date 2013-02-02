package main 

import (
	"os"
	"fmt"
	"strconv"
	"path"
	"net"
	"net/url"
	"net/http"
	//"html/template"
)

var (
	form url.Values
	header http.Header
	cookies []*http.Cookie
)

func initServer() {
	if port == 0 {
		port = 80
	}
	getStyleLinks("manage")
	listener,err := net.Listen("tcp", domain+":"+strconv.Itoa(port))
	if(err != nil) {
		error_log.Write(err.Error())
		fmt.Println("Failed listening on "+domain+":"+strconv.Itoa(port)+", see log for details")
		os.Exit(2)
	}
	http.Handle("/", makeHandler(serveFile))
	http.Serve(listener, nil)
}

func getFileHTTPCode(filename string) int {
	filename = document_root+"/"+filename

	stat, err := os.Stat(filename);
	if err == nil {
		return 200
	} else {
		if stat.IsDir() {
			num_indexes := len(first_page)
			for i := 0; i < num_indexes; i++ {
				_,newerr := os.Stat(filename+"/"+first_page[i])
				if newerr == nil {
					return 200
				} else {
					return 404
				}
			}
		} else {
			return 200
		}
	}
	return 500
}

func serveFile(w http.ResponseWriter, request *http.Request, request_url string) {
	cookies = request.Cookies()
	request.ParseForm()
	form = request.Form

	if request.URL.Path == "/manage" {
		callManageFunction(w,request)
	} else {
		http.ServeFile(w, request, path.Join(document_root, request_url))
	}
	access_log.Write("Success: 200 from " + request.RemoteAddr + " @ " + request.RequestURI)
}

func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, request *http.Request) {
		defer func() {
			if _, ok := recover().(error); ok {
				//don't panic if the file doesn't exist
				//w.WriteHeader(404)
				http.ServeFile(w, request, path.Join(document_root, "404.html"))
				error_log.Write("Error: 404 Not Found from " + request.RemoteAddr + " @ " + request.RequestURI)
				return
			}
		}()
		title := request.URL.Path
		fn(w, request, title)
	}
}