package gcsql

import (
	"log"
	"os"
	"testing"

	"github.com/gochan-org/gochan/pkg/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func populateTestDB() error {
	var err error
	if err = initDB("../../initdb_sqlite3.sql"); err != nil {
		return err
	}
	err = CreateDefaultBoardIfNoneExist()
	if err != nil {
		return err
	}
	err = CreateDefaultAdminIfNoStaff()
	return err
}

func TestDatabaseVersion(t *testing.T) {
	dbVersion, dbFlag, err := GetCompleteDatabaseVersion()
	if err != nil {
		t.Fatalf(err.Error())
	}
	if dbVersion < 1 {
		t.Fatalf("Database version should be > 0 (got %d)", dbVersion)
	}
	if dbFlag != DBClean && dbFlag != DBUpToDate {
		t.Fatalf("Got an unexpected DB flag (%#x), should be either clean or up to date", dbFlag)
	}
}

func TestMain(m *testing.M) {
	log.SetFlags(0)
	config.InitConfig("3.2.0")

	os.Remove("./testdata/gochantest.db")
	var err error
	gcdb, err = Open("./testdata/gochantest.db", "sqlite3", "gochan", "gochan", "gochan", "gc_")
	if err != nil {
		panic(err.Error())
	}
	defer gcdb.Close()

	if err = populateTestDB(); err != nil {
		panic(err.Error())
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
