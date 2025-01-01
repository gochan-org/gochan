package pre2021

import (
	"path"
	"testing"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	sqlite3DBPath = "tools/gochan-pre2021.sqlite3db" // relative to gochan project root
)

func TestMigrateToNewDB(t *testing.T) {
	dir, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, common.InitTestMigrationLog(t)) {
		return
	}
	dbPath := path.Join(dir, sqlite3DBPath)

	oldSQLConfig := config.SQLConfig{
		DBtype:     "sqlite3",
		DBname:     path.Base(dbPath),
		DBhost:     dbPath,
		DBprefix:   "gc_",
		DBusername: "gochan",
		DBpassword: "password",
	}
	migrator := Pre2021Migrator{
		config: Pre2021Config{
			SQLConfig: oldSQLConfig,
		},
	}
	outDir := t.TempDir()

	config.SetTestDBConfig("sqlite3", path.Join(outDir, "gochan-migrated.sqlite3db"), "gochan-migrated.sqlite3db", "gochan", "password", "gc_")
	sqlConfig := config.GetSQLConfig()

	if !assert.NoError(t, gcsql.ConnectToDB(&sqlConfig)) {
		return
	}
	if !assert.NoError(t, gcsql.CheckAndInitializeDatabase("sqlite3", "4")) {
		return
	}

	assert.NoError(t, migrator.migrateBoardsToNewDB())
}
