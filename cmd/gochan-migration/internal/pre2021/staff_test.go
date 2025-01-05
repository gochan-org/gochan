package pre2021

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMigrateStaffToNewDB(t *testing.T) {
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
}
