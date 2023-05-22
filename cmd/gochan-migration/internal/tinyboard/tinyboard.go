package tinyboard

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	unimplemented = common.NewMigrationError("tinyboard", "unimplemented")
)

type TinyBoardMigrator struct {
	db      *gcsql.GCDB
	options *common.MigrationOptions
}

func (m *TinyBoardMigrator) Init(options *common.MigrationOptions) error {
	m.options = options
	return unimplemented
}

func (m *TinyBoardMigrator) MigrateDB() error {
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

func (*TinyBoardMigrator) MigrateBoards() error {
	return nil
}

func (*TinyBoardMigrator) MigratePosts() error {
	return nil
}

func (*TinyBoardMigrator) MigrateStaff(_ string) error {
	return nil
}

func (*TinyBoardMigrator) MigrateBans() error {
	return nil
}

func (*TinyBoardMigrator) MigrateAnnouncements() error {
	return nil
}

func (m *TinyBoardMigrator) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
