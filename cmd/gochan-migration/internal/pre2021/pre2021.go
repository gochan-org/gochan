// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type Pre2021Config struct {
	config.SQLConfig
	DocumentRoot string
}

type Pre2021Migrator struct {
	db      *gcsql.GCDB
	options *common.MigrationOptions
	config  Pre2021Config

	migrationUser *gcsql.Staff
	boards        []migrationBoard
	sections      []migrationSection
	staff         []migrationStaff
}

// IsMigratingInPlace implements common.DBMigrator.
func (m *Pre2021Migrator) IsMigratingInPlace() bool {
	sqlConfig := config.GetSQLConfig()
	return m.config.DBname == sqlConfig.DBname && m.config.DBhost == sqlConfig.DBhost && m.config.DBprefix == sqlConfig.DBprefix
}

func (m *Pre2021Migrator) readConfig() error {
	ba, err := os.ReadFile(m.options.OldChanConfig)
	if err != nil {
		return err
	}
	m.config.SQLConfig = config.GetSQLConfig()
	return json.Unmarshal(ba, &m.config)
}

func (m *Pre2021Migrator) Init(options *common.MigrationOptions) error {
	m.options = options
	var err error

	if err = m.readConfig(); err != nil {
		return err
	}

	m.db, err = gcsql.Open(&m.config.SQLConfig)
	return err
}

func (m *Pre2021Migrator) IsMigrated() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.config.DBTimeoutSeconds)*time.Second)
	defer cancel()
	var sqlConfig config.SQLConfig
	if m.IsMigratingInPlace() {
		sqlConfig = config.GetSQLConfig()
	} else {
		sqlConfig = m.config.SQLConfig
	}
	return common.TableExists(ctx, m.db, nil, "DBPREFIXdatabase_version", &sqlConfig)
}

func (m *Pre2021Migrator) renameTablesForInPlace() error {
	var err error
	errEv := common.LogError()
	defer errEv.Discard()
	if _, err = m.db.Exec(nil, "DROP TABLE DBPREFIXinfo"); err != nil {
		errEv.Err(err).Caller().Msg("Error dropping info table")
		return err
	}
	for _, table := range renameTables {
		if _, err = m.db.Exec(nil, fmt.Sprintf(renameTableStatementTemplate, table, table)); err != nil {
			errEv.Caller().Err(err).
				Str("table", table).
				Msg("Error renaming table")
			return err
		}
	}

	if err = gcsql.CheckAndInitializeDatabase(m.config.DBtype, true); err != nil {
		errEv.Caller().Err(err).Msg("Error checking and initializing database")
		return err
	}

	if err = m.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Error closing database")
		return err
	}
	m.config.SQLConfig.DBprefix = "_tmp_" + m.config.DBprefix
	m.db, err = gcsql.Open(&m.config.SQLConfig)
	if err != nil {
		errEv.Err(err).Caller().Msg("Error reopening database with new prefix")
		return err
	}

	common.LogInfo().Msg("Renamed tables for in-place migration")
	return err
}

func (m *Pre2021Migrator) MigrateDB() (bool, error) {
	errEv := common.LogError()
	defer errEv.Discard()
	migrated, err := m.IsMigrated()
	if err != nil {
		errEv.Caller().Err(err).Msg("Error checking if database is migrated")
		return false, err
	}
	if migrated {
		return true, nil
	}

	if m.IsMigratingInPlace() {
		if err = m.renameTablesForInPlace(); err != nil {
			return false, err
		}
	}

	if err := m.MigrateBoards(); err != nil {
		return false, err
	}
	common.LogInfo().Msg("Migrated boards successfully")

	if err = m.MigratePosts(); err != nil {
		return false, err
	}
	common.LogInfo().Msg("Migrated threads, posts, and uploads successfully")

	if err = m.MigrateStaff(); err != nil {
		return false, err
	}
	common.LogInfo().Msg("Migrated staff successfully")

	if err = m.MigrateBans(); err != nil {
		return false, err
	}
	common.LogInfo().Msg("Migrated bans and filters successfully")

	if err = m.MigrateAnnouncements(); err != nil {
		return false, err
	}
	common.LogInfo().Msg("Migrated staff announcements successfully")

	return false, nil
}

func (m *Pre2021Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
