package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigrateBans(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigratePosts()) {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateStaff()) {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBans()) {
		t.FailNow()
	}

	validateBanMigration(t)
}

func validateBanMigration(t *testing.T) {
	bans, err := gcsql.GetIPBans(0, 200, false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 6, len(bans), "Expected to have 4 valid bans")
	assert.NotZero(t, bans[0].StaffID, "Expected ban staff ID field to be set")

	var numInvalidBans int
	assert.NoError(t, gcsql.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXip_ban WHERE message = ?", []any{"Full ban on 8.8.0.0/16"}, []any{&numInvalidBans}))
	assert.Equal(t, 0, numInvalidBans, "Expected the invalid test to not be migrated")

	filters, err := gcsql.GetAllFilters(gcsql.TrueOrFalse)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 1, len(filters))
	conditions, err := filters[0].Conditions()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 3, len(conditions), "Expected filter to have three conditions")
}
