package pre2021

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestMigratePosts(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	var numThreads int
	if !assert.NoError(t, migrator.db.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXposts WHERE parentid = 0 AND deleted_timestamp IS NULL", nil, []any{&numThreads}), "Failed to get number of threads") {
		t.FailNow()
	}
	assert.Equal(t, 2, numThreads, "Expected to have two threads pre-migration")

	if !assert.NoError(t, migrator.MigratePosts()) {
		t.FailNow()
	}
	validatePostMigration(t)
}

func validatePostMigration(t *testing.T) {
	var numThreads int
	if !assert.NoError(t, gcsql.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXthreads", nil, []any{&numThreads}), "Failed to get number of threads") {
		t.FailNow()
	}
	assert.Equal(t, 2, numThreads, "Expected to have two threads pre-migration")

	var numUploadPosts int
	assert.NoError(t, gcsql.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXfiles", nil, []any{&numUploadPosts}))
	assert.Equal(t, 1, numUploadPosts, "Expected to have 1 upload post")

	var ip string
	assert.NoError(t, gcsql.QueryRow(nil, "SELECT IP_NTOA FROM DBPREFIXposts WHERE id = 1", nil, []any{&ip}))
	assert.Equal(t, "192.168.56.1", ip, "Expected to have the correct IP address")

	var numMigratedThreads int
	if !assert.NoError(t, gcsql.QueryRow(nil, "SELECT COUNT(*) FROM DBPREFIXthreads", nil, []any{&numMigratedThreads}), "Failed to get number of migrated threads") {
		t.FailNow()
	}
	assert.Equal(t, 2, numMigratedThreads, "Expected to have three migrated threads")

	var locked bool
	if !assert.NoError(t, gcsql.QueryRow(nil, "SELECT locked FROM DBPREFIXthreads WHERE id = 1", nil, []any{&locked})) {
		t.FailNow()
	}
	assert.True(t, locked, "Expected thread ID 1 to be locked")
}
