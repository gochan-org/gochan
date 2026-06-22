package pre2021

import (
	"io"
	"os"
	"path"
	"testing"

	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	sqlite3DBDir = "tools/" // relative to gochan project root
)

func setupMigrationTest(t *testing.T, outDir string, migrateInPlace bool) *Pre2021Migrator {
	dir, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, common.InitTestMigrationLog(t)) {
		t.FailNow()
	}

	dbName := "gochan-pre2021.db"
	dbHost := path.Join(dir, sqlite3DBDir, dbName)
	migratedDBName := "gochan-migrated.db"
	migratedDBHost := path.Join(outDir, migratedDBName)

	if migrateInPlace {
		oldDbFile, err := os.Open(dbHost)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer oldDbFile.Close()

		newDbFile, err := os.OpenFile(migratedDBHost, os.O_CREATE|os.O_WRONLY, 0600)
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
		dbHost = migratedDBHost
		dbName = migratedDBName
	}

	oldSQLConfig := config.SQLConfig{
		DBtype:           "sqlite3",
		DBname:           dbName,
		DBhost:           dbHost,
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
	migrator.db, err = gcsql.Open(&oldSQLConfig)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	config.SetTestDBConfig("sqlite3", migratedDBHost, migratedDBName, "gochan", "password", "gc_")
	sqlConfig := config.GetSQLConfig()

	if !assert.NoError(t, gcsql.ConnectToDB(&sqlConfig)) {
		t.FailNow()
	}
	if !migrateInPlace {
		// if migrating in place, this shouldn't be done until after migration is complete
		if !assert.NoError(t, gcsql.CheckAndInitializeDatabase(sqlConfig.DBtype, true)) {
			t.FailNow()
		}
	}

	return migrator
}

// TODO: Add test cases for MySQL and Postgres, skipping if connection fails (assuming server isn't running)
func TestPre2021MigrationToNewDB(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, false)
	if !assert.False(t, migrator.IsMigratingInPlace(), "This test should not be migrating in place") {
		t.FailNow()
	}
	migrated, err := migrator.MigrateDB()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.False(t, migrated)

	validateBoardMigration(t)
	validatePostMigration(t)
	validateBanMigration(t)
	validateStaffMigration(t)
	validateAppealMigration(t)
}

// TODO: add test cases for MySQL and Postgres, as above
func TestPre2021MigrationInPlace(t *testing.T) {
	outDir := t.TempDir()
	migrator := setupMigrationTest(t, outDir, true)
	if !assert.True(t, migrator.IsMigratingInPlace(), "This test should be migrating in place") {
		t.FailNow()
	}
	migrated, err := migrator.MigrateDB()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.False(t, migrated)

	validateBoardMigration(t)
	validatePostMigration(t)
	validateBanMigration(t)
	validateStaffMigration(t)
	validateAppealMigration(t)
}
