package gcsql

import (
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

//Check if it can find version
//if version found, excecute version check
// if version < current version, launch migration funcs in order
// if version == current version, do nothing
// if version > current version, throw panic
//If no version found, check for old info thing
//if found, excecute achaic db migration script. Then run the check again (will match against version 1)
//If not found, check if any databases excist with a gochan prefix as per config (?)
//If found, unknown or corrupted database, as for confirmation to continue
//If no old gochan tables found, excecute new install build

const targetDatabaseVersion = 1

var notImplemented = gcutil.NewError("Not implemented", false)

func handleVersioning(dbType string) *gcutil.GcError {
	versionTableExists, err := doesTableExist("database_version")
	if err != nil {
		return err
	}
	if versionTableExists {
		databaseVersion, versionError := getDatabaseVersion()
		if versionError != nil {
			gclog.Println(fatalSQLFlags, "Database contains database_version table but zero or more than one versions were found")
			return nil
		}
		return versionHandler(databaseVersion)
	}
	isOldDesign, err := doesTableExist("info")
	if err != nil {
		return err
	}
	if isOldDesign {
		return migratePreApril2020Database()
	}
	//No old or current database versioning tables found.
	if config.Config.DBprefix != "" {
		//Check if any gochan tables exist
		gochanTableExists, err := doesGochanPrefixTableExist()
		if err != nil {
			return err
		}
		if gochanTableExists {
			gclog.Println(fatalSQLFlags, "Database contains gochan prefixed tables. Database is possible corrupted.")
			return nil
		}
	}
	//At this point, assume new database
	buildNewDatabase(dbType)
	return nil
}

func buildNewDatabase(dbType string) {
	var err *gcutil.GcError
	if err = initDB("initdb_" + dbType + ".sql"); err != nil {
		gclog.Print(fatalSQLFlags, "Failed initializing DB: ", err.Error())
	}
	err = CreateDefaultBoardIfNoneExist()
	if err != nil {
		gclog.Print(fatalSQLFlags, "Failed creating default board: ", err.Message)
	}
	err = CreateDefaultAdminIfNoStaff()
	if err != nil {
		gclog.Print(fatalSQLFlags, "Failed creating default admin account: ", err.Error())
	}
}

func versionHandler(foundDatabaseVersion int) *gcutil.GcError {
	if foundDatabaseVersion < targetDatabaseVersion {
		for foundDatabaseVersion < targetDatabaseVersion {
			gclog.Print(gclog.LStdLog, "Migrating database from version %v to version %v", foundDatabaseVersion, foundDatabaseVersion+1)
			err := migrations[foundDatabaseVersion]()
			if err != nil {
				gclog.Print(fatalSQLFlags, "Failed migration: ", err.Error())
				return err
			}
			gclog.Print(gclog.LStdLog, "Finished migrating database to version %v", foundDatabaseVersion+1)
			foundDatabaseVersion++
		}
		return nil
	}
	if foundDatabaseVersion == targetDatabaseVersion {
		gclog.Print(gclog.LStdLog, "Database already populated")
		return nil
	}
	gclog.Println(gclog.LFatal, "Found database version higher than target version.\nFound version: %v\n Target version: %v", foundDatabaseVersion, targetDatabaseVersion)
	return nil

}

func migratePreApril2020Database() *gcutil.GcError {
	return notImplemented
}

var migrations = map[int]func() *gcutil.GcError{}
