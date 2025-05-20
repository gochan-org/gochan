package main

import (
	"bytes"
	"context"
	"database/sql"
	"flag"
	"fmt"
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

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
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
	flag.IntVar(&workingConfig.Port, "port", 0, "Port to bind to (REQUIRED)")
	flag.BoolVar(&workingConfig.UseFastCGI, "fastcgi", false, "Use FastCGI instead of HTTP")
	flag.StringVar(&workingConfig.WebRoot, "webroot", "/", "Web root path")
	flag.StringVar(&workingConfig.TemplateDir, "template-dir", "", "Template directory (REQUIRED)")
	flag.StringVar(&workingConfig.DocumentRoot, "document-root", "", "Document root directory (REQUIRED)")
	flag.Parse()

	if jsonPath := config.GetGochanJSONPath(); jsonPath != "" {
		infoEv.Str("jsonPath", jsonPath).
			Msg("Gochan already installed (found gochan.json)")
		os.Exit(0)
	}

	config.SetSiteConfig(&workingConfig.SiteConfig)
	config.SetSystemCriticalConfig(&workingConfig.SystemCriticalConfig)

	systemCriticalConfig := config.GetSystemCriticalConfig()

	if systemCriticalConfig.TemplateDir == "" {
		flag.Usage()
		fatalEv.Msg("-template-dir command line argument is required")
	}

	if err = initTemplates(); err != nil {
		os.Exit(1)
	}

	listenAddr := net.JoinHostPort(workingConfig.SiteHost, strconv.Itoa(workingConfig.Port))

	router := server.GetRouter()
	router.GET(path.Join(workingConfig.WebRoot, "/install"), installHandler)
	router.POST(path.Join(workingConfig.WebRoot, "/install/:page"), installHandler)

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
	infoEv.Str("listenAddr", listenAddr).Msg("Starting installer server")
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

type pathsForm struct {
	ConfigPath   string `form:"configpath,required,notempty" method:"POST"`
	TemplateDir  string `form:"templatedir,required,notempty" method:"POST"`
	DocumentRoot string `form:"documentroot,required,notempty" method:"POST"`
	LogDir       string `form:"logdir,required,notempty" method:"POST"`
	WebRoot      string `form:"webroot,required,notempty" method:"POST"`
}

type dbForm struct {
	DBtype   string `form:"dbtype,required,notempty" method:"POST"`
	DBhost   string `form:"dbhost,required,notempty" method:"POST"`
	DBname   string `form:"dbname,required,notempty" method:"POST"`
	DBuser   string `form:"dbuser,required,notempty" method:"POST"`
	DBpass   string `form:"dbpass" method:"POST"`
	DBprefix string `form:"dbprefix" method:"POST"`
}

func testDB(form *dbForm) (tablesExist bool, err error) {
	var connStr string
	var query string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch form.DBtype {
	case "mysql":
		connStr = fmt.Sprintf(gcsql.MySQLConnStr, form.DBuser, form.DBpass, form.DBhost, form.DBname)
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`
	case "postgres":
		connStr = fmt.Sprintf(gcsql.PostgresConnStr, form.DBuser, form.DBpass, form.DBhost, form.DBname)
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE table_catalog = CURRENT_DATABASE() AND table_name = ?`
	case "sqlite3":
		connStr = fmt.Sprintf(gcsql.SQLite3ConnStr, form.DBhost, form.DBuser, form.DBpass)
		query = `SELECT COUNT(*) FROM sqlite_master WHERE name = ? AND type = 'table'`
	default:
		return false, gcsql.ErrUnsupportedDB
	}
	db, err := sql.Open(form.DBtype, connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var count int
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	if err = stmt.QueryRowContext(ctx, form.DBprefix+"database_version").Scan(&count); err != nil {
		return false, err
	}
	tablesExist = count > 0
	if err = stmt.Close(); err != nil {
		return
	}
	if err = db.Close(); err != nil {
		return
	}
	return
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
	systemCriticalConfig := config.GetSystemCriticalConfig()
	data := map[string]any{
		"page":                 page,
		"systemCriticalConfig": systemCriticalConfig,
		"siteConfig":           config.GetSiteConfig(),
		"nextButton":           "Next",
	}
	var stopServer bool
	switch page {
	case "":
		pageTitle = "Gochan Installation"
		data["nextPage"] = "license"
	case "license":
		pageTitle = "License"
		data["license"] = licenseTxt
		data["nextPage"] = "paths"
	case "paths":
		pageTitle = "Paths"
		data["nextPage"] = "database"
	case "database":

		pageTitle = "Database Setup"
		data["nextPage"] = "dbtest"
		data["nextButton"] = "Test Connection"
	case "dbtest":
		pageTitle = "Database Test"
		var dbFormData dbForm
		if err = forms.FillStructFromForm(req.Request, &dbFormData); err != nil {
			httpStatus = http.StatusBadRequest
			errEv.Err(err).Msg("Failed to fill form data")
			return
		}
		var tablesExist bool
		if tablesExist, err = testDB(&dbFormData); err != nil {
			httpStatus = http.StatusBadRequest
			errEv.Err(err).Msg("Database test failed")
			return err
		}
		data["tablesExist"] = tablesExist
		if tablesExist {
			data["testResult"] = fmt.Sprintf(
				"Database connection was successful but the database appears to contain Gochan tables (found %sdatabase_version). "+
					"Press Next to continue to use this database or your browser's back button to change the database settings.",
				dbFormData.DBprefix)
			warnEv.Str("dbprefix", dbFormData.DBprefix).Str("dbname", dbFormData.DBname).
				Msg("Database test successful but tables already exist")
		} else {
			data["testResult"] = "Database connection was successful. Press Next to continue."
		}
		systemCriticalConfig.DBtype = dbFormData.DBtype
		systemCriticalConfig.DBhost = dbFormData.DBhost
		systemCriticalConfig.DBname = dbFormData.DBname
		systemCriticalConfig.DBusername = dbFormData.DBuser
		systemCriticalConfig.DBpassword = dbFormData.DBpass
		systemCriticalConfig.DBprefix = dbFormData.DBprefix
		config.SetSystemCriticalConfig(systemCriticalConfig)
		data["nextPage"] = "install"
	case "stop":
		stopServer = true
	default:
		httpStatus = http.StatusNotFound
		pageTitle = "Page Not Found"
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
	if stopServer {
		installServerStopper <- 1
	}

	return nil
}
