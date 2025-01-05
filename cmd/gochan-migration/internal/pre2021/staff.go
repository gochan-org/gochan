package pre2021

import (
	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type migrationStaff struct {
	gcsql.Staff

	oldID    int
	boardIDs []int
}

func (*Pre2021Migrator) migrateStaffInPlace() error {
	err := common.NewMigrationError("pre2021", "migrateSectionsInPlace not yet implemented")
	common.LogError().Err(err).Caller().Msg("Failed to migrate sections")
	return err
}

func (*Pre2021Migrator) migrateStaffToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	err := common.NewMigrationError("pre2021", "migrateStaffToNewDB not yet implemented")
	errEv.Err(err).Caller().Msg("Failed to migrate sections")

	return err
}

func (m *Pre2021Migrator) MigrateStaff() error {
	if m.IsMigratingInPlace() {
		return m.migrateStaffInPlace()
	}
	return m.migrateStaffToNewDB()
}
