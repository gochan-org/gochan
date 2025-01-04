package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigrateBansToNewDB(t *testing.T) {
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

	if !assert.NoError(t, migrator.MigrateBans()) {
		t.FailNow()
	}
	bans, err := gcsql.GetIPBans(0, 200, false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, 4, len(bans), "Expected to have 4 valid bans")

	var numInvalidBans int
	assert.NoError(t, gcsql.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXip_ban WHERE message = ?", []any{"Full ban on 8.8.0.0/16"}, []any{&numInvalidBans}))
	assert.Equal(t, 0, numInvalidBans, "Expected the invalid test to not be migrated")
}
