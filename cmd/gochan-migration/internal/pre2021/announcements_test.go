package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigrateAnnouncements(t *testing.T) {
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

	if !assert.NoError(t, migrator.MigrateAnnouncements()) {
		t.FailNow()
	}

	var numAnnouncements int
	assert.NoError(t, gcsql.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXannouncements WHERE staff_id > 0", nil, []any{&numAnnouncements}))
	assert.Equal(t, 2, numAnnouncements, "Expected to have two announcement")
}
