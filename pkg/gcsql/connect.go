package gcsql

import (
	"database/sql"
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
)

// ConnectToDB initializes the database connection and exits if there are any errors
func ConnectToDB(cfg *config.SQLConfig) error {
	var err error
	gcdb, err = Open(cfg)
	return err
}

// SetDB sets the global database connection (mainly used by gochan-migration)
func SetDB(db *GCDB) {
	gcdb = db
}

func SetTestingDB(dbDriver string, dbName string, dbPrefix string, db *sql.DB) (err error) {
	testutil.PanicIfNotTest()
	sqlConfig := config.GetSQLConfig()
	if sqlConfig.DBname == "" {
		return ErrNotConnected
	}

	gcdb, err = setupDBConn(&config.SQLConfig{
		DBtype:               dbDriver,
		DBhost:               "localhost",
		DBname:               dbName,
		DBusername:           "gochan",
		DBpassword:           "gochan",
		DBprefix:             dbPrefix,
		DBTimeoutSeconds:     config.DefaultSQLTimeout,
		DBMaxOpenConnections: config.DefaultSQLMaxConns,
		DBMaxIdleConnections: config.DefaultSQLMaxConns,
		DBConnMaxLifetimeMin: config.DefaultSQLConnMaxLifetimeMin,
	})
	if err != nil {
		return
	}
	gcdb.db = db
	return
}

// RunSQLFile cuts a given sql file into individual statements and runs it.
func RunSQLFile(path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(gcdb.replacer.Replace(sqlStr), ";")

	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = ExecTxSQL(tx, statement); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func ResetFunctions() error {
	if gcdb == nil {
		return ErrNotConnected
	}
	var functionsFile string
	switch gcdb.driver {
	case "mysql":
		functionsFile = findSQLFile("functions_mysql.sql")
	case "postgres", "postgresql":
		functionsFile = findSQLFile("functions_postgres.sql")
	case "sqlite3", "sqlite3-inet6":
		return nil // handled internally
	default:
		return ErrUnsupportedDB
	}

	ba, err := os.ReadFile(functionsFile)
	if err != nil {
		return err
	}
	if len(ba) == 0 {
		return errors.New("functions file is empty")
	}

	// we have to use the underlying sql.DB because the wrapper assumes that the incoming query is a single line, single statement
	if _, err = gcdb.db.Exec(string(ba)); err != nil {
		return err
	}

	_, err, recovered := events.TriggerEvent("db-functions-reset")
	if err != nil {
		return err
	}
	if recovered {
		return errors.New("recovered from panic while running reset functions event")
	}
	return nil
}
