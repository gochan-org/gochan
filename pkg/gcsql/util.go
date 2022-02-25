package gcsql

import (
	"database/sql"
	"errors"
	"strings"
)

const (
	MySQLDatetimeFormat = "2006-01-02 15:04:05"
)

var (
	ErrUnsupportedDB = errors.New("unsupported SQL driver")
	ErrNotConnected  = errors.New("error connecting to database")
)

// PrepareSQL is used for generating a prepared SQL statement formatted according to the configured database driver
func PrepareSQL(query string, tx *sql.Tx) (*sql.Stmt, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.PrepareSQL(query, tx)
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
	result, err := gcsql.ExecSQL(db, "mysql",
		"INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
*/
func ExecSQL(query string, values ...interface{}) (sql.Result, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.ExecSQL(query, values...)
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:
	id := 32
	var intVal int
	var stringVal string
	err := QueryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{&id},
		[]interface{}{&intVal, &stringVal})
*/
func QueryRowSQL(query string, values, out []interface{}) error {
	if gcdb == nil {
		return ErrNotConnected
	}
	return gcdb.QueryRowSQL(query, values, out)
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

func BeginTx() (*sql.Tx, error) {
	if gcdb == nil {
		return nil, ErrNotConnected
	}
	return gcdb.BeginTx()
}

// ResetBoardSectionArrays is run when the board list needs to be changed
// (board/section is added, deleted, etc)
func ResetBoardSectionArrays() {
	AllBoards = nil
	AllSections = nil

	allBoardsArr, _ := GetAllBoards()
	AllBoards = append(AllBoards, allBoardsArr...)

	allSectionsArr, _ := GetAllSections()
	AllSections = append(AllSections, allSectionsArr...)
}

// interfaceSlice creates a new interface slice from an arbitrary collection of values
func interfaceSlice(args ...interface{}) []interface{} {
	return args
}

func errFilterDuplicatePrimaryKey(err error) (isPKerror bool, nonPKerror error) {
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
}
