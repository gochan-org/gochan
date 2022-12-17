package gcsql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// GochanVersionKeyConstant is the key value used in the version table of the database to store and receive the (database) version of base gochan
	gochanVersionKeyConstant = "gochan"
	UnsupportedSQLVersionMsg = `Received syntax error while preparing a SQL string.
	This means that either there is a bug in gochan's code (hopefully not) or that you are using an unsupported MySQL/PostgreSQL version.
	Before reporting an error, make sure that you are using the up to date version of your selected SQL server.
	Error text: %s`
	mysqlConnStr    = "%s:%s@tcp(%s)/%s?parseTime=true&collation=utf8mb4_unicode_ci"
	postgresConnStr = "postgres://%s:%s@%s/%s?sslmode=disable"
	sqlite3ConnStr  = "file:%s?_auth&_auth_user=%s&_auth_pass=%s&_auth_crypt=sha1"
)

var gcdb *GCDB

type GCDB struct {
	db       *sql.DB
	connStr  string
	driver   string
	replacer *strings.Replacer
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

func (db *GCDB) Close() error {
	if db.db != nil {
		return db.db.Close()
	}
	return nil
}

func (db *GCDB) PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	var prepared string
	var err error
	if prepared, err = SetupSQLString(query, db); err != nil {
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
Begin creates and returns a new SQL transaction using the GCDB. Note that it doesn't use gochan's
database variables, e.g. DBPREFIX, DBNAME, etc so it should be used sparingly or with
gcsql.SetupSQLString
*/
func (db *GCDB) Begin() (*sql.Tx, error) {
	return db.db.Begin()
}

/*
BeginTx creates and returns a new SQL transaction using the GCDB with the specified context
and transaction options. Note that it doesn't use gochan's database variables, e.g. DBPREFIX,
DBNAME, etc so it should be used sparingly or with gcsql.SetupSQLString
*/
func (db *GCDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return db.db.BeginTx(ctx, opts)
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:

	id := 32
	var intVal int
	var stringVal string
	err := db.QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{id},
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

	if dbDriver != "sqlite3" {
		addrMatches := tcpHostIsolator.FindAllStringSubmatch(host, -1)
		if len(addrMatches) > 0 && len(addrMatches[0]) > 2 {
			host = addrMatches[0][2]
		}
	}

	switch dbDriver {
	case "mysql":
		db.connStr = fmt.Sprintf(mysqlConnStr, username, password, host, dbName)
		// db.nilTimestamp = "0000-00-00 00:00:00"
	case "sqlite3":
		db.connStr = fmt.Sprintf(sqlite3ConnStr, host, username, password)
	case "postgres":
		db.connStr = fmt.Sprintf(postgresConnStr, username, password, host, dbName)
		// db.nilTimestamp = "0001-01-01 00:00:00"
	default:
		return nil, ErrUnsupportedDB
	}
	db.db, err = sql.Open(db.driver, db.connStr)
	return db, err
}

// OptimizeDatabase peforms a database optimisation
func OptimizeDatabase() error {
	tableRows, tablesErr := QuerySQL("SHOW TABLES")
	if tablesErr != nil {
		return tablesErr
	}
	defer tableRows.Close()
	for tableRows.Next() {
		var table string
		tableRows.Scan(&table)
		if _, err := ExecSQL("OPTIMIZE TABLE " + table); err != nil {
			return err
		}
	}
	return nil
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
