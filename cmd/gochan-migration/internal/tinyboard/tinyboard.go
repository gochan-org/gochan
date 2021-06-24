package tinyboard

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
)

var (
	unimplemented = common.NewMigrationError("tinyboard", "unimplemented")
)

type TinyBoardMigrator struct {
	// db      *gcsql.GCDB
	// options common.DBOptions
}

func (m *TinyBoardMigrator) Init(options common.DBOptions) error {
	return unimplemented
}

func (m *TinyBoardMigrator) MigrateDB() error {
	return unimplemented
}

func (m *TinyBoardMigrator) Close() error {
	/* if m.db != nil {
		return m.db.Close()
	} */
	return nil
}
