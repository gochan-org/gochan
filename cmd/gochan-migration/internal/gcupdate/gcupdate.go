package gcupdate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

type GCDatabaseUpdater struct {
	options *common.MigrationOptions
	db      *gcsql.GCDB
	// if the database version is less than TargetDBVer, it is assumed to be out of date, and the schema needs to be adjusted.
	// It is expected to be set by the build script
	TargetDBVer int
}

func (dbu *GCDatabaseUpdater) Init(options *common.MigrationOptions) error {
	dbu.options = options
	sqlCfg := config.GetSQLConfig()
	var err error
	dbu.db, err = gcsql.Open(&sqlCfg)
	return err
}

func (dbu *GCDatabaseUpdater) IsMigrated() (bool, error) {
	var currentDatabaseVersion int
	err := dbu.db.QueryRowSQL(`SELECT version FROM DBPREFIXdatabase_version WHERE component = 'gochan'`, nil,
		[]any{&currentDatabaseVersion})
	if err != nil {
		return false, err
	}
	if currentDatabaseVersion == dbu.TargetDBVer {
		return true, nil
	}
	if currentDatabaseVersion > dbu.TargetDBVer {
		return false, fmt.Errorf("database layout is ahead of current version (%d), target version: %d",
			currentDatabaseVersion, dbu.TargetDBVer)
	}
	return false, nil
}

func (dbu *GCDatabaseUpdater) MigrateDB() (bool, error) {
	migrated, err := dbu.IsMigrated()
	if migrated || err != nil {
		return migrated, err
	}

	sqlConfig := config.GetSQLConfig()
	ctx := context.Background()
	tx, err := dbu.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	switch sqlConfig.DBtype {
	case "mysql":
		err = updateMysqlDB(dbu.db, tx, &sqlConfig)
	case "postgres":
		err = updatePostgresDB(dbu.db, tx, &sqlConfig)
	case "sqlite3":
		err = updateSqliteDB(dbu.db, tx, &sqlConfig)
	}
	if err != nil {
		return false, err
	}

	filterTableExists, err := common.TableExists(dbu.db, nil, "DBPREFIXfilters", &sqlConfig)
	if err != nil {
		return false, err
	}

	if !filterTableExists {
		if err = common.AddFilterTables(dbu.db, ctx, tx, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateFileBans(dbu.db, ctx, tx, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateFilenameBans(dbu.db, ctx, tx, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateUsernameBans(dbu.db, ctx, tx, &sqlConfig); err != nil {
			return false, err
		}
	}

	query := `UPDATE DBPREFIXdatabase_version SET version = ? WHERE component = 'gochan'`
	_, err = dbu.db.ExecTxSQL(tx, query, dbu.TargetDBVer)
	if err != nil {
		return false, err
	}
	return false, tx.Commit()
}

func (*GCDatabaseUpdater) MigrateBoards() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigratePosts() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateStaff(_ string) error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateBans() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateAnnouncements() error {
	return gcutil.ErrNotImplemented
}

func (dbu *GCDatabaseUpdater) Close() error {
	if dbu.db != nil {
		return dbu.db.Close()
	}
	return nil
}
