package gcsql

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
)

const (
	DBIsPreApril = 1 << iota
	DBCorrupted
	DBClean

	targetDatabaseVersion = 1
)

var (
	// ErrInvalidVersion is used when the db contains a database_version table
	// but zero or more than one versions were found
	ErrInvalidVersion = errors.New("Database contains database_version table but zero or more than one versions were found")
)

// GetCompleteDatabaseVersion checks the database for any versions and errors that may exist.
// If a version is found, execute the version check. Otherwise check for deprecated info
// If no deprecated info is found, check if any databases exist prefixed with config.DBprefix
// if no prefixed databases exist, assume this is a new installation
func GetCompleteDatabaseVersion() (dbVersion int, dbFlag int, err error) {
	versionTableExists, err := doesTableExist("database_version")
	if err != nil {
		return 0, 0, err
	}
	if versionTableExists {
		databaseVersion, versionError := getDatabaseVersion()
		if versionError != nil {
			return 0, 0, ErrInvalidVersion
		}
		return databaseVersion, 0, nil
	}
	isOldDesign, err := doesTableExist("info")
	if err != nil {
		return 0, 0, err
	}
	if isOldDesign {
		return 0, DBIsPreApril, nil
	}
	//No old or current database versioning tables found.
	if config.Config.DBprefix != "" {
		//Check if any gochan tables exist
		gochanTableExists, err := doesGochanPrefixTableExist()
		if err != nil {
			return 0, 0, err
		}
		if gochanTableExists {
			return 0, DBCorrupted, nil
		}
	}
	return 0, DBClean, nil
}

//CheckAndInitializeDatabase checks the validity of the database and initialises it if it is empty
func CheckAndInitializeDatabase(dbType string) {
	dbVersion, versionFlag, err := GetCompleteDatabaseVersion()
	if err != nil {
		gclog.Printf(FatalSQLFlags, "Failed to initialise database: %s", err.Error())
	}

	switch {
	case versionFlag == DBIsPreApril:
		fallthrough
	case dbVersion < targetDatabaseVersion:
		gclog.Printf(FatalSQLFlags,
			"Database layout is deprecated. Please run gochan-migrate. Target version is %d", targetDatabaseVersion) //TODO give exact command
	case versionFlag == DBClean:
		buildNewDatabase(dbType)
		return
	case versionFlag == DBCorrupted:
		gclog.Println(FatalSQLFlags,
			"Database contains gochan prefixed tables but is missing versioning tables. Database is possible corrupted. Please contact the devs for help.")
	case dbVersion > targetDatabaseVersion:
		gclog.Printf(gclog.LFatal,
			"Database layout is ahead of current version. Current version %d, target version: %d.\n"+
				"Are you running an old gochan version?", dbVersion, targetDatabaseVersion)
	}

	gclog.Printf(FatalSQLFlags,
		"Failed to initialise database: Checkdatabase, none of paths matched. Should never be executed. Check for outcome of GetCompleteDatabaseVersion()")
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
