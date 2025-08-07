package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"syscall"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/rs/zerolog"

	"github.com/gochan-org/gochan/pkg/gcutil"

	"github.com/gochan-org/gochan/pkg/gcplugin"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcsql/dbupdate"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func cleanup() {
	gcsql.Close()
	geoip.Close()
	gcplugin.ClosePlugins()
}

func main() {
	if len(os.Args) > 1 {
		parseCommandLine()
		return
	}
	gcutil.LogInfo().Str("version", config.GochanVersion).Msg("Starting gochan")
	fatalEv := gcutil.LogFatal()
	defer func() {
		fatalEv.Discard()
		cleanup()
	}()
	err := config.InitConfig()
	if errors.Is(err, fs.ErrNotExist) {
		fatalEv.Err(err).Caller()
		gcutil.LogArray("searchPaths", config.StandardConfigSearchPaths, fatalEv)
		fatalEv.Msg(config.ConfigNotFoundInPathsMessage)
	} else if err != nil {
		fatalEv.Err(err).Caller().Send()
	}

	uid, gid := config.GetUser()
	systemCritical := config.GetSystemCriticalConfig()
	if err = gcutil.InitLogs(systemCritical.LogDir, &gcutil.LogOptions{
		LogLevel: systemCritical.LogLevel(),
		UID:      uid,
		GID:      gid,
	}); err != nil {
		fatalEv.Err(err).Caller().
			Str("LogDir", systemCritical.LogDir).
			Int("uid", uid).
			Int("gid", gid).
			Msg("Unable to open logs")
	}
	fatalEv.Discard()
	fatalEv = gcutil.LogFatal() // reset fatalEv to use log file

	testIP := os.Getenv(gcutil.TestingIPEnvVar)
	if testIP != "" {
		gcutil.LogInfo().Str(gcutil.TestingIPEnvVar, testIP).
			Msg("Custom testing IP address set from environment variable")
	}

	if err = gcplugin.LoadPlugins(systemCritical.Plugins); err != nil {
		fatalEv.Err(err).Msg("Failed loading plugins")
	}

	events.TriggerEvent("startup")

	initDB(fatalEv)

	serverutil.InitMinifier()
	siteCfg := config.GetSiteConfig()
	if err = geoip.SetupGeoIP(siteCfg.GeoIPType, siteCfg.GeoIPOptions); err != nil {
		fatalEv.Err(err).Caller().Msg("Unable to initialize GeoIP")
	}
	if err = posting.InitCaptcha(); err != nil {
		fatalEv.Err(err).Caller().
			Str("CaptchaType", siteCfg.Captcha.Type).
			Msg("Unable to initialize CAPTCHA")
	}

	if err = gctemplates.InitTemplates(); err != nil {
		fatalEv.Err(err).Caller().Msg("Unable to initialize templates")
	}

	for _, board := range gcsql.AllBoards {
		if _, err = board.DeleteOldThreads(); err != nil {
			fatalEv.Err(err).Caller().
				Str("board", board.Dir).
				Msg("Failed deleting old threads")
		}
	}

	gcutil.LogInfo().Msg("Building consts.js")
	if err = building.BuildJS(); err != nil {
		fatalEv.Err(err).Caller().Msg("Failed building consts.js")
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	posting.InitPosting()
	defer events.TriggerEvent("shutdown")
	manage.InitManagePages()
	go initServer()
	gcutil.LogInfo().
		Str("ListenAddress", systemCritical.ListenAddress).
		Int("Port", systemCritical.Port).
		Str("SiteHost", systemCritical.SiteHost).
		Msg("Gochan server started")
	<-sc
}

func initDB(fatalEv *zerolog.Event, commandLine ...bool) {
	systemCritical := config.GetSystemCriticalConfig()
	if err := gcsql.ConnectToDB(&systemCritical.SQLConfig); err != nil {
		if len(commandLine) > 0 && commandLine[0] {
			fmt.Fprintln(os.Stderr, "Failed to connect to the database:", err)
		}
		fatalEv.Err(err).Msg("Failed to connect to the database")
	}
	events.TriggerEvent("db-connected")
	gcutil.LogInfo().
		Str("DBtype", systemCritical.DBtype).
		Str("DBhost", systemCritical.DBhost).
		Str("DBname", systemCritical.DBname).
		Msg("Connected to database")

	err := gcsql.CheckAndInitializeDatabase(systemCritical.DBtype, true)
	var db *gcsql.GCDB
	if errors.Is(err, gcsql.ErrDeprecatedDB) {
		db, err = gcsql.GetDatabase()
		if err == nil {
			err = dbupdate.UpdateDatabase(db)
		}
	}
	if err != nil {
		cleanup()
		if len(commandLine) > 0 && commandLine[0] {
			fmt.Fprintln(os.Stderr, "Failed to initialize the database:", err)
		}
		fatalEv.Err(err).Msg("Failed to initialize the database")
	}
	events.TriggerEvent("db-initialized")
	if err := gcsql.ResetViews(); err != nil {
		if len(commandLine) > 0 && commandLine[0] {
			fmt.Fprintln(os.Stderr, "Failed resetting SQL views:", err)
		}
		fatalEv.Err(err).Caller().Msg("Failed resetting SQL views")
	}
	events.TriggerEvent("db-views-reset")
}
