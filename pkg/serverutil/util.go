package serverutil

import (
	"net/http"
	"os"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// ServeJSON serves data as a JSON string
func ServeJSON(writer http.ResponseWriter, data map[string]interface{}) {
	jsonStr, _ := gcutil.MarshalJSON(data, false)
	writer.Header().Set("Content-Type", "application/json")
	MinifyWriter(writer, []byte(jsonStr), "application/json")
}

// ServeError serves the given map as a JSON file (with the error string included) if wantsJSON is true,
// otherwise it serves a regular HTML error page
func ServeError(writer http.ResponseWriter, err string, wantsJSON bool, data map[string]interface{}) {
	if wantsJSON {
		servedMap := data
		if servedMap == nil {
			servedMap = make(map[string]interface{})
		}
		if _, ok := servedMap["error"]; !ok {
			servedMap["error"] = err
		}
		ServeJSON(writer, servedMap)
	} else {
		ServeErrorPage(writer, err)
	}
}

// ServeErrorPage shows a general error page if something goes wrong
func ServeErrorPage(writer http.ResponseWriter, err string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	MinifyTemplate(gctemplates.ErrorPage, map[string]interface{}{
		"systemCritical": config.GetSystemCriticalConfig(),
		"siteConfig":     config.GetSiteConfig(),
		"boardConfig":    config.GetBoardConfig(""),
		"errorTitle":     "Error :c",
		"errorHeader":    "Error",
		"errorText":      err,
	}, writer, "text/html")
}

// ServeNotFound shows an error page if a requested file is not found
func ServeNotFound(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusNotFound)
	systemCritical := config.GetSystemCriticalConfig()
	errorPage, err := os.ReadFile(systemCritical.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and /error/404.html not found"))
	} else {
		MinifyWriter(writer, errorPage, "text/html")
	}
	gcutil.LogAccess(request).Int("status", 404).Msg("requested page or resource not found")
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

func IsRequestingJSON(request *http.Request) bool {
	request.ParseForm()
	field := request.Form["json"]
	return len(field) == 1 && (field[0] == "1" || field[0] == "true")
}
