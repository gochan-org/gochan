package server

import (
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
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

// set mime type/cache headers according to the file's extension
func setFileHeaders(filename string, writer http.ResponseWriter) {
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
