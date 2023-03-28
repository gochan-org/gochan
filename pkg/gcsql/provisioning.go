package gcsql

import (
	"errors"
	"fmt"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
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
	ErrInvalidVersion   = errors.New("database contains database_version table but zero or more than one versions were found")
	ErrCorruptedDB      = errors.New("database contains gochan prefixed tables but is missing versioning tables (possibly corrupted)")
	ErrDeprecatedDB     = errors.New("database layout is deprecated, please run gochan-migrate")
	ErrInvalidDBVersion = errors.New("invalid version flag returned by GetCompleteDatabaseVersion()")
)

func initDB(initFile string) error {
	filePath := gcutil.FindResource(initFile,
		"./sql/"+initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return fmt.Errorf("missing SQL database initialization file (%s), please reinstall gochan", initFile)
	}
	return RunSQLFile(filePath)
}

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
		databaseVersion, versionError := getDatabaseVersion(gochanVersionKeyConstant)
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

// CheckAndInitializeDatabase checks the validity of the database and initialises it if it is empty
func CheckAndInitializeDatabase(dbType string) error {
	dbVersion, versionFlag, err := GetCompleteDatabaseVersion()
	if err != nil {
		return err
	}
	switch versionFlag {
	case DBIsPreApril:
		fallthrough
	case DBModernButBehind:
		err = ErrDeprecatedDB
	case DBClean:
		err = buildNewDatabase(dbType)
	case DBUpToDate:
		err = nil
	case DBCorrupted:
		err = ErrCorruptedDB
	case DBModernButAhead:
		// Uer might be running an old gochan version
		err = fmt.Errorf("database layout is ahead of current version (%d), target version: %d", dbVersion, targetDatabaseVersion)
	default:
		err = ErrInvalidDBVersion
	}
	if err != nil {
		return err
	}
	return tmpSqlAdjust()
}

func buildNewDatabase(dbType string) error {
	var err error
	if err = initDB("initdb_" + dbType + ".sql"); err != nil {
		return err
	}
	if err = createDefaultAdminIfNoStaff(); err != nil {
		return errors.New("failed creating default admin account: " + err.Error())
	}
	if err = createDefaultBoardIfNoneExist(); err != nil {
		return errors.New("failed creating default board if non already exists: " + err.Error())
	}
	return nil
}
