package server

import (
	"net/http"
	"os"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/uptrace/bunrouter"
)

var (
	router *bunrouter.Router
)

// ServeJSON serves data as a JSON string
func ServeJSON(writer http.ResponseWriter, data map[string]interface{}) {
	jsonStr, _ := gcutil.MarshalJSON(data, false)
	writer.Header().Set("Content-Type", "application/json")
	serverutil.MinifyWriter(writer, []byte(jsonStr), "application/json")
}

// ServeErrorPage shows a general error page if something goes wrong
func ServeErrorPage(writer http.ResponseWriter, err string) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	serverutil.MinifyTemplate(gctemplates.ErrorPage, map[string]interface{}{
		"systemCritical": config.GetSystemCriticalConfig(),
		"siteConfig":     config.GetSiteConfig(),
		"boardConfig":    config.GetBoardConfig(""),
		"errorTitle":     "Error :c",
		"errorHeader":    "Error",
		"errorText":      err,
	}, writer, "text/html")
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

// ServeNotFound shows an error page if a requested file is not found
func ServeNotFound(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.WriteHeader(http.StatusNotFound)
	systemCritical := config.GetSystemCriticalConfig()
	errorPage, err := os.ReadFile(systemCritical.DocumentRoot + "/error/404.html")
	if err != nil {
		writer.Write([]byte("Requested page not found, and /error/404.html not found"))
	} else {
		serverutil.MinifyWriter(writer, errorPage, "text/html")
	}
	gcutil.LogAccess(request).Int("status", 404).Msg("requested page or resource not found")
}

func InitRouter() {
	router = bunrouter.New(
		bunrouter.WithNotFoundHandler(bunrouter.HTTPHandlerFunc(serveFile)),
	)
}

func GetRouter() *bunrouter.Router {
	if router == nil {
		InitRouter()
	}
	return router
}
