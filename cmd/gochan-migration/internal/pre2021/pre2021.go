// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"encoding/json"
	"fmt"
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
	DBtype     string
	DBhost     string
	DBname     string
	DBusername string
	DBpassword string
	DBprefix   string
}

type Pre2021Migrator struct {
	db      *gcsql.GCDB
	options common.MigrationOptions
	config  Pre2021Config
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
	err := m.readConfig()
	if err != nil {
		return err
	}
	m.db, err = gcsql.Open(
		m.config.DBhost, m.config.DBtype, m.config.DBname, m.config.DBusername,
		m.config.DBpassword, m.config.DBprefix)
	return err
}

func (m *Pre2021Migrator) MigrateDB() error {
	// select id,thread_id,name,tripcode,email,subject,message from gc_posts;
	rows, err := m.db.QuerySQL(`SELECT id,parentid,name,tripcode,email,subject,message FROM DBPREFIXposts`)
	if err != nil {
		return err
	}
	var id int
	var thread int
	var name string
	var tripcode string
	var email string
	var subject string
	var message string
	for rows.Next() {
		if err = rows.Scan(&id, &thread, &name, &tripcode, &email, &subject, &message); err != nil {
			return err
		}
		fmt.Printf(
			"Post #%d in %d by %s!%s, email %q, subject %q, message: %q\n",
			id, thread, name, tripcode, email, subject, message,
		)
	}

	return nil
}

func (m *Pre2021Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return gcsql.Close()
}
