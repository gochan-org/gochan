package main 

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
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
)

func initServer() {
	if config.Port == 0 {
		config.Port = 80
	}
	listener,err := net.Listen("tcp", config.Domain+":"+strconv.Itoa(config.Port))
	if(err != nil) {
		error_log.Write(err.Error())
		fmt.Printf("Failed listening on "+config.Domain+":%d, see log for details",config.Port)
		os.Exit(2)
	}
	http.Handle("/", makeHandler(fileHandle))
	http.Handle("/manage",makeHandler(callManageFunction))
	http.Handle("/post",makeHandler(makePost))
	//http.Handle("/util",makeHandler(utilHandler))
	http.Serve(listener, nil)
}

func fileHandle(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	cookies = request.Cookies()
	request.ParseForm()
	form = request.Form
	request_url := request.URL.Path

	filepath := path.Join(config.DocumentRoot, request_url)
	results,err := os.Stat(filepath)

	if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {

	}

	if err == nil {
		//the file exists, or there is a folder here
		if results.IsDir() {
			found_index := false
			newpath := ""

			//check to see if one of the specified index pages exists
			for i := 0; i < len(config.FirstPage); i++ {
				newpath = path.Join(filepath,config.FirstPage[i])
				_,err := os.Stat(newpath)
				if err == nil {
					serveFile(w, newpath)
					found_index = true
					break
				}
			}

			if !found_index {
				error404()
			}
		} else {
			//the file exists, and is not a folder
			//writer.Header().Add("Cache-Control", fmt.Sprintf("max-age=%d, public, must-revalidate, proxy-revalidate", 500))
			serveFile(w, filepath)
		}
	} else {
		//there is nothing at the requested address
		error404()
	}
}

func makeHandler(fn func(http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		//defer serverError()
		if !exit_error {
			fn(w, r)
			exit_error = false
		} else {
			exit_error = false
		}
	}
}

func exitWithErrorPage(writer http.ResponseWriter, err string) {
	error_page_bytes,_ := ioutil.ReadFile("templates/error.html")
	error_page := string(error_page_bytes)
	error_page = strings.Replace(error_page,"{ERRORTEXT}", err,-1)
	fmt.Fprintf(writer,error_page)
	exit_error = true
}

func redirect(location string) {
	//http.Redirect(writer,&request,location,http.StatusMovedTemporarily)
}

func error404() {
	http.ServeFile(writer, &request, path.Join(config.DocumentRoot, "/error/404.html"))
	error_log.Write("Error: 404 Not Found from " + request.RemoteAddr + " @ " + request.RequestURI)
}

func serverError() {
	if _, ok := recover().(error); ok {
		//something went wrong, now we need to throw a 500
		http.ServeFile(writer,&request, path.Join(config.DocumentRoot, "/error/500.html"))
		error_log.Write("Error: 500 Internal Server error from " + request.RemoteAddr + " @ " + request.RequestURI)	
		return
	}
}

func serveFile(w http.ResponseWriter, filepath string) {
	http.ServeFile(w, &request, filepath)
	access_log.Write("Success: 200 from " + request.RemoteAddr + " @ " + request.RequestURI)
}
