package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/rs/zerolog"

	"github.com/gochan-org/gochan/pkg/gcutil"

	"github.com/gochan-org/gochan/pkg/gcplugin"
	"github.com/gochan-org/gochan/pkg/gcsql"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

var (
	versionStr   string
	dbVersionStr string
)

func cleanup() {
	gcsql.Close()
	geoip.Close()
	gcplugin.ClosePlugins()
}

func main() {
	gcutil.LogInfo().Str("version", versionStr).Msg("Starting gochan")
	fatalEv := gcutil.LogFatal()
	defer func() {
		fatalEv.Discard()
		cleanup()
	}()
	err := config.InitConfig(versionStr)
	if err != nil {
		fatalEv.Err(err).Caller().
			Str("jsonLocation", config.JSONLocation()).
			Msg("Unable to load configuration")
	}

	uid, gid := config.GetUser()
	systemCritical := config.GetSystemCriticalConfig()
	if err = gcutil.InitLogs(systemCritical.LogDir, systemCritical.LogLevel(), uid, gid); err != nil {
		fatalEv.Err(err).Caller().
			Str("logDir", systemCritical.LogDir).
			Int("uid", uid).
			Int("gid", gid).
			Msg("Unable to open logs")
	}
	fatalEv.Discard()
	fatalEv = gcutil.LogFatal() // reset fatalEv to use log file

	testIP := os.Getenv("GC_TESTIP")
	if testIP != "" {
		gcutil.LogInfo().Str("GC_TESTIP", testIP).
			Msg("Custom testing IP address set from environment variable")
	}

	if err = gcplugin.LoadPlugins(systemCritical.Plugins); err != nil {
		fatalEv.Err(err).Msg("Failed loading plugins")
	}

	events.TriggerEvent("startup")

	if err = gcsql.ConnectToDB(&systemCritical.SQLConfig); err != nil {
		fatalEv.Err(err).Msg("Failed to connect to the database")
	}
	events.TriggerEvent("db-connected")
	gcutil.LogInfo().
		Str("DBtype", systemCritical.DBtype).
		Str("DBhost", systemCritical.DBhost).
		Msg("Connected to database")

	if err = gcsql.CheckAndInitializeDatabase(systemCritical.DBtype, dbVersionStr); err != nil {
		cleanup()
		gcutil.LogFatal().Err(err).Msg("Failed to initialize the database")
	}
	events.TriggerEvent("db-initialized")
	if err = gcsql.ResetViews(); err != nil {
		fatalEv.Err(err).Caller().Msg("Failed resetting SQL views")
	}

	parseCommandLine(fatalEv)
	serverutil.InitMinifier()
	siteCfg := config.GetSiteConfig()
	if err = geoip.SetupGeoIP(siteCfg.GeoIPType, siteCfg.GeoIPOptions); err != nil {
		fatalEv.Err(err).Caller().Msg("Unable to initialize GeoIP")
	}
	posting.InitCaptcha()

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

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	posting.InitPosting()
	defer events.TriggerEvent("shutdown")
	manage.InitManagePages()
	go initServer()
	<-sc
}

func parseCommandLine(fatalEv *zerolog.Event) {
	var newstaff string
	var delstaff string
	var rebuild string
	var rank int
	var err error
	flag.StringVar(&newstaff, "newstaff", "", "<newusername>:<newpassword>")
	flag.StringVar(&delstaff, "delstaff", "", "<username>")
	flag.StringVar(&rebuild, "rebuild", "", "accepted values are boards,front,js, or all")
	flag.IntVar(&rank, "rank", 0, "New staff member rank, to be used with -newstaff or -delstaff")
	flag.Parse()

	rebuildFlag := buildNone
	switch rebuild {
	case "boards":
		rebuildFlag = buildBoards
	case "front":
		rebuildFlag = buildFront
	case "js":
		rebuildFlag = buildJS
	case "all":
		rebuildFlag = buildAll
	}
	if rebuildFlag > 0 {
		startupRebuild(rebuildFlag, fatalEv)
	}

	if newstaff != "" {
		arr := strings.Split(newstaff, ":")
		if len(arr) < 2 || delstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		if _, err = gcsql.NewStaff(arr[0], arr[1], rank); err != nil {
			fatalEv.Err(err).Caller().
				Str("source", "commandLine").
				Str("username", arr[0]).
				Msg("Failed creating new staff account")
		}
		gcutil.LogInfo().
			Str("source", "commandLine").
			Str("username", arr[0]).
			Msg("New staff account created")
		os.Exit(0)
	}
	if delstaff != "" {
		if newstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		fmt.Printf("Are you sure you want to delete the staff account %q? [y/N]: ", delstaff)

		var answer string
		fmt.Scanln(&answer)
		answer = strings.ToLower(answer)
		if answer == "y" || answer == "yes" {
			if err = gcsql.DeactivateStaff(delstaff); err != nil {
				fatalEv.Err(err).Caller().
					Str("source", "commandLine").
					Str("username", delstaff).
					Msg("Unable to delete staff account")
			}
			gcutil.LogInfo().Str("newStaff", delstaff).Msg("Staff account deleted")
		} else {
			fmt.Println("Not deleting.")
		}
	}
}
