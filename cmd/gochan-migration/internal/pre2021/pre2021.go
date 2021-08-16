// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	// check to see if the old db exists, if the new db exists, and the number of tables
	// in the new db
	dbInfoSQL = `SELECT
		(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?) AS olddb,
		(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?) as newdb,
		(SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?) as num_tables`
)

type Pre2021Migrator struct {
	db      *gcsql.GCDB
	options common.DBOptions
}

func (m *Pre2021Migrator) Init(options common.DBOptions) error {
	m.options = options
	var err error
	m.db, err = gcsql.Open(
		m.options.Host, m.options.DBType, "", m.options.Username,
		m.options.Password, options.TablePrefix)
	return err
}

func (m *Pre2021Migrator) MigrateDB() error {
	chkDbStmt, err := m.db.PrepareSQL(dbInfoSQL)
	if err != nil {
		return err
	}
	defer chkDbStmt.Close()
	var olddb []byte
	var newdb []byte
	var numTables int

	if err = chkDbStmt.QueryRow(m.options.OldDBName, m.options.NewDBName, m.options.NewDBName).Scan(&olddb, &newdb, &numTables); err != nil {
		return common.NewMigrationError("pre2021", err.Error())
	}
	if olddb == nil {
		return common.NewMigrationError("pre2021", "old database doesn't exist")
	}
	if newdb == nil {
		return common.NewMigrationError("pre2021", "new database doesn't exist")
	}
	if numTables > 0 {
		return common.NewMigrationError("pre2021", "new database must be empty")
	}
	gcsql.ConnectToDB(
		m.options.Host, m.options.DBType, m.options.NewDBName,
		m.options.Username, m.options.Password, m.options.TablePrefix)
	cfg := config.GetSystemCriticalConfig()
	gcsql.CheckAndInitializeDatabase(cfg.DBtype)

	GetPosts(m.db)

	return nil
}

func (m *Pre2021Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return gcsql.Close()
}
