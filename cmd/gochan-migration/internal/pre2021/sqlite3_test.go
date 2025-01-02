package pre2021

import (
	"path"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	sqlite3DBPath = "tools/gochan-pre2021.sqlite3db" // relative to gochan project root
)

func setupMigrationTest(t *testing.T) *Pre2021Migrator {
	dir, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	dbPath := path.Join(dir, sqlite3DBPath)

	oldSQLConfig := config.SQLConfig{
		DBtype:           "sqlite3",
		DBname:           path.Base(dbPath),
		DBhost:           dbPath,
		DBprefix:         "gc_",
		DBusername:       "gochan",
		DBpassword:       "password",
		DBTimeoutSeconds: 600,
	}
	migrator := &Pre2021Migrator{
		config: Pre2021Config{
			SQLConfig: oldSQLConfig,
		},
	}
	db, err := gcsql.Open(&oldSQLConfig)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	migrator.db = db

	outDir := t.TempDir()
	config.SetTestDBConfig("sqlite3", path.Join(outDir, "gochan-migrated.sqlite3db"), "gochan-migrated.sqlite3db", "gochan", "password", "gc_")
	sqlConfig := config.GetSQLConfig()
	sqlConfig.DBTimeoutSeconds = 600

	if !assert.NoError(t, gcsql.ConnectToDB(&sqlConfig)) {
		t.FailNow()
	}
	if !assert.NoError(t, gcsql.CheckAndInitializeDatabase("sqlite3", "4")) {
		t.FailNow()
	}

	return migrator
}

func TestMigrateToNewDB(t *testing.T) {
	migrator := setupMigrationTest(t)

	assert.NoError(t, migrator.migrateBoardsToNewDB())

	newBoards, err := gcsql.GetAllBoards(false)
	if !assert.NoError(t, err) {
		return
	}
	assert.GreaterOrEqual(t, len(newBoards), 2, "Expected new boards list to have at least 2 boards") // old DB has 2 boards, /test/ and /hidden/

	hiddenBoard, err := gcsql.GetBoardFromDir("hidden")
	if !assert.NoError(t, err) {
		return
	}
	t.Logf("Hidden board section ID: %d", hiddenBoard.SectionID)
	hiddenSection, err := gcsql.GetSectionFromID(hiddenBoard.SectionID)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, "Hidden", hiddenSection.Name, "Expected Hidden section to have name 'Hidden'")
	assert.True(t, hiddenSection.Hidden, "Expected Hidden section to be hidden")
}
