package gcupdate

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
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
		err = updateMysqlDB(ctx, dbu.db, tx, &sqlConfig)
	case "postgres":
		err = updatePostgresDB(ctx, dbu.db, tx, &sqlConfig)
	case "sqlite3":
		err = updateSqliteDB(ctx, dbu.db, tx, &sqlConfig)
	}
	if err != nil {
		return false, err
	}

	// commit the transaction and start a new one (to avoid deadlocks)
	if err = tx.Commit(); err != nil {
		return false, err
	}

	if err = ctx.Err(); err != nil {
		return false, err
	}

	filterTableExists, err := common.TableExists(ctx, dbu.db, nil, "DBPREFIXfilters", &sqlConfig)
	if err != nil {
		return false, err
	}

	if !filterTableExists {
		// DBPREFIXfilters not found, create it and migrate data from DBPREFIXfile_bans, DBPREFIXfilename_bans, and DBPREFIXusername_bans,
		if err = common.AddFilterTables(ctx, dbu.db, nil, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateFileBans(ctx, dbu.db, nil, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateFilenameBans(ctx, dbu.db, nil, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateUsernameBans(ctx, dbu.db, nil, &sqlConfig); err != nil {
			return false, err
		}
		if err = common.MigrateWordfilters(ctx, dbu.db, nil, &sqlConfig); err != nil {
			return false, err
		}
	}

	query := `UPDATE DBPREFIXdatabase_version SET version = ? WHERE component = 'gochan'`
	_, err = dbu.db.ExecContextSQL(ctx, nil, query, dbu.TargetDBVer)
	if err != nil {
		return false, err
	}
	// return false, tx.Commit()
	return false, nil
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
