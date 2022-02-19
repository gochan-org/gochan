// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"encoding/json"
	"io/ioutil"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
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

func (m *Pre2021Migrator) IsMigrated() (bool, error) {
	var migrated bool
	var err error
	var query string
	switch m.config.DBtype {
	case "mysql":
		fallthrough
	case "postgres":
		query = `SELECT COUNT(*) > 0 FROM INFORMATION_SCHEMA.TABLES
		WHERE TABLE_NAME = ? AND TABLE_SCHEMA = ?`
	}
	if err = m.db.QueryRowSQL(query,
		[]interface{}{m.config.DBprefix + "migrated", m.config.DBname},
		[]interface{}{&migrated}); err != nil {
		return migrated, err
	}
	return migrated, err
}

func (m *Pre2021Migrator) MigrateDB() (bool, error) {
	migrated, err := m.IsMigrated()
	if err != nil {
		return false, err
	}
	if migrated {
		// db is already migrated, stop
		return true, nil
	}

	if err := m.MigrateBoards(); err != nil {
		return false, err
	}
	if err = m.MigratePosts(); err != nil {
		return false, err
	}
	if err = m.MigrateStaff("password"); err != nil {
		return false, err
	}
	if err = m.MigrateBans(); err != nil {
		return false, err
	}
	if err = m.MigrateAnnouncements(); err != nil {
		return false, err
	}

	return true, nil
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
