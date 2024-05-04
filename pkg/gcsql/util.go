package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

var (
	dateTimeFormats = []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
	}
	ErrUnsupportedDB = errors.New("unsupported SQL driver")
	ErrNotConnected  = errors.New("error connecting to database")
)

// BeginTx begins a new transaction for the gochan database. It uses a background context
func BeginTx() (*sql.Tx, error) {
	return BeginContextTx(context.Background())
}

// BeginContextTx begins a new transaction for the gochan database, using the specified context
func BeginContextTx(ctx context.Context) (*sql.Tx, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.BeginTx(ctx, &sql.TxOptions{
		Isolation: 0,
		ReadOnly:  false,
	})
}

// PrepareSQL is used for generating a prepared SQL statement formatted according to the configured database driver
func PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.PrepareSQL(query, tx)
}

// SetupSQLString applies the gochan databases keywords (DBPREFIX, DBNAME, etc) based on the database
// type (MySQL, Postgres, etc) to be passed to PrepareSQL
func SetupSQLString(query string, dbConn *GCDB) (string, error) {
	var prepared string
	var err error
	if dbConn == nil {
		return "", ErrNotConnected
	}
	switch dbConn.driver {
	case "mysql":
		prepared = query
	case "sqlite3":
		fallthrough
	case "postgres":
		arr := strings.Split(query, "?")
		for i := range arr {
			if i == len(arr)-1 {
				break
			}
			arr[i] += fmt.Sprintf("$%d", i+1)
		}
		prepared = strings.Join(arr, "")
	case "sqlmock":
		if config.VerboseMode() {
			prepared = query
			break
		}
		fallthrough
	default:
		return "", ErrUnsupportedDB
	}

	return prepared, err
}

// Close closes the connection to the SQL database
func Close() error {
	if gcdb != nil {
		return gcdb.Close()
	}
	return nil
}

/*
ExecSQL automatically escapes the given values and caches the statement
Example:

	var intVal int
	var stringVal string
	result, err := gcsql.ExecSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func ExecSQL(query string, values ...interface{}) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.ExecSQL(query, values...)
}

/*
ExecTxSQL automatically escapes the given values and caches the statement
Example:

	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	var intVal int
	var stringVal string
	result, err := gcsql.ExecTxSQL(tx, "INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func ExecTxSQL(tx *sql.Tx, query string, values ...interface{}) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(values...)
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:

	id := 32
	var intVal int
	var stringVal string
	err := QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{id},
		[]interface{}{&intVal, &stringVal})
*/
func QueryRowSQL(query string, values, out []interface{}) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowSQL(query, values, out)
}

/*
QueryRowTxSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:

	id := 32
	var intVal int
	var stringVal string
	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	err = QueryRowTxSQL(tx, "SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{id},
		[]interface{}{&intVal, &stringVal})
*/
func QueryRowTxSQL(tx *sql.Tx, query string, values, out []interface{}) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowTxSQL(tx, query, values, out)
}

/*
QuerySQL gets all rows from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:

	rows, err := sqlutil.QuerySQL("SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func QuerySQL(query string, a ...interface{}) (*sql.Rows, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.QuerySQL(query, a...)
}

/*
QueryTxSQL gets all rows from the db using the transaction tx with the values in values[] and fills the
respective pointers in out[]. Automatically escapes the given values and caches the query
Example:

	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	rows, err := sqlutil.QueryTxSQL(tx, "SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func QueryTxSQL(tx *sql.Tx, query string, a ...interface{}) (*sql.Rows, error) {
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	return stmt.Query(a...)
}

func ParseSQLTimeString(str string) (time.Time, error) {
	var t time.Time
	var err error
	for _, layout := range dateTimeFormats {
		if t, err = time.Parse(layout, str); err == nil {
			return t, nil
		}
	}
	return t, fmt.Errorf("unrecognized timestamp string format %q", str)
}

// getLatestID returns the latest inserted id column value from the given table
func getLatestID(tableName string, tx *sql.Tx) (id int, err error) {
	query := `SELECT MAX(id) FROM ` + tableName
	if tx != nil {
		var stmt *sql.Stmt
		stmt, err = PrepareSQL(query, tx)
		if err != nil {
			return 0, err
		}
		err = stmt.QueryRow().Scan(&id)
	} else {
		err = QueryRowSQL(query, nil, interfaceSlice(&id))
	}
	return
}

func doesTableExist(tableName string) (bool, error) {
	var existQuery string

	switch config.GetSystemCriticalConfig().DBtype {
	case "mysql":
		fallthrough
	case "postgres":
		existQuery = `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME = ?`
	case "sqlite3":
		existQuery = `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`
	default:
		return false, ErrUnsupportedDB
	}

	var count int
	err := QueryRowSQL(existQuery, []interface{}{config.GetSystemCriticalConfig().DBprefix + tableName}, []interface{}{&count})
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// getDatabaseVersion gets the version of the database, or an error if none or multiple exist
func getDatabaseVersion(componentKey string) (int, error) {
	const sql = `SELECT version FROM DBPREFIXdatabase_version WHERE component = ?`
	var version int
	err := QueryRowSQL(sql, []interface{}{componentKey}, []interface{}{&version})
	if err != nil {
		return 0, err
	}
	return version, err
}

// doesGochanPrefixTableExist returns true if any table with a gochan prefix was found.
// Returns false if the prefix is an empty string
func doesGochanPrefixTableExist() (bool, error) {
	systemCritical := config.GetSystemCriticalConfig()
	if systemCritical.DBprefix == "" {
		return false, nil
	}
	var prefixTableExist string
	switch systemCritical.DBtype {
	case "mysql":
		fallthrough
	case "postgresql":
		prefixTableExist = `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME LIKE 'DBPREFIX%'`
	case "sqlite3":
		prefixTableExist = `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name LIKE 'DBPREFIX%'`
	}

	var count int
	err := QueryRowSQL(prefixTableExist, []interface{}{}, []interface{}{&count})
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return count > 0, nil
}

// interfaceSlice creates a new interface slice from an arbitrary collection of values
func interfaceSlice(args ...interface{}) []interface{} {
	return args
}

/* func errFilterDuplicatePrimaryKey(err error) (isPKerror bool, nonPKerror error) {
	if err == nil {
		return false, nil
	}

	switch gcdb.driver {
	case "mysql":
		if !strings.Contains(err.Error(), "Duplicate entry") {
			return false, err
		}
	case "postgres":
		if !strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			return false, err
		}
	}
	return true, nil
} */

// createArrayPlaceholder creates a string of ?s based on the size of arr
func createArrayPlaceholder(arr []interface{}) string {
	params := make([]string, len(arr))
	for p := range params {
		params[p] = "?"
	}
	return "(" + strings.Join(params, ",") + ")"
}
