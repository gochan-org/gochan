package main

import (
	"flag"
	"log"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/pkg/config"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	banner = `Welcome to the gochan database migration tool for gochan %s!
This migration tool is currently very unstable, and will likely go through
several changes before it can be considered "stable", so make sure you check
the README and/or the -h command line flag before you use it.

`
	migrateCompleteTxt = `Database migration successful!
To migrate the uploads for each board, move or copy the uploads to /path/to/gochan/document/root/<boardname>/src/
Then copy the thumbnails to /path/to/gochan/documentroot/<boardname>/thumb/
Then start the gochan server and go to http://yoursite/manage?action=rebuildall to generate the html files
for the threads and board pages`

	allowedDirActions = "Valid values are noaction, copy, and move (defaults to noaction if unset)"
)

var (
	versionStr string
)

func main() {
	var options common.MigrationOptions
	var dirAction string

	log.SetFlags(0)
	config.InitConfig(versionStr)
	flag.StringVar(&options.ChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.OldChanConfig, "oldconfig", "", "The path to the old chan's configuration file")
	/* flag.StringVar(&dirAction, "diraction", "",
	"Action taken on each board directory after it has been migrated. "+allowedDirActions) */

	flag.Parse()

	if options.ChanType == "" || options.OldChanConfig == "" {
		flag.PrintDefaults()
		log.Fatal("Missing required oldchan value")
		return
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
	systemCritical := config.GetSystemCriticalConfig()

	gcsql.ConnectToDB(
		systemCritical.DBhost, systemCritical.DBtype, systemCritical.DBname,
		systemCritical.DBusername, systemCritical.DBpassword, systemCritical.DBprefix)
	gcsql.CheckAndInitializeDatabase(systemCritical.DBtype)
	defer gcsql.Close()

	err := migrator.Init(&options)
	if err != nil {
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
		log.Fatalf("Database is already migrated")
	}
	log.Println(migrateCompleteTxt)
}
