package kusabax

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
)

var (
	unimplemented = common.NewMigrationError("tinyboard", "unimplemented")
)

type KusabaXMigrator struct {
	// db      *gcsql.GCDB
	// options common.DBOptions
}

func (m *KusabaXMigrator) Init(options common.DBOptions) error {
	return unimplemented
}

func (m *KusabaXMigrator) MigrateDB() error {
	return unimplemented
}

func (m *KusabaXMigrator) Close() error {
	/* if m.db != nil {
		return m.db.Close()
	} */
	return nil
}
