package pre2021

import (
	"context"
	"testing"
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

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

	assert.Equal(t, len(migratedBoards), 3, "Expected updated boards list to have three boards")
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
	assert.Equal(t, 1, testBoard.ID)
	assert.Equal(t, "Testing Board", testBoard.Title)
	assert.Equal(t, "Board for testing pre-2021 migration", testBoard.Subtitle)
	assert.Equal(t, "Board for testing pre-2021 migration description", testBoard.Description)
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

func TestMigrateBoardsInPlace(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, true)
	if !assert.True(t, migrator.IsMigratingInPlace(), "This test should be migrating in place") {
		t.FailNow()
	}

	if !assert.NoError(t, migrator.MigrateBoards()) {
		t.FailNow()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(migrator.config.DBTimeoutSeconds))
	defer cancel()
	var uri string
	var sectionID int
	assert.NoError(t, migrator.db.QueryRowContextSQL(ctx, nil, "SELECT uri FROM DBPREFIXboards WHERE dir = ?", []any{"test"}, []any{&uri}))
	assert.NoError(t, migrator.db.QueryRowContextSQL(ctx, nil, "SELECT section_id FROM DBPREFIXboards WHERE dir = ?", []any{"test"}, []any{&sectionID}))
	assert.Equal(t, "", uri)
	assert.Greater(t, sectionID, 0)
}
