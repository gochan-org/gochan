package gcsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	UnsupportedSQLVersionMsg = `Received syntax error while preparing a SQL string.
	This means that either there is a bug in gochan's code (hopefully not) or that you are using an unsupported MySQL/PostgreSQL version.
	Before reporting an error, make sure that you are using the up to date version of your selected SQL server.
	Error text: %s`
	mysqlConnStr    = "%s:%s@tcp(%s)/%s?parseTime=true&collation=utf8mb4_unicode_ci"
	postgresConnStr = "postgres://%s:%s@%s/%s?sslmode=disable"
)

type GCDB struct {
	db       *sql.DB
	connStr  string
	driver   string
	replacer *strings.Replacer
	// nilTimestamp string
}

func (db *GCDB) ConnectionString() string {
	return db.connStr
}

func (db *GCDB) Connection() *sql.DB {
	return db.db
}

func (db *GCDB) SQLDriver() string {
	return db.driver
}

/* func (db *GCDB) NilSQLTimestamp() string {
	return db.nilTimestamp
} */

func (db *GCDB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *GCDB) PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	var prepared string
	var err error
	if prepared, err = PrepareSQLString(query, db); err != nil {
		return nil, err
	}
	var stmt *sql.Stmt
	if tx != nil {
		stmt, err = tx.Prepare(db.replacer.Replace(prepared))
	} else {
		stmt, err = db.db.Prepare(db.replacer.Replace(prepared))
	}
	if err != nil {
		return stmt, err
	}
	return stmt, sqlVersionError(err, db.driver, &prepared)
}

/*
ExecSQL automatically escapes the given values and caches the statement
Example:
	var intVal int
	var stringVal string
	result, err := db.ExecSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
*/
func (db *GCDB) ExecSQL(query string, values ...interface{}) (sql.Result, error) {
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Exec(values...)
}

/*
BeginTx creates and returns a new SQL transaction. Note that it doesn't use gochan's database variables,
e.g. DBPREFIX, DBNAME, etc so it should be used sparingly
*/
func (db *GCDB) BeginTx() (*sql.Tx, error) {
	tx, err := db.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelSerializable})
	return tx, err
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:
	id := 32
	var intVal int
	var stringVal string
	err := db.QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{&id},
		[]interface{}{&intVal, &stringVal})
*/
func (db *GCDB) QueryRowSQL(query string, values, out []interface{}) error {
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return err
	}
	defer stmt.Close()
	return stmt.QueryRow(values...).Scan(out...)
}

/*
QuerySQL gets all rows from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:
	rows, err := db.QuerySQL("SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func (db *GCDB) QuerySQL(query string, a ...interface{}) (*sql.Rows, error) {
	stmt, err := db.PrepareSQL(query, nil)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Query(a...)
}

func Open(host, dbDriver, dbName, username, password, prefix string) (db *GCDB, err error) {
	db = &GCDB{
		driver: dbDriver,
		replacer: strings.NewReplacer(
			"DBNAME", dbName,
			"DBPREFIX", prefix,
			"\n", " "),
	}

	addrMatches := tcpHostIsolator.FindAllStringSubmatch(host, -1)
	if len(addrMatches) > 0 && len(addrMatches[0]) > 2 {
		host = addrMatches[0][2]
	}

	switch dbDriver {
	case "mysql":
		db.connStr = fmt.Sprintf(mysqlConnStr, username, password, host, dbName)
		// db.nilTimestamp = "0000-00-00 00:00:00"
	case "postgres":
		db.connStr = fmt.Sprintf(postgresConnStr, username, password, host, dbName)
		// db.nilTimestamp = "0001-01-01 00:00:00"
	default:
		return nil, ErrUnsupportedDB
	}

	db.db, err = sql.Open(db.driver, db.connStr)
	return db, err
}

func sqlVersionError(err error, dbDriver string, query *string) error {
	if err == nil {
		return nil
	}
	errText := err.Error()
	switch dbDriver {
	case "mysql":
		if !strings.Contains(errText, "You have an error in your SQL syntax") {
			return err
		}
	case "postgres":
		if !strings.Contains(errText, "syntax error at or near") {
			return err
		}
	}
	if config.GetSystemCriticalConfig().DebugMode {
		return fmt.Errorf(UnsupportedSQLVersionMsg+"\nQuery: "+*query, errText)
	}
	return fmt.Errorf(UnsupportedSQLVersionMsg, errText)
}
