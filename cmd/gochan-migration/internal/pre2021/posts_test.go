package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigratePostsToNewDB(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	var numThreads int
	if !assert.NoError(t, migrator.db.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXposts WHERE parentid = 0 AND deleted_timestamp IS NULL", nil, []any{&numThreads}), "Failed to get number of threads") {
		t.FailNow()
	}
	assert.Equal(t, 2, numThreads, "Expected to have two threads pre-migration")

	if !assert.NoError(t, migrator.MigratePosts()) {
		t.FailNow()
	}

	var numMigratedThreads int
	if !assert.NoError(t, gcsql.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXthreads", nil, []any{&numMigratedThreads}), "Failed to get number of migrated threads") {
		t.FailNow()
	}
	assert.Equal(t, 2, numMigratedThreads, "Expected to have three migrated threads")

	var locked bool
	if !assert.NoError(t, gcsql.QueryRowSQL("SELECT locked FROM DBPREFIXthreads WHERE id = 1", nil, []any{&locked})) {
		t.FailNow()
	}
	assert.True(t, locked, "Expected thread ID 1 to be locked")

	// make sure deleted posts and threads weren't migrated
	var numDeleted int
	assert.NoError(t, gcsql.QueryRowSQL("SELECT COUNT(*) FROM DBPREFIXposts WHERE message_raw LIKE '%deleted%' OR is_deleted", nil, []any{&numDeleted}))
	assert.Zero(t, numDeleted, "Expected no deleted threads to be migrated")
}
