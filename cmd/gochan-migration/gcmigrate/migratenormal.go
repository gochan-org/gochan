package gcmigrate

import (
	"fmt"

	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func versionHandler(foundDatabaseVersion, targetDatabaseVersion int) error {
	if foundDatabaseVersion < targetDatabaseVersion {
		for foundDatabaseVersion < targetDatabaseVersion {
			gclog.Printf(gclog.LStdLog, "Migrating databasefrom version %v to version %v", foundDatabaseVersion, foundDatabaseVersion+1)
			err := migrations[foundDatabaseVersion]()
			if err != nil {
				gclog.Print(gcsql.FatalSQLFlags, "Failedmigration: ", err.Error())
				return err
			}
			gclog.Print(gclog.LStdLog, "Finished migrating database to version %v", foundDatabaseVersion+1)
			foundDatabaseVersion++
		}
		return nil
	}
	if foundDatabaseVersion == targetDatabaseVersion {
		gclog.Print(gclog.LStdLog, "Database up to date")
		return nil
	}
	gclog.Print(gclog.LFatal, "Found database version higher than target version.\nFound version: %v\n Target version: %v", foundDatabaseVersion, targetDatabaseVersion)
	return nil
}

func checkMigrationsExist(currentVersion, target int) error {
	for i := currentVersion; i < target; i++ {
		if _, ok := migrations[i]; !ok {
			return fmt.Errorf("This version of the migrator does not contain a migration from version %v to %v, please upgrade the migrator", currentVersion, target)
		}
	}
	return nil
}

var migrations = map[int]func() error{}
