package gcsql

import (
	"log"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
)

var (
	sqm sqlmock.Sqlmock
)

func connectTest() (err error) {
	gcdb = &GCDB{
		driver: "sqlmock",
		replacer: strings.NewReplacer(
			"DBNAME", "gochan",
			"DBPREFIX", "gc_",
			"\n", " "),
	}

	gcdb.db, sqm, err = sqlmock.New()
	if err != nil {
		return err
	}
	return err
}

func prepTestQueryString(str string) string {
	return regexp.QuoteMeta(strings.Replace(str, "\n", " ", -1))
}

func populateTestSchema() error {
	var err error
	sqm.ExpectPrepare(prepTestQueryString(`INSERT INTO gc_database_version(component, version)
VALUES('gochan', 1);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))
	if _, err = ExecSQL(`INSERT INTO DBPREFIXdatabase_version(component, version)
VALUES('gochan', 1);`); err != nil {
		return err
	}

	return sqm.ExpectationsWereMet()
}

func TestDatabaseVersion(t *testing.T) {
	sqm.ExpectPrepare(prepTestQueryString(
		`SELECT component,version FROM gc_database_version WHERE component = ?;`,
	)).ExpectQuery().WithArgs("gochan").WillReturnRows(
		sqlmock.NewRows([]string{"component", "version"}).AddRow("gochan", 1),
	)

	var component string
	var version int
	err := QueryRowSQL(`SELECT component,version FROM DBPREFIXdatabase_version WHERE component = ?;`,
		interfaceSlice("gochan"),
		interfaceSlice(&component, &version))
	if err != nil {
		t.Fatalf(err.Error())
	}
	if err = sqm.ExpectationsWereMet(); err != nil {
		t.Fatal(err.Error())
	}
	if version < 1 {
		t.Fatalf("Component %q has version: %d", component, version)
	}
}

func TestMain(m *testing.M) {
	log.SetFlags(0)
	config.InitConfig("3.2.0")

	err := connectTest()
	if err != nil {
		log.Fatalln("Failed setting up sqlmock db:", err.Error())
	}
	defer gcdb.Close()

	if err = createMockSchema(); err != nil {
		log.Fatalln("Failed setting up sqlmock db tables:", err.Error())
	}

	if err = populateTestSchema(); err != nil {
		log.Fatalln("Failed populating test schema:", err.Error())
	}

	exitCode := m.Run()

	os.Exit(exitCode)
}
