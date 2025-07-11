package main

import (
	"flag"
	"html/template"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server"
)

var (
	installTemplate      *template.Template
	installServerStopper chan int

	cfg *config.GochanConfig = config.GetDefaultConfig()
)

func main() {
	var err error

	slices.Reverse(cfgPaths)

	fatalEv := gcutil.LogFatal()
	infoEv := gcutil.LogInfo()
	defer gcutil.LogDiscard(infoEv, fatalEv)

	flag.StringVar(&cfg.ListenAddress, "host", "127.0.0.1", "Host to listen on")
	flag.IntVar(&cfg.Port, "port", 0, "Port to bind to (REQUIRED)")
	flag.BoolVar(&cfg.UseFastCGI, "fastcgi", false, "Use FastCGI instead of HTTP")
	flag.StringVar(&cfg.WebRoot, "webroot", "/", "Web root path")
	flag.StringVar(&cfg.TemplateDir, "template-dir", "", "Template directory (REQUIRED)")
	flag.StringVar(&cfg.DocumentRoot, "document-root", "", "Document root directory (REQUIRED)")
	flag.StringVar(&cfg.SiteHost, "site-host", "", "Site host (e.g. example.com) that will be used for incoming URLs")
	flag.Parse()

	if cfg.SiteHost == "" {
		cfg.SiteHost = cfg.ListenAddress
	}
	if jsonPath := config.GetGochanJSONPath(); jsonPath != "" {
		infoEv.Str("jsonPath", jsonPath).
			Msg("Gochan already installed (found gochan.json)")
		os.Exit(0)
	}

	config.SetSiteConfig(&cfg.SiteConfig)
	config.SetSystemCriticalConfig(&cfg.SystemCriticalConfig)

	systemCriticalConfig := config.GetSystemCriticalConfig()

	if systemCriticalConfig.TemplateDir == "" {
		flag.Usage()
		fatalEv.Msg("-template-dir command line argument is required")
	}

	if err = initTemplates(); err != nil {
		os.Exit(1)
	}

	listenAddr := net.JoinHostPort(cfg.ListenAddress, strconv.Itoa(cfg.Port))

	router := server.GetRouter()
	router.GET(path.Join(cfg.WebRoot, "/install"), installHandler)
	router.POST(path.Join(cfg.WebRoot, "/install/:page"), installHandler)

	if cfg.DocumentRoot == "" {
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
	infoEv.Str("siteHost", cfg.SiteHost).Str("listenAddr", listenAddr).Msg("Starting installer server")
	if cfg.UseFastCGI {
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
