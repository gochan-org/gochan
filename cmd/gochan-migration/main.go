package main

import (
	"flag"
	"log"
	"os"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/gcupdate"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/pkg/config"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	banner = `Welcome to the gochan database migration tool for gochan %s!
This migration tool is currently unstable, and will likely go through
several changes before it can be considered "stable", so make sure you check
the README and/or the -h command line flag before you use it.

`
	migrateCompleteTxt = `Database migration successful!
To migrate the uploads for each board, move or copy the uploads to /path/to/gochan/document/root/<boardname>/src/
Then copy the thumbnails to /path/to/gochan/documentroot/<boardname>/thumb/
Then start the gochan server and go to http://yoursite/manage/rebuildall to generate the html files
for the threads and board pages`

	allowedDirActions = "Valid values are noaction, copy, and move (defaults to noaction if unset)"
)

var (
	versionStr   string
	dbVersionStr string
)

func main() {
	var options common.MigrationOptions
	var dirAction string
	var updateDB bool

	log.SetFlags(0)
	flag.BoolVar(&updateDB, "updatedb", false, "If this is set, gochan-migrate will check, and if needed, update gochan's database schema")
	flag.StringVar(&options.ChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.OldChanConfig, "oldconfig", "", "The path to the old chan's configuration file")
	// flag.StringVar(&dirAction, "diraction", "", "Action taken on each board directory after it has been migrated. "+allowedDirActions)
	flag.Parse()

	config.InitConfig(versionStr)

	if !updateDB && (options.ChanType == "" || options.OldChanConfig == "") {
		flag.PrintDefaults()
		log.Fatal("Missing required oldchan value")
		return
	} else if updateDB {
		options.ChanType = "gcupdate"
	}
	switch dirAction {
	case "":
		fallthrough
	case "noaction":
		options.DirAction = common.DirNoAction
	case "copy":
		options.DirAction = common.DirCopy
	case "move":
		options.DirAction = common.DirMove
	default:
		log.Fatalln("Invalid diraction value. " + allowedDirActions)
	}

	log.Printf(banner, versionStr)
	var migrator common.DBMigrator
	switch options.ChanType {
	case "gcupdate":
		migrator = &gcupdate.GCDatabaseUpdater{}
	case "pre2021":
		migrator = &pre2021.Pre2021Migrator{}
	case "kusabax":
		fallthrough
	case "tinyboard":
		fallthrough
	default:
		log.Fatalf(
			"Unsupported chan type %q, Currently only pre2021 database migration is supported\n",
			options.ChanType)
		return
	}
	config.InitConfig(versionStr)
	var err error
	if !updateDB {
		systemCritical := config.GetSystemCriticalConfig()
		err = gcsql.ConnectToDB(
			systemCritical.DBhost, systemCritical.DBtype, systemCritical.DBname,
			systemCritical.DBusername, systemCritical.DBpassword, systemCritical.DBprefix)
		if err != nil {
			log.Fatalf("Failed to connect to the database: %s", err.Error())
		}
		if err = gcsql.CheckAndInitializeDatabase(systemCritical.DBtype); err != nil {
			log.Fatalf("Failed to initialize the database: %s", err.Error())
		}
		defer gcsql.Close()
	}

	if err = migrator.Init(&options); err != nil {
		log.Fatalf("Unable to initialize %s migrator: %s\n",
			options.ChanType, err.Error())
		return
	}
	defer migrator.Close()
	var migrated bool

	if migrated, err = migrator.MigrateDB(); err != nil {
		log.Fatalln("Error migrating database:", err.Error())
	}
	if migrated {
		log.Println("Database is already migrated")
		os.Exit(0)
	}
	log.Println(migrateCompleteTxt)
}
