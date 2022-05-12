package main

import (
	"flag"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
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

	fatalLogFlags     = gclog.LFatal | gclog.LErrorLog | gclog.LStdLog
	allowedDirActions = "Valid values are noaction, copy, and move (defaults to noaction if unset)"
)

var (
	versionStr string
)

func main() {
	var options common.MigrationOptions

	config.InitConfig(versionStr)
	var dirAction string
	flag.StringVar(&options.ChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.OldChanConfig, "oldconfig", "", "The path to the old chan's configuration file")
	/* flag.StringVar(&dirAction, "diraction", "",
	"Action taken on each board directory after it has been migrated. "+allowedDirActions) */

	flag.Parse()

	if options.ChanType == "" || options.OldChanConfig == "" {
		flag.PrintDefaults()
		gclog.Println(fatalLogFlags, "Missing required oldchan value")
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
		gclog.Println(fatalLogFlags, "Invalid diraction value. "+allowedDirActions)
	}

	gclog.Printf(gclog.LAccessLog, banner, versionStr)
	var migrator common.DBMigrator
	switch options.ChanType {
	case "pre2021":
		migrator = &pre2021.Pre2021Migrator{}
	case "kusabax":
		fallthrough
	case "tinyboard":
		fallthrough
	default:
		gclog.Printf(fatalLogFlags,
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
		gclog.Printf(fatalLogFlags,
			"Unable to initialize %s migrator: %s\n", options.ChanType, err.Error())
		return
	}
	defer migrator.Close()
	var migrated bool

	if migrated, err = migrator.MigrateDB(); err != nil {
		gclog.Println(fatalLogFlags, "Error migrating database:", err.Error())
		return
	}
	if migrated {
		gclog.Printf(gclog.LStdLog|gclog.LAccessLog, "Database is already migrated")
		return
	}
	gclog.Println(gclog.LStdLog, migrateCompleteTxt)
}
