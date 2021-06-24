// used for migrating pre-refactor gochan databases to the new schema
package pre2021

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type Pre2021Migrator struct {
	db      *gcsql.GCDB
	options common.DBOptions
}

func (m *Pre2021Migrator) Init(options common.DBOptions) error {
	m.options = options
	var err error
	m.db, err = gcsql.Open(
		m.options.Host, m.options.DBType, "", m.options.Username, m.options.Password, "",
	)
	return err
}

func (m *Pre2021Migrator) MigrateDB() error {
	return nil
}

func (m *Pre2021Migrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
