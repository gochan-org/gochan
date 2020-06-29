package gcmigrate

import (
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	stdFatalFlag = gclog.LStdLog | gclog.LFatal
)

//Entry runs all the migration logic until the database matches the given database version
func Entry(targetVersion int) error {
	gcsql.ConnectToDB(
		config.Config.DBhost, config.Config.DBtype, config.Config.DBname,
		config.Config.DBusername, config.Config.DBpassword, config.Config.DBprefix)

	return runMigration(targetVersion)
}

func runMigration(targetVersion int) error {
	dbVersion, dbFlags, err := gcsql.GetCompleteDatabaseVersion()
	if err != nil {
		return err
	}
	switch dbFlags {
	case gcsql.DBCorrupted:
		gclog.Println(stdFatalFlag, "Database found is corrupted, please contact the devs.")
	case gcsql.DBClean:
		gclog.Println(stdFatalFlag,
			"Database found is clean and ready for a gochan install, please run gochan to automatically setup the database.")
	case gcsql.DBIsPreApril:
		if err = checkMigrationsExist(1, targetVersion); err != nil {
			return err
		}
		gclog.Println(gclog.LStdLog, "Migrating pre april 2020 version to version 1 of modern system.")
		if err = migratePreApril2020Database(config.Config.DBtype); err != nil {
			return err
		}
		gclog.Println(gclog.LStdLog, "Finish migrating to version 1.")
		return runMigration(targetVersion)
	}

	if err = checkMigrationsExist(dbVersion, targetVersion); err != nil {
		return err
	}
	return versionHandler(dbVersion, targetVersion)
}
