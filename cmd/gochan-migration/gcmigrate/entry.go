package gcmigrate

import (
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

//Entry runs all the migration logic until the database matches the given database version
func Entry(targetVersion int) error {
	gcsql.ConnectToDB(
		config.Config.DBhost, config.Config.DBtype, config.Config.DBname,
		config.Config.DBusername, config.Config.DBpassword, config.Config.DBprefix)

	isPreAprilVersion, databaseVersion, isCorrupted, cleanDatabase, err := gcsql.GetCompleteDatabaseVersion()
	if err != nil {
		return err
	}
	if isCorrupted {
		println("Database found is corrupted, please contact the devs.")
		return nil
	}
	if cleanDatabase {
		println("Database found is clean and ready for a gochan install, please run gochan to autmatically setup the database.")
		return nil
	}
	if isPreAprilVersion {
		err = checkMigrationsExist(1, targetVersion)
		if err != nil {
			return err
		}
		println("Migrating pre april 2020 version to version 1 of modern system.")
		err = migratePreApril2020Database(config.Config.DBtype)
		if err != nil {
			return err
		}
		println("Finish migrating to version 1.")
		return Entry(targetVersion)
	}
	err = checkMigrationsExist(databaseVersion, targetVersion)
	if err != nil {
		return err
	}
	return versionHandler(databaseVersion, targetVersion)
}
