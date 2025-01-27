package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigrateAnnouncementsToNewDB(t *testing.T) {
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

	validateAnnouncementMigration(t)
}

func TestMigrateAnnouncementsInPlace(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, true)
	if !assert.True(t, migrator.IsMigratingInPlace(), "This test should be migrating in place") {
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
	validateAnnouncementMigration(t)
}

func validateAnnouncementMigration(t *testing.T) {
	var numAnnouncements int
	assert.NoError(t, gcsql.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXannouncements WHERE staff_id > 0", nil, []any{&numAnnouncements}))
	assert.Equal(t, 2, numAnnouncements, "Expected to have two announcement")
}
