package gcsql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	MySQLDatetimeFormat      = "2006-01-02 15:04:05"
	unsupportedSQLVersionMsg = `Received syntax error while preparing a SQL string.
This means that either there is a bug in gochan's code (hopefully not) or that you are using an unsupported My/Postgre/SQLite version.
Before reporting an error, make sure that you are using the up to date version of your selected SQL server.
Error text: %s`
)

func sqlVersionErr(err error, query *string) *gcutil.GcError {
	if err == nil {
		return nil
	}
	errText := err.Error()
	switch dbDriver {
	case "mysql":
		if !strings.Contains(errText, "You have an error in your SQL syntax") {
			return gcutil.FromError(err, false)
		}
	case "postgres":
		if !strings.Contains(errText, "syntax error at or near") {
			return gcutil.FromError(err, false)
		}
	case "sqlite3":
		if !strings.Contains(errText, "Error: near ") {
			return gcutil.FromError(err, false)
		}
	}
	if config.Config.DebugMode {
		return gcutil.NewError(fmt.Sprintf(unsupportedSQLVersionMsg+"\nQuery: "+*query, errText), false)
	}
	return gcutil.NewError(fmt.Sprintf(unsupportedSQLVersionMsg, errText), false)
}

// PrepareSQL is used for generating a prepared SQL statement formatted according to config.DBtype
func PrepareSQL(query string) (*sql.Stmt, *gcutil.GcError) {
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
	if err != nil {
		gclog.Print(gclog.LErrorLog,
			"Error preparing sql query:", "\n", query, "\n", err.Error())
	}
	return stmt, sqlVersionErr(err, &preparedStr)
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
func ExecSQL(query string, values ...interface{}) (sql.Result, *gcutil.GcError) {
	stmt, gcerr := PrepareSQL(query)
	if gcerr != nil {
		return nil, gcerr
	}
	defer stmt.Close()
	res, err := stmt.Exec(values...)
	return res, gcutil.FromError(err, false)
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
func QueryRowSQL(query string, values []interface{}, out []interface{}) *gcutil.GcError {
	stmt, err := PrepareSQL(query)
	if err != nil {
		return err
	}
	defer stmt.Close()
	return gcutil.FromError(stmt.QueryRow(values...).Scan(out...), false)
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
func QuerySQL(query string, a ...interface{}) (*sql.Rows, *gcutil.GcError) {
	stmt, gcerr := PrepareSQL(query)
	if gcerr != nil {
		return nil, gcerr
	}
	defer stmt.Close()
	rows, err := stmt.Query(a...)
	return rows, gcutil.FromError(err, false)
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

func errFilterDuplicatePrimaryKey(err *gcutil.GcError) (isPKerror bool, nonPKerror *gcutil.GcError) {
	if err == nil {
		return false, nil
	}

	switch dbDriver {
	case "mysql":
		if !strings.Contains(err.Message, "Duplicate entry") {
			return false, err
		}
	case "postgres":
		if !strings.Contains(err.Message, "duplicate key value violates unique constraint") {
			return false, err
		}
	case "sqlite3":
		return false, gcutil.ErrNotImplemented
		// if !strings.Contains(err.Error(), "Error: near ") {//TODO fill in correct error string
		// 	return false, err
		// }
	}
	return true, nil
}
