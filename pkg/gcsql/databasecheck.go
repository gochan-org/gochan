package gcsql

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
)

//Check if it can find version
//if version found, excecute version check
//If no version found, check for old info thing
//If not found, check if any databases excist with a gochan prefix as per config (?)
//If no old gochan tables found, excecute new install build

const targetDatabaseVersion = 1

//GetCompleteDatabaseVersion Checks the database for any versions and errors that may exist
func GetCompleteDatabaseVersion() (isPreAprilVersion bool, databaseVersion int, isCorrupted bool, cleanDatabase bool, err error) {
	versionTableExists, err := doesTableExist("database_version")
	if err != nil {
		return false, 0, false, false, err
	}
	if versionTableExists {
		databaseVersion, versionError := getDatabaseVersion()
		if versionError != nil {
			return false, 0, false, false, errors.New("Database contains database_version table but zero or more than one versions were found")
		}
		return false, databaseVersion, false, false, nil
	}
	isOldDesign, err := doesTableExist("info")
	if err != nil {
		return false, 0, false, false, err
	}
	if isOldDesign {
		return true, 0, false, false, nil
	}
	//No old or current database versioning tables found.
	if config.Config.DBprefix != "" {
		//Check if any gochan tables exist
		gochanTableExists, err := doesGochanPrefixTableExist()
		if err != nil {
			return false, 0, false, false, err
		}
		if gochanTableExists {
			return false, 0, true, false, nil
		}
	}
	return false, 0, false, true, nil
}

//CheckAndInitializeDatabase checks the validity of the database and initialises it if it is empty
func CheckAndInitializeDatabase(dbType string) {
	isPreAprilVersion, databaseVersion, isCorrupted, cleanDatabase, err := GetCompleteDatabaseVersion()
	if err != nil {
		gclog.Printf(FatalSQLFlags, "Failed to initialise database: %s", err.Error())
	}
	if cleanDatabase {
		buildNewDatabase(dbType)
		return
	}
	if isCorrupted {
		gclog.Println(FatalSQLFlags, "Database contains gochan prefixed tables but is missing versioning tables. Database is possible corrupted. Please contact the devs for help.")
	}
	if isPreAprilVersion || databaseVersion < targetDatabaseVersion {
		gclog.Printf(gclog.LFatal, "Database layout is deprecated. Please run gochan-migrate. Target version is %s", targetDatabaseVersion) //TODO give exact command
	}
	if databaseVersion == targetDatabaseVersion {
		gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Existing database is valid...")
		return
	}
	if databaseVersion > targetDatabaseVersion {
		gclog.Printf(gclog.LFatal, `Database layout is ahead of current version. Current version %s, target version: %s. 
		Are you running an old gochan version?`, databaseVersion, targetDatabaseVersion)
	}
	gclog.Printf(FatalSQLFlags, "Failed to initialise database: Checkdatabase, none of paths matched. Should never be executed. Check for outcome of GetCompleteDatabaseVersion()")
}

func buildNewDatabase(dbType string) {
	var err error
	if err = initDB("initdb_" + dbType + ".sql"); err != nil {
		gclog.Print(FatalSQLFlags, "Failed initializing DB: ", err.Error())
	}
	err = CreateDefaultBoardIfNoneExist()
	if err != nil {
		gclog.Print(FatalSQLFlags, "Failed creating default board: ", err.Error())
	}
	err = CreateDefaultAdminIfNoStaff()
	if err != nil {
		gclog.Print(FatalSQLFlags, "Failed creating default admin account: ", err.Error())
	}
	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Finished building database...")
}
