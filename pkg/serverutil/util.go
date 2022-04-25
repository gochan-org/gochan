package serverutil

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// ServeJSON serves data as a JSON string
func ServeJSON(writer http.ResponseWriter, data map[string]interface{}) {
	jsonStr, _ := gcutil.MarshalJSON(data, false)
	writer.Header().Set("Content-Type", "application/json")
	MinifyWriter(writer, []byte(jsonStr), "application/json")
}

// ServeErrorPage shows a general error page if something goes wrong
func ServeErrorPage(writer http.ResponseWriter, err string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
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

// DeleteCookie deletes the given cookie if it exists. It returns true if it exists and false
// with no errors if it doesn't
func DeleteCookie(writer http.ResponseWriter, request *http.Request, cookieName string) bool {
	cookie, err := request.Cookie(cookieName)
	if err != nil {
		return false
	}
	cookie.MaxAge = 0
	cookie.Expires = time.Now().Add(-7 * 24 * time.Hour)
	http.SetCookie(writer, cookie)
	return true
}
