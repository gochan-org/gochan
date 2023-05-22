package kusabax

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	unimplemented = common.NewMigrationError("tinyboard", "unimplemented")
)

type KusabaXMigrator struct {
	db      *gcsql.GCDB
	options *common.MigrationOptions
}

func (m *KusabaXMigrator) Init(options *common.MigrationOptions) error {
	m.options = options
	return unimplemented
}

func (m *KusabaXMigrator) MigrateDB() error {
	var err error
	if err = m.MigrateBoards(); err != nil {
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
	return unimplemented
}

func (*KusabaXMigrator) MigrateBoards() error {
	return nil
}

func (*KusabaXMigrator) MigratePosts() error {
	return nil
}

func (*KusabaXMigrator) MigrateStaff(_ string) error {
	return nil
}

func (*KusabaXMigrator) MigrateBans() error {
	return nil
}

func (*KusabaXMigrator) MigrateAnnouncements() error {
	return nil
}

func (m *KusabaXMigrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
