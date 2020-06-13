package gcmigrate

import (
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func versionHandler(foundDatabaseVersion int, targetDatabaseVersion int) error {
	if foundDatabaseVersion < targetDatabaseVersion {
		for foundDatabaseVersion < targetDatabaseVersion {
			gclog.Printf(gclog.LStdLog, "Migrating database from version %v to version %v", foundDatabaseVersion, foundDatabaseVersion+1)
			err := migrations[foundDatabaseVersion]()
			if err != nil {
				gclog.Print(gcsql.FatalSQLFlags, "Failed migration: ", err.Error())
				return err
			}
			gclog.Printf(gclog.LStdLog, "Finished migrating database to version %v", foundDatabaseVersion+1)
			foundDatabaseVersion++
		}
		return nil
	}
	if foundDatabaseVersion == targetDatabaseVersion {
		gclog.Print(gclog.LStdLog, "Database up to date")
		return nil
	}
	gclog.Printf(gclog.LFatal, "Found database version higher than target version.\nFound version: %v\n Target version: %v", foundDatabaseVersion, targetDatabaseVersion)
	return nil
}

var migrations = map[int]func() error{}
