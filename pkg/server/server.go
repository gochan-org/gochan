package server

import (
	"fmt"
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

type ServerError struct {
	Err        any
	StatusCode int
}

func (e *ServerError) Error() string {
	return fmt.Sprint(e.Err)
}

func (e *ServerError) Unwrap() error {
	if err, ok := e.Err.(error); ok {
		return err
	}
	return nil
}

func NewServerError(message any, statusCode int) error {
	return &ServerError{Err: message, StatusCode: statusCode}
}

// ServeJSON serves data as a JSON string
func ServeJSON(writer http.ResponseWriter, data map[string]any) {
	jsonStr, _ := gcutil.MarshalJSON(data, false)
	writer.Header().Set("Content-Type", "application/json")
	serverutil.MinifyWriter(writer, []byte(jsonStr), "application/json")
}

// ServeErrorPage shows a general error page if something goes wrong
func ServeErrorPage(writer http.ResponseWriter, err any) {
	if se, ok := err.(*ServerError); ok {
		writer.WriteHeader(se.StatusCode)
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	serverutil.MinifyTemplate(gctemplates.ErrorPage, map[string]any{
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
func ServeError(writer http.ResponseWriter, err any, wantsJSON bool, data map[string]any) {
	if wantsJSON {
		servedMap := data
		if servedMap == nil {
			servedMap = make(map[string]any)
		}
		servedMap["error"] = err
		if se, ok := err.(*ServerError); ok {
			writer.WriteHeader(se.StatusCode)
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
