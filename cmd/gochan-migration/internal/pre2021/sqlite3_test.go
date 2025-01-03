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

func setupMigrationTest(t *testing.T, outDir string) *Pre2021Migrator {
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
	migratedDBPath := path.Join(outDir, "gochan-migrated.sqlite3db")
	t.Log("Migrated DB path:", migratedDBPath)

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
	migrator := setupMigrationTest(t, outDir)
	assert.NoError(t, gcsql.ResetBoardSectionArrays())

	numBoards := len(gcsql.AllBoards)
	numSections := len(gcsql.AllSections)

	assert.Equal(t, 1, numBoards, "Expected to have 1 board pre-migration (/test/ is automatically created during provisioning)")
	assert.Equal(t, 1, numSections, "Expected to have 1 section pre-migration (Main is automatically created during provisioning)")

	assert.NoError(t, migrator.migrateBoardsToNewDB())

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

	hiddenBoard, err := gcsql.GetBoardFromDir("hidden")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	t.Logf("Hidden board section ID: %d", hiddenBoard.SectionID)

	t.Log("Number of sections:", len(migratedSections))
	for _, section := range migratedSections {
		t.Logf("Section ID %d: %#v", section.ID, section)
	}
	hiddenSection, err := gcsql.GetSectionFromID(hiddenBoard.SectionID)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "Hidden section", hiddenSection.Name, "Expected /hidden/ board's section to have name 'Hidden'")
	assert.True(t, hiddenSection.Hidden, "Expected Hidden section to be hidden")
}
