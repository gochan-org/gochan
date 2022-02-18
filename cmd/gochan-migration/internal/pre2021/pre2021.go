// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"encoding/json"
	"io/ioutil"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

const (
	// check to see if the old db exists, if the new db exists, and the number of tables
	// in the new db
	mysqlDbInfoSQL = `SELECT
		(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?) AS olddb,
		(SELECT SCHEMA_NAME FROM INFORMATION_SCHEMA.SCHEMATA WHERE SCHEMA_NAME = ?) as newdb,
		(SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ?) as num_tables`
)

type Pre2021Config struct {
	DBtype       string
	DBhost       string
	DBname       string
	DBusername   string
	DBpassword   string
	DBprefix     string
	DocumentRoot string
}

type Pre2021Migrator struct {
	db      *gcsql.GCDB
	options common.MigrationOptions
	config  Pre2021Config

	posts     []postTable
	oldBoards map[int]string // map[boardid]dir
	newBoards map[int]string // map[board]dir
}

func (m *Pre2021Migrator) readConfig() error {
	ba, err := ioutil.ReadFile(m.options.OldChanConfig)
	if err != nil {
		return err
	}
	return json.Unmarshal(ba, &m.config)
}

func (m *Pre2021Migrator) Init(options common.MigrationOptions) error {
	m.options = options
	var err error
	if err = m.readConfig(); err != nil {
		return err
	}
	m.db, err = gcsql.Open(
		m.config.DBhost, m.config.DBtype, m.config.DBname, m.config.DBusername,
		m.config.DBpassword, m.config.DBprefix)
	return err
}

func (m *Pre2021Migrator) MigrateDB() error {
	var err error
	if err := m.MigrateBoards(); err != nil {
		return err
	}
	if err = m.MigratePosts(); err != nil {
		return err
	}
	if err = m.MigrateStaff("password"); err != nil {
		return err
	}
	if err = m.MigrateBans(); err != nil {
		return err
	}
	if err = m.MigrateAnnouncements(); err != nil {
		return err
	}

	return nil
}

func (m *Pre2021Migrator) MigrateStaff(password string) error {
	return nil
}

func (m *Pre2021Migrator) MigrateBans() error {
	return nil
}

func (m *Pre2021Migrator) MigrateAnnouncements() error {
	return nil
}

func (m *Pre2021Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
