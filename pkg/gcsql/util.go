package gcsql

import (
	"database/sql"
	"fmt"
	"strings"
)

const (
	MySQLDatetimeFormat      = "2006-01-02 15:04:05"
	unsupportedSQLVersionMsg = `Received syntax error while preparing a SQL string.
This means that either there is a bug in gochan's code (hopefully not) or that you are using an unsupported My/Postgre/SQLite version.
Before reporting an error, make sure that you are using the up to date version of your selected SQL server.
Error text: %s`
)

func sqlVersionErr(err error) error {
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
	case "sqlite3":
		if !strings.Contains(errText, "Error: near ") {
			return err
		}
	}
	return fmt.Errorf(unsupportedSQLVersionMsg, errText)
}

// used for generating a prepared SQL statement formatted according to config.DBtype
func prepareSQL(query string) (*sql.Stmt, error) {
	var preparedStr string
	switch dbDriver {
	case "mysql":
		fallthrough
	case "sqlite3":
		preparedStr = query
	case "postgres":
		arr := strings.Split(query, "?")
		for i := range arr {
			if i == len(arr)-1 {
				break
			}
			arr[i] += fmt.Sprintf("$%d", i+1)
		}
		preparedStr = strings.Join(arr, "")
	}
	stmt, err := db.Prepare(sqlReplacer.Replace(preparedStr))
	return stmt, sqlVersionErr(err)
}

// Close closes the connection to the SQL database
func Close() {
	if db != nil {
		db.Close()
	}
}

/*
ExecSQL automatically escapes the given values and caches the statement
Example:
	var intVal int
	var stringVal string
	result, err := gcsql.ExecSQL(
		"INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
*/
func ExecSQL(query string, values ...interface{}) (sql.Result, error) {
	stmt, err := prepareSQL(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Exec(values...)
}

/*
QueryRowSQL gets a row from the db with the values in values[] and fills the respective pointers in out[]
Automatically escapes the given values and caches the query
Example:
	id := 32
	var intVal int
	var stringVal string
	err := queryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
		[]interface{}{&id},
		[]interface{}{&intVal, &stringVal})
*/
func QueryRowSQL(query string, values []interface{}, out []interface{}) error {
	stmt, err := prepareSQL(query)
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
func QuerySQL(query string, a ...interface{}) (*sql.Rows, error) {
	stmt, err := prepareSQL(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	return stmt.Query(a...)
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

// InterfaceSlice creates a new interface slice from an arbitrary collection of values
func InterfaceSlice(args ...interface{}) []interface{} {
	return args
}
