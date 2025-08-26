package server

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	knownFileHeaders = map[string]StaticFileHeaders{
		".png":  {ContentType: "image/png", CacheControl: "max-age=86400"},
		".gif":  {ContentType: "image/gif", CacheControl: "max-age=86400"},
		".jpg":  {ContentType: "image/jpeg", CacheControl: "max-age=86400"},
		".jpeg": {ContentType: "image/jpeg", CacheControl: "max-age=86400"},
		".svg":  {ContentType: "image/svg+xml", CacheControl: "max-age=86400"},
		".css":  {ContentType: "text/css", CacheControl: "max-age=43200"},
		".js":   {ContentType: "text/javascript", CacheControl: "max-age=43200"},
		".json": {ContentType: "application/json", CacheControl: "max-age=5, must-revalidate"},
		".webm": {ContentType: "video/webm", CacheControl: "max-age=86400"},
		".htm":  {ContentType: "text/html", CacheControl: "max-age=5, must-revalidate"},
		".html": {ContentType: "text/html", CacheControl: "max-age=5, must-revalidate"},
	}
)

func serveFile(writer http.ResponseWriter, request *http.Request) {
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()

	requestPath := request.URL.Path
	if len(systemCritical.WebRoot) > 0 && systemCritical.WebRoot != "/" {
		requestPath = requestPath[len(systemCritical.WebRoot):]
	}
	filePath := path.Join(systemCritical.DocumentRoot, requestPath)
	var fileBytes []byte
	results, err := os.Stat(filePath)
	if err != nil {
		// the requested path isn't a file or directory, 404
		ServeNotFound(writer, request)
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
			ServeNotFound(writer, request)
			return
		}
	}
	setFileHeaders(filePath, writer)

	// serve the requested file
	fileBytes, _ = os.ReadFile(filePath)
	gcutil.LogAccess(request).Int("status", 200).Send()
	writer.Write(fileBytes)
}

type StaticFileHeaders struct {
	ContentType  string
	CacheControl string
	Other        map[string]string
}

// set mime type/cache headers according to the file's extension
func setFileHeaders(filename string, writer http.ResponseWriter) {
	extension := strings.ToLower(path.Ext(filename))
	header, ok := knownFileHeaders[extension]
	if ok {
		writer.Header().Set("Content-Type", header.ContentType)
		writer.Header().Set("Cache-Control", header.CacheControl)
		for key, value := range header.Other {
			writer.Header().Set(key, value)
		}
	} else {
		writer.Header().Set("Content-Type", "application/octet-stream")
		writer.Header().Set("Cache-Control", "max-age=86400")
	}
}
