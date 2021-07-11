package serverutil

import (
	"io/ioutil"
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// ServeErrorPage shows a general error page if something goes wrong
func ServeErrorPage(writer http.ResponseWriter, err string) {
	MinifyTemplate(gctemplates.ErrorPage, map[string]interface{}{
		"systemCritical": config.GetSystemCriticalConfig(),
		"siteConfig":     config.GetSiteConfig(),
		"boardConfig":    config.GetBoardConfig(""),
		"ErrorTitle":     "Error :c",
		// "ErrorImage":  "/error/lol 404.gif",
		"ErrorHeader": "Error",
		"ErrorText":   err,
	}, writer, "text/html")
}

// ServeNotFound shows an error page if a requested file is not found
func ServeNotFound(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Add("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(404)
	systemCritical := config.GetSystemCriticalConfig()
	errorPage, err := ioutil.ReadFile(systemCritical.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and /error/404.html not found"))
	} else {
		MinifyWriter(writer, errorPage, "text/html")
	}
	gclog.Printf(gclog.LAccessLog, "Error: 404 Not Found from %s @ %s", gcutil.GetRealIP(request), request.URL.Path)
}
