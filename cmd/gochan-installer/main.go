package main

import (
	"bytes"
	"flag"
	"html/template"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	_ "embed"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/uptrace/bunrouter"
)

var (
	//go:embed license.txt
	licenseTxt string

	installTemplate      *template.Template
	installServerStopper chan int
	// workingConfig        *config.GochanConfig = config.GetDefaultConfig()
)

func main() {
	var err error

	fatalEv := gcutil.LogFatal()
	infoEv := gcutil.LogInfo()
	defer gcutil.LogDiscard(infoEv, fatalEv)

	workingConfig := config.GetDefaultConfig()

	flag.StringVar(&workingConfig.SiteHost, "host", "127.0.0.1", "Host to listen on")
	flag.IntVar(&workingConfig.Port, "port", 8080, "Port to bind to")
	flag.BoolVar(&workingConfig.UseFastCGI, "fastcgi", false, "Use FastCGI instead of HTTP")
	flag.StringVar(&workingConfig.WebRoot, "webroot", "/", "Web root path")
	flag.StringVar(&workingConfig.TemplateDir, "template-dir", "", "Template directory")
	flag.StringVar(&workingConfig.DocumentRoot, "document-root", "", "Document root directory")
	flag.Parse()

	if jsonPath := config.GetGochanJSONPath(); jsonPath != "" {
		infoEv.Str("jsonPath", jsonPath).
			Msg("Gochan already installed (found gochan.json)")
		os.Exit(0)
	}

	config.SetSiteConfig(&workingConfig.SiteConfig)
	config.SetSystemCriticalConfig(&workingConfig.SystemCriticalConfig)

	if err = initTemplates(); err != nil {
		os.Exit(1)
	}

	listenAddr := net.JoinHostPort(workingConfig.SiteHost, strconv.Itoa(workingConfig.Port))

	router := server.GetRouter()
	router.GET(path.Join(workingConfig.WebRoot, "/install"), installHandler)
	router.POST(path.Join(workingConfig.WebRoot, "/install/:page"), installHandler)
	// router.GET(path.Join(workingConfig.WebRoot, "/install/:page"), installHandler)

	if workingConfig.DocumentRoot == "" {
		fatalEv.Msg("-document-root command line argument is required")
		os.Exit(1)
	}

	var listener net.Listener
	installServerStopper = make(chan int)
	go func() {
		<-installServerStopper
		if listener != nil {
			if err = listener.Close(); err != nil {
				fatalEv.Err(err).Caller().Msg("Failed to close listener")
			}
		}
	}()
	if workingConfig.UseFastCGI {
		listener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			fatalEv.Err(err).Caller().Msg("Failed listening on address/port")
		}
		if err = fcgi.Serve(listener, router); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			fatalEv.Err(err).Caller().Msg("Failed to serve FastCGI")
		}
	} else {
		httpServer := &http.Server{
			Addr:              listenAddr,
			Handler:           router,
			ReadHeaderTimeout: 5 * time.Second,
		}
		if err = httpServer.ListenAndServe(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			fatalEv.Err(err).Caller().Msg("Failed to serve HTTP")
		}
	}

	if err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
		fatalEv.Err(err).Caller().Msg("Error initializing server")
	}
}

func initTemplates() error {
	var err error

	fatalEv := gcutil.LogFatal()
	defer fatalEv.Discard()

	systemCriticalConfig := config.GetSystemCriticalConfig()

	if systemCriticalConfig.TemplateDir == "" {
		fatalEv.Msg("-template-dir command line argument is required")
		return nil
	}

	if err = gctemplates.InitTemplates(); err != nil {
		fatalEv.Err(err).Caller().Msg("Failed to initialize templates")
		return err
	}

	installTemplateBytes, err := os.ReadFile(path.Join(systemCriticalConfig.TemplateDir, "install.html"))
	if err != nil {
		fatalEv.Err(err).Caller().Msg("Failed to read install template")
	}
	if installTemplate, err = gctemplates.ParseTemplate("install.html", string(installTemplateBytes)); err != nil {
		fatalEv.Err(err).Caller().Msg("Failed to parse install template")
		return err
	}

	return nil
}

func installHandler(writer http.ResponseWriter, req bunrouter.Request) (err error) {
	infoEv, warnEv, errEv := gcutil.LogRequest(req.Request)
	var buf bytes.Buffer
	httpStatus := http.StatusOK
	defer func() {
		gcutil.LogDiscard(infoEv, warnEv, errEv)
		writer.WriteHeader(httpStatus)
		if err == nil {
			writer.Write(buf.Bytes())
		} else {
			server.ServeError(writer, err, false, nil)
		}
	}()
	var pageTitle string
	page := req.Param("page")
	data := map[string]any{
		"page": page,
	}
	switch page {
	case "":
		pageTitle = "Gochan Installation"
	case "license":
		pageTitle = "License"
		data["license"] = licenseTxt
	case "database":
		pageTitle = "Database Setup"
	case "stop":
		writer.Write([]byte("Stopping server..."))
		installServerStopper <- 1 // Stop the server
		return nil
	}

	if err = building.BuildPageHeader(&buf, pageTitle, "", data); err != nil {
		httpStatus = http.StatusInternalServerError
		errEv.Err(err).Msg("Failed to build page header")
		return
	}
	if err = serverutil.MinifyTemplate(installTemplate, data, &buf, "text/html"); err != nil {
		httpStatus = http.StatusInternalServerError
		errEv.Err(err).Msg("Failed to minify template")
		return
	}
	if err = building.BuildPageFooter(&buf); err != nil {
		httpStatus = http.StatusInternalServerError
		errEv.Err(err).Msg("Failed to build page footer")
		return
	}

	return nil
}
