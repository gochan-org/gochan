package pre2021

import (
	"io"
	"os"
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

func setupMigrationTest(t *testing.T, outDir string, migrateInPlace bool) *Pre2021Migrator {
	dir, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	dbPath := path.Join(dir, sqlite3DBPath)
	if migrateInPlace {
		oldDbFile, err := os.Open(dbPath)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer oldDbFile.Close()

		newDbFile, err := os.OpenFile(path.Join(outDir, "gochan-pre2021.sqlite3db"), os.O_CREATE|os.O_RDWR, 0644)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer newDbFile.Close()

		_, err = io.Copy(newDbFile, oldDbFile)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.NoError(t, oldDbFile.Close())
		assert.NoError(t, newDbFile.Close())
		dbPath = path.Join(outDir, "gochan-pre2021.sqlite3db")
	}

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
	migratedDBPath := path.Join(outDir, "gochan-migrated.sqlite3db")

	config.SetTestDBConfig("sqlite3", migratedDBPath, path.Base(migratedDBPath), "gochan", "password", "gc_")
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

func TestMigrateBoardsToNewDB(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}
	assert.NoError(t, gcsql.ResetBoardSectionArrays())

	numBoards := len(gcsql.AllBoards)
	numSections := len(gcsql.AllSections)

	assert.Equal(t, 1, numBoards, "Expected to have 1 board pre-migration (/test/ is automatically created during provisioning)")
	assert.Equal(t, 1, numSections, "Expected to have 1 section pre-migration (Main is automatically created during provisioning)")

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	migratedBoards, err := gcsql.GetAllBoards(false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	migratedSections, err := gcsql.GetAllSections(false)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	assert.Equal(t, len(migratedBoards), 2, "Expected updated boards list to have two boards")
	assert.Equal(t, len(migratedSections), 2, "Expected updated sections list to have two sections")

	// Test migrated sections
	mainSection, err := gcsql.GetSectionFromName("Main")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "mainmigration", mainSection.Abbreviation, "Expected Main section to have updated abbreviation name 'mainmigration'")

	// Test migrated boards
	testBoard, err := gcsql.GetBoardFromDir("test")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "Testing Board", testBoard.Title)
	assert.Equal(t, "Board for testing pre-2021 migration", testBoard.Subtitle)
	testBoardSection, err := gcsql.GetSectionFromID(testBoard.SectionID)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "Main", testBoardSection.Name, "Expected /test/ board to be in Main section")

	hiddenBoard, err := gcsql.GetBoardFromDir("hidden")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "Hidden Board", hiddenBoard.Title)
}
