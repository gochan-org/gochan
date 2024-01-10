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

const (
	// if the database version is less than this, it is assumed to be out of date, and the schema needs to be adjusted
	latestDatabaseVersion = 3
)

type GCDatabaseUpdater struct {
	options *common.MigrationOptions
	db      *gcsql.GCDB
}

func (dbu *GCDatabaseUpdater) Init(options *common.MigrationOptions) error {
	dbu.options = options
	criticalCfg := config.GetSystemCriticalConfig()
	var err error
	dbu.db, err = gcsql.Open(
		criticalCfg.DBhost, criticalCfg.DBtype, criticalCfg.DBname, criticalCfg.DBusername, criticalCfg.DBpassword,
		criticalCfg.DBprefix,
	)
	return err
}

func (dbu *GCDatabaseUpdater) IsMigrated() (bool, error) {
	var currentDatabaseVersion int
	err := dbu.db.QueryRowSQL(`SELECT version FROM DBPREFIXdatabase_version WHERE component = 'gochan'`, nil,
		[]any{&currentDatabaseVersion})
	if err != nil {
		return false, err
	}
	if currentDatabaseVersion == latestDatabaseVersion {
		return true, nil
	}
	if currentDatabaseVersion > latestDatabaseVersion {
		return false, fmt.Errorf("database layout is ahead of current version (%d), target version: %d",
			currentDatabaseVersion, latestDatabaseVersion)
	}
	return false, nil
}

func (dbu *GCDatabaseUpdater) MigrateDB() (bool, error) {
	migrated, err := dbu.IsMigrated()
	if migrated || err != nil {
		return migrated, err
	}

	criticalConfig := config.GetSystemCriticalConfig()
	ctx := context.Background()
	tx, err := dbu.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	switch criticalConfig.DBtype {
	case "mysql":
		err = updateMysqlDB(dbu.db, tx, &criticalConfig)
	case "postgres":
		err = updatePostgresDB(dbu.db, tx, &criticalConfig)
	case "sqlite3":
		err = updateSqliteDB(dbu.db, tx, &criticalConfig)
	}
	if err != nil {
		return false, err
	}

	query := `UPDATE DBPREFIXdatabase_version SET version = ? WHERE component = 'gochan'`
	_, err = dbu.db.ExecTxSQL(tx, query, latestDatabaseVersion)
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
