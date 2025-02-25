package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/gcupdate"
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/pre2021"
	"github.com/gochan-org/gochan/pkg/config"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	versionStr   string
	migrator     common.DBMigrator
	dbVersionStr string
)

func cleanup() {
	var err error
	exitCode := 0
	if migrator != nil {
		if err = migrator.Close(); err != nil {
			common.LogError().Err(err).Caller().Msg("Error closing migrator")
			exitCode = 1
		}
	}
	if err = gcsql.Close(); err != nil {
		common.LogError().Err(err).Caller().Msg("Error closing SQL connection")
		exitCode = 1
	}
	os.Exit(exitCode)
}

func main() {
	var options common.MigrationOptions
	var updateDB bool

	flag.BoolVar(&updateDB, "updatedb", false, "If this is set, gochan-migrate will check, and if needed, update gochan's database schema")
	flag.StringVar(&options.ChanType, "oldchan", "", "The imageboard we are migrating from (currently only pre2021 is supported, but more are coming")
	flag.StringVar(&options.OldChanConfig, "oldconfig", "", "The path to the old chan's configuration file")
	flag.Parse()

	config.InitConfig(versionStr)
	err := common.InitMigrationLog()
	if err != nil {
		log.Fatalln("Unable to initialize migration log:", err.Error())
	}
	fatalEv := common.LogFatal()
	defer func() {
		cleanup()
		fatalEv.Discard()
	}()

	if !updateDB {
		if options.ChanType == "" {
			flag.PrintDefaults()
			fatalEv.Msg("Missing required oldchan value")
		} else if options.OldChanConfig == "" {
			flag.PrintDefaults()
			fatalEv.Msg("Missing required oldconfig value")
		}
	} else if updateDB {
		options.ChanType = "gcupdate"
	}
	fatalEv.Str("chanType", options.ChanType)

	switch options.ChanType {
	case "gcupdate":
		targetDBVer, err := strconv.Atoi(dbVersionStr)
		if err != nil {
			fatalEv.Err(err).Caller().Msg("Invalid database version string, unable to parse as integer")
		}
		migrator = &gcupdate.GCDatabaseUpdater{
			TargetDBVer: targetDBVer,
		}
	case "pre2021":
		migrator = &pre2021.Pre2021Migrator{}
	case "kusabax":
		fallthrough
	case "tinyboard":
		fallthrough
	default:
		fatalEv.Msg("Unsupported chan type, Currently only pre2021 database migration is supported")
	}
	migratingInPlace := migrator.IsMigratingInPlace()
	common.LogInfo().
		Str("oldChanType", options.ChanType).
		Str("oldChanConfig", options.OldChanConfig).
		Bool("migratingInPlace", migratingInPlace).
		Msg("Starting database migration")

	config.InitConfig(versionStr)
	sqlCfg := config.GetSQLConfig()
	if migratingInPlace && sqlCfg.DBtype == "sqlite3" && !updateDB {
		common.LogWarning().
			Str("dbType", sqlCfg.DBtype).
			Bool("migrateInPlace", migratingInPlace).
			Msg("SQLite has limitations with table column changes")
	}
	if !migratingInPlace {
		err = gcsql.ConnectToDB(&sqlCfg)
		if err != nil {
			fatalEv.Err(err).Caller().Msg("Failed to connect to the database")
		}
		if err = gcsql.CheckAndInitializeDatabase(sqlCfg.DBtype, dbVersionStr); err != nil {
			fatalEv.Err(err).Caller().Msg("Unable to initialize the database")
		}
	}

	if err = migrator.Init(&options); err != nil {
		fatalEv.Err(err).Caller().Msg("Unable to initialize migrator")
	}

	var migrated bool
	if migrated, err = migrator.MigrateDB(); err != nil {
		fatalEv.Msg("Unable to migrate database")
	}
	if migrated {
		common.LogWarning().Msg("Database is already migrated")
	} else {
		common.LogInfo().Str("chanType", options.ChanType).Msg("Database migration complete")
	}
}
