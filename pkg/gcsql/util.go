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

// ActiveFilter is used for optionally limiting the results of tables with an is_active column to
type ActiveFilter int

// whereClause returns part of the where clause of a SQL string. If and is true, it starts with AND, otherwise it starts with WHERE
func (af ActiveFilter) whereClause(and bool) string {
	out := " WHERE "
	if and {
		out = " AND "
	}
	if af == OnlyActiveFilters {
		return out + "is_active = TRUE"
	} else if af == OnlyInactiveFilters {
		return out + "is_active = FALSE"
	}
	return ""
}

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

func PrepareContextSQL(ctx context.Context, query string, tx *sql.Tx) (*sql.Stmt, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.PrepareContextSQL(ctx, query, tx)
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
ExecSQL executes the given SQL statement with the given parameters
Example:

	var intVal int
	var stringVal string
	result, err := gcsql.ExecSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func ExecSQL(query string, values ...any) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.ExecSQL(query, values...)
}

/*
ExecContextSQL executes the given SQL statement with the given context, optionally with the given transaction (if non-nil)

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	var intVal int
	var stringVal string
	result, err := gcsql.ExecContextSQL(ctx, nil, "INSERT INTO tablename (intval,stringval) VALUES(?,?)",
		intVal, stringVal)
*/
func ExecContextSQL(ctx context.Context, tx *sql.Tx, sqlStr string, values ...any) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.ExecContextSQL(ctx, tx, sqlStr, values...)
}

func ExecTimeoutSQL(tx *sql.Tx, sqlStr string, values ...any) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	return ExecContextSQL(ctx, tx, sqlStr, values...)
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
func ExecTxSQL(tx *sql.Tx, sqlStr string, values ...any) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	stmt, err := PrepareSQL(sqlStr, tx)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	res, err := stmt.Exec(values...)
	if err != nil {
		return res, err
	}
	return res, stmt.Close()
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query

Example:

	id := 32
	var intVal int
	var stringVal string
	err := gcsql.QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]any{id},
		[]any{&intVal, &stringVal})
*/
func QueryRowSQL(query string, values, out []any) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowSQL(query, values, out)
}

/*
QueryRowContextSQL gets a row from the database with the values in values[] and fills the respective pointers in out[]
using the given context as a deadline, and the given transaction (if non-nil)

Example:

	id := 32
	var name string
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	err := gcsql..QueryRowContextSQL(ctx, nil, "SELECT name FROM DBPREFIXposts WHERE id = ? LIMIT 1",
		[]any{id}, []any{&name})
*/
func QueryRowContextSQL(ctx context.Context, tx *sql.Tx, query string, values, out []any) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowContextSQL(ctx, tx, query, values, out)
}

// QueryRowTimeoutSQL is a helper function for querying a single row with the configured default timeout.
// It creates a context with the default timeout to only be used for this query and then disposed.
// It should only be used by a function that does a single SQL query, otherwise use QueryRowContextSQL
func QueryRowTimeoutSQL(tx *sql.Tx, query string, values, out []any) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	return QueryRowContextSQL(ctx, tx, query, values, out)
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
	err = gcsql.QueryRowTxSQL(tx, "SELECT intval,stringval FROM table WHERE id = ?",
		[]any{id}, []any{&intVal, &stringVal})
*/
func QueryRowTxSQL(tx *sql.Tx, query string, values, out []any) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowTxSQL(tx, query, values, out)
}

/*
QuerySQL gets all rows from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query

Example:

	rows, err := gcsql.QuerySQL("SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func QuerySQL(query string, a ...any) (*sql.Rows, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.QuerySQL(query, a...)
}

/*
QueryContextSQL queries the database with a prepared statement and the given parameters, using the given context
for a deadline

Example:

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds) * time.Second)
	defer cancel()
	rows, err := gcsql.QueryContextSQL(ctx, nil, "SELECT name from posts where NOT is_deleted")
*/
func QueryContextSQL(ctx context.Context, tx *sql.Tx, query string, a ...any) (*sql.Rows, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.QueryContextSQL(ctx, tx, query, a...)
}

// QueryTimeoutSQL creates a new context with the configured default timeout and passes it and
// the given transaction, query, and parameters to QueryContextSQL. If it returns an error,
// the context is cancelled, and the error is returned. Otherwise, it returns the rows,
// cancel function (for the calling function to call later), and nil error. It should only be used
// if the calling function is only doing one SQL query, otherwise use QueryContextSQL.
func QueryTimeoutSQL(tx *sql.Tx, query string, a ...any) (*sql.Rows, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	rows, err := QueryContextSQL(ctx, tx, query, a...)
	if err != nil {
		cancel()
		return nil, cancel, err
	}
	return rows, cancel, nil
}

/*
QueryTxSQL gets all rows from the db using the transaction tx with the values in values[] and fills the
respective pointers in out[]. Automatically escapes the given values and caches the query
Example:

	tx, err := BeginTx()
	// do error handling stuff
	defer tx.Rollback()
	rows, err := gcsql.QueryTxSQL(tx, "SELECT * FROM table")
	if err == nil {
		for rows.Next() {
			var intVal int
			var stringVal string
			rows.Scan(&intVal, &stringVal)
			// do something with intVal and stringVal
		}
	}
*/
func QueryTxSQL(tx *sql.Tx, query string, a ...any) (*sql.Rows, error) {
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(a...)
	if err != nil {
		return nil, err
	}
	return rows, stmt.Close()
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
		defer stmt.Close()
		if err = stmt.QueryRow().Scan(&id); err != nil {
			return
		}
		err = stmt.Close()
	} else {
		err = QueryRowSQL(query, nil, []any{&id})
	}
	return
}

func doesTableExist(tableName string) (bool, error) {
	var existQuery string

	switch config.GetSQLConfig().DBtype {
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
	err := QueryRowSQL(existQuery, []any{config.GetSystemCriticalConfig().DBprefix + tableName}, []any{&count})
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

// getDatabaseVersion gets the version of the database, or an error if none or multiple exist
func getDatabaseVersion(componentKey string) (int, error) {
	const sql = `SELECT version FROM DBPREFIXdatabase_version WHERE component = ?`
	var version int
	err := QueryRowSQL(sql, []any{componentKey}, []any{&version})
	if err != nil {
		return 0, err
	}
	return version, err
}

// doesGochanPrefixTableExist returns true if any table with a gochan prefix was found.
// Returns false if the prefix is an empty string
func doesGochanPrefixTableExist() (bool, error) {
	sqlConfig := config.GetSQLConfig()
	if sqlConfig.DBprefix == "" {
		return false, nil
	}
	var prefixTableExist string
	switch sqlConfig.DBtype {
	case "mysql":
		fallthrough
	case "postgresql":
		prefixTableExist = `SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_NAME LIKE 'DBPREFIX%'`
	case "sqlite3":
		prefixTableExist = `SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name LIKE 'DBPREFIX%'`
	}

	var count int
	err := QueryRowSQL(prefixTableExist, []any{}, []any{&count})
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}
	return count > 0, nil
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
func createArrayPlaceholder(arr []any) string {
	params := make([]string, len(arr))
	for p := range params {
		params[p] = "?"
	}
	return "(" + strings.Join(params, ",") + ")"
}
