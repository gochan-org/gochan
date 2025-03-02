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
	fmt.Printf("Starting gochan v%s\n", versionStr)
	config.InitConfig(versionStr)

	uid, gid := config.GetUser()
	systemCritical := config.GetSystemCriticalConfig()
	err := gcutil.InitLogs(systemCritical.LogDir, true, uid, gid)
	if err != nil {
		fmt.Println("Error opening logs:", err.Error())
		cleanup()
		os.Exit(1)
	}

	testIP := os.Getenv("GC_TESTIP")
	if testIP != "" {
		gcutil.LogInfo().Str("GC_TESTIP", testIP).
			Msg("Custom testing IP address set from environment variable")
	}

	if err = gcplugin.LoadPlugins(systemCritical.Plugins); err != nil {
		cleanup()
		gcutil.LogFatal().Err(err).Msg("failed loading plugins")
	}

	events.TriggerEvent("startup")

	if err = gcsql.ConnectToDB(&systemCritical.SQLConfig); err != nil {
		fmt.Println("Failed to connect to the database:", err.Error())
		cleanup()
		gcutil.LogFatal().Err(err).Msg("Failed to connect to the database")
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
	events.RegisterEvent([]string{"db-views-reset"}, func(_ string, _ ...any) error {
		gcutil.LogInfo().Msg("SQL views reset")
		return nil
	})
	if err = gcsql.ResetViews(); err != nil {
		gcutil.LogFatal().Err(err).Caller().Msg("Failed resetting SQL views")
	}

	parseCommandLine()
	serverutil.InitMinifier()
	siteCfg := config.GetSiteConfig()
	if err = geoip.SetupGeoIP(siteCfg.GeoIPType, siteCfg.GeoIPOptions); err != nil {
		cleanup()
		gcutil.LogFatal().Err(err).Msg("Unable to initialize GeoIP")
	}
	posting.InitCaptcha()

	if err = gctemplates.InitTemplates(); err != nil {
		fmt.Println("Failed initializing templates:", err.Error())
		cleanup()
		gcutil.LogFatal().Err(err).Send()
	}

	for _, board := range gcsql.AllBoards {
		if _, err = board.DeleteOldThreads(); err != nil {
			cleanup()
			gcutil.LogFatal().Err(err).Caller().
				Str("board", board.Dir).
				Msg("Failed deleting old threads")
		}
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	posting.InitPosting()
	if err = gcutil.InitLogs(systemCritical.LogDir, systemCritical.Verbose, uid, gid); err != nil {
		fmt.Println("Error opening logs:", err.Error())
		cleanup()
		os.Exit(1) // skipcq: CRT-D0011
	}
	defer events.TriggerEvent("shutdown")
	manage.InitManagePages()
	go initServer()
	<-sc
}

func parseCommandLine() {
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
		startupRebuild(rebuildFlag)
	}

	if newstaff != "" {
		arr := strings.Split(newstaff, ":")
		if len(arr) < 2 || delstaff != "" {
			flag.Usage()
			os.Exit(1)
		}
		fmt.Printf("Creating new staff: %q, with password: %q and rank: %d from command line", arr[0], arr[1], rank)
		if _, err = gcsql.NewStaff(arr[0], arr[1], rank); err != nil {
			fmt.Printf("Failed creating new staff account for %q: %s\n", arr[0], err.Error())
			gcutil.LogFatal().Err(err).Caller().
				Str("staff", "add").
				Str("source", "commandLine").
				Str("username", arr[0]).
				Msg("Failed creating new staff account")
		}
		gcutil.LogInfo().
			Str("staff", "add").
			Str("source", "commandLine").
			Str("username", arr[0]).
			Msg("New staff account created")
		fmt.Printf("New staff account created: %s\n", arr[0])
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
				fmt.Printf("Error deleting %q: %s", delstaff, err.Error())
				gcutil.LogFatal().Str("staff", "delete").Err(err).Send()
			}
			gcutil.LogInfo().Str("newStaff", delstaff).Send()
		} else {
			fmt.Println("Not deleting.")
		}
	}
}
