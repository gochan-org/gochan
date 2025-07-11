package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/uptrace/bunrouter"
)

const (
	dbStatusUnknown dbStatus = iota
	dbStatusClean
	dbStatusNoPrefix
	dbStatusTablesExist
)

var (
	//go:embed license.txt
	licenseTxt string

	configPath      string
	currentDBStatus = dbStatusUnknown
	adminUser       *gcsql.Staff
	cfgPaths        = slices.Clone(config.StandardConfigSearchPaths)
)

type dbStatus int

func (dbs dbStatus) String() string {
	switch dbs {
	case dbStatusClean:
		return "The database does not appear to contain any Gochan tables. It will be provisioned in the next step."
	case dbStatusNoPrefix:
		return "Since no prefix was specified, the installer will attempt to provision the database in the next step."
	case dbStatusTablesExist:
		return fmt.Sprintf("The database appears to contain Gochan tables with the prefix %s. The next step (database provisioning) may return errors", config.GetSystemCriticalConfig().DBprefix)
	default:
		return "unknown"
	}
}

func installHandler(writer http.ResponseWriter, req bunrouter.Request) (err error) {
	infoEv, warnEv, errEv := gcutil.LogRequest(req.Request)
	var buf bytes.Buffer
	httpStatus := http.StatusOK
	page := req.Param("page")

	defer func() {
		gcutil.LogDiscard(infoEv, warnEv, errEv)
		writer.WriteHeader(httpStatus)
		if err == nil {
			writer.Write(buf.Bytes())
		} else {
			server.ServeError(writer, err, false, nil)
		}
		if page == "save" {
			installServerStopper <- 1
		}
	}()
	var pageTitle string
	data := map[string]any{
		"page":       page,
		"config":     cfg,
		"nextButton": "Next",
	}

	refererResult, err := serverutil.CheckReferer(req.Request)
	if err != nil {
		httpStatus = http.StatusBadRequest
		warnEv.Err(err).Caller().
			Str("referer", req.Referer()).
			Msg("Failed to check referer")
		return
	}

	if refererResult == serverutil.NoReferer && req.Method == http.MethodPost {
		httpStatus = http.StatusBadRequest
		warnEv.Caller().Msg("No referer present for POST request")
		return
	} else if refererResult == serverutil.ExternalReferer {
		httpStatus = http.StatusForbidden
		warnEv.Caller().
			Str("referer", req.Referer()).
			Msg("Request came from an external referer (not allowed during installation)")
		return errors.New("your post looks like spam")
	}

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
		data["cfgPaths"] = cfgPaths
		data["nextPage"] = "database"
	case "database":
		var pathFormData pathsForm
		if err = forms.FillStructFromForm(req.Request, &pathFormData); err != nil {
			httpStatus = http.StatusBadRequest
			errEv.Err(err).Msg("Failed to fill form data")
			return
		}
		if err = pathFormData.validate(warnEv, errEv); err != nil {
			httpStatus = http.StatusBadRequest
			return
		}
		configPath = pathFormData.ConfigPath
		cfg.DocumentRoot = pathFormData.DocumentRoot
		cfg.LogDir = pathFormData.LogDir
		cfg.TemplateDir = pathFormData.TemplateDir
		cfg.WebRoot = pathFormData.WebRoot
		config.SetSystemCriticalConfig(&cfg.SystemCriticalConfig)

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
		if currentDBStatus, err = dbFormData.validate(); err != nil {
			httpStatus = http.StatusBadRequest
			errEv.Err(err).Msg("Database test failed")
			return err
		}

		data["testResult"] = currentDBStatus.String()
		cfg.DBtype = dbFormData.DBtype
		cfg.DBhost = dbFormData.DBhost
		cfg.DBname = dbFormData.DBname
		cfg.DBusername = dbFormData.DBuser
		cfg.DBpassword = dbFormData.DBpass
		cfg.DBprefix = dbFormData.DBprefix
		config.SetSystemCriticalConfig(&cfg.SystemCriticalConfig)

		data["nextPage"] = "staff"
	case "staff":
		pageTitle = "Create Administrator Account"

		if adminUser != nil {
			data["nextButton"] = "Next"
			data["alreadyCreated"] = true
			break
		}

		// staff not created yet, show new admin form
		if currentDBStatus == dbStatusUnknown {
			httpStatus = http.StatusBadRequest
			errEv.Msg("Database status is unknown, cannot proceed with provisioning")
			return errors.New("database status is unknown, cannot proceed with provisioning")
		}

		err := gcsql.CheckAndInitializeDatabase(cfg.DBtype, false)
		if err != nil {
			errEv.Err(err).Msg("Failed to initialize database")
			httpStatus = http.StatusInternalServerError
			return err
		}

		if err = gcsql.ResetViews(); err != nil {
			errEv.Err(err).Msg("Failed to reset database views")
			httpStatus = http.StatusInternalServerError
			return err
		}

		data["nextPage"] = "pre-save"
	case "pre-save":
		pageTitle = "Configuration Confirmation"

		var staffFormData staffForm
		if err = forms.FillStructFromForm(req.Request, &staffFormData); err != nil {
			httpStatus = http.StatusBadRequest
			errEv.Err(err).Msg("Failed to fill form data")
			return
		}
		if err = staffFormData.validate(); err != nil {
			httpStatus = http.StatusBadRequest
			warnEv.Err(err).Msg("Invalid staff form data")
			return
		}

		adminUser, err = gcsql.NewStaff(staffFormData.Username, staffFormData.Password, 3)
		if err != nil {
			httpStatus = http.StatusInternalServerError
			errEv.Err(err).Msg("Failed to create administrator account")
			return err
		}

		if configPath == "" {
			httpStatus = http.StatusBadRequest
			errEv.Msg("Configuration path is not set")
			return errors.New("configuration path is not set")
		}

		var jsonBuf bytes.Buffer
		encoder := json.NewEncoder(&jsonBuf)
		encoder.SetIndent("", "   ")
		if err = encoder.Encode(cfg); err != nil {
			httpStatus = http.StatusInternalServerError
			errEv.Err(err).Msg("Failed to encode configuration to JSON")
			return err
		}
		data["configJSON"] = jsonBuf.String()
		data["configPath"] = configPath
		data["nextButton"] = "Save"
		data["nextPage"] = "save"
	case "save":
		pageTitle = "Save Configuration"
		if configPath == "" {
			httpStatus = http.StatusBadRequest
			errEv.Msg("Configuration path is not set")
			return errors.New("configuration path is not set")
		}

		if err = config.WriteConfig(configPath); err != nil {
			httpStatus = http.StatusInternalServerError
			errEv.Err(err).Msg("Failed to write configuration")
			return err
		}

		if err = building.BuildFrontPage(); err != nil {
			httpStatus = http.StatusInternalServerError
			errEv.Err(err).Msg("Failed to build front page")
			return err
		}

		if err = building.BuildBoards(true); err != nil {
			httpStatus = http.StatusInternalServerError
			errEv.Err(err).Msg("Failed to build boards")
			return err
		}

		infoEv.Str("configPath", configPath).Msg("Configuration written successfully")
		data["nextPage"] = ""
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

	return nil
}
