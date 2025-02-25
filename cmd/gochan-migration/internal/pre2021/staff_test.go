package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigrateStaff(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateStaff()) {
		t.FailNow()
	}
	validateStaffMigration(t)
}

func validateStaffMigration(t *testing.T) {
	migratedAdmin, err := gcsql.GetStaffByUsername("migratedadmin", true)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 3, migratedAdmin.Rank)
}
