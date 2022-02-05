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
	DBModernButBehind
	DBUpToDate
	DBModernButAhead

	targetDatabaseVersion = 1
)

var (
	// ErrInvalidVersion is used when the db contains a database_version table
	// but zero or more than one versions were found
	ErrInvalidVersion = errors.New("database contains database_version table but zero or more than one versions were found")
)

// GetCompleteDatabaseVersion checks the database for any versions and errors that may exist.
// If a version is found, execute the version check. Otherwise check for deprecated info
// If no deprecated info is found, check if any databases exist prefixed with config.DBprefix
// if no prefixed databases exist, assume this is a new installation
func GetCompleteDatabaseVersion() (dbVersion, dbFlag int, err error) {
	versionTableExists, err := doesTableExist("database_version")
	if err != nil {
		return 0, 0, err
	}
	if versionTableExists {
		databaseVersion, versionError := getDatabaseVersion(GochanVersionKeyConstant)
		if versionError != nil {
			return 0, 0, ErrInvalidVersion
		}
		if databaseVersion < targetDatabaseVersion {
			return databaseVersion, DBModernButBehind, nil
		}
		if databaseVersion > targetDatabaseVersion {
			return databaseVersion, DBModernButAhead, nil
		}
		return databaseVersion, DBUpToDate, nil
	}
	isOldDesign, err := doesTableExist("info")
	if err != nil {
		return 0, 0, err
	}
	if isOldDesign {
		return 0, DBIsPreApril, nil
	}
	//No old or current database versioning tables found.
	if config.GetSystemCriticalConfig().DBprefix != "" {
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
	good := false
	defer func() {
		if !good {
			return
		}
		if err = tmpSqlAdjust(); err != nil {
			gclog.Print(FatalSQLFlags, "Failed updating database structure: ", err.Error())
			return
		}
	}()
	switch versionFlag {
	case DBIsPreApril:
		fallthrough
	case DBModernButBehind:
		gclog.Printf(FatalSQLFlags,
			"Database layout is deprecated. Please run gochan-migrate. Target version is %d", targetDatabaseVersion) //TODO give exact command
	case DBClean:
		buildNewDatabase(dbType)
		good = true
		return
	case DBUpToDate:
		good = true
		return
	case DBCorrupted:
		gclog.Println(FatalSQLFlags,
			"Database contains gochan prefixed tables but is missing versioning tables. Database is possible corrupted. Please contact the devs for help.")
		return
	case DBModernButAhead:
		gclog.Printf(gclog.LFatal,
			"Database layout is ahead of current version. Current version %d, target version: %d.\n"+
				"Are you running an old gochan version?", dbVersion, targetDatabaseVersion)
		return
	default:
		gclog.Printf(FatalSQLFlags,
			"Failed to initialise database: Checkdatabase, none of paths matched. Should never be executed. Check for outcome of GetCompleteDatabaseVersion()")
		return
	}
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
