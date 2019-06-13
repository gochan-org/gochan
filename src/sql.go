package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const (
	mysqlDatetimeFormat      = "2006-01-02 15:04:05"
	unsupportedSQLVersionMsg = `Received syntax error while preparing a SQL string.
This means that either there is a bug in gochan's code (hopefully not) or that you are using an unsupported My/Postgre/SQLite version.
Before reporting an error, make sure that you are using the up to date version of your selected SQL server.
Error text: %s
`
)

var (
	db           *sql.DB
	nilTimestamp string
)

func connectToSQLServer() {
	var err error
	var connStr string
	println(0, "Initializing server...")

	switch config.DBtype {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@%s/%s?parseTime=true&collation=utf8mb4_unicode_ci",
			config.DBusername, config.DBpassword, config.DBhost, config.DBname)
		nilTimestamp = "0000-00-00 00:00:00"
	case "postgres":
		connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=verify-ca",
			config.DBusername, config.DBpassword, config.DBhost, config.DBname)
		nilTimestamp = "0001-01-01 00:00:00"
	case "sqlite3":
		connStr = config.DBhost
		nilTimestamp = "0001-01-01 00:00:00+00:00"
	default:
		handleError(0, "Invalid DBtype '%s' in gochan.json, valid values are 'mysql' and 'postgres'", config.DBtype)
		os.Exit(2)
	}

	nullTime, _ = time.Parse("2006-01-02 15:04:05", nilTimestamp)
	if db, err = sql.Open(config.DBtype, connStr); err != nil {
		handleError(0, "Failed to connect to the database: %s\n", customError(err))
		os.Exit(2)
	}

	if err = initDB("initdb_" + config.DBtype + ".sql"); err != nil {
		println(0, "Failed initializing DB:", sqlVersionErr(err))
		os.Exit(2)
	}

	var sqlVersionStr string
	err = queryRowSQL("SELECT value FROM "+config.DBprefix+"info WHERE name = 'version'",
		[]interface{}{}, []interface{}{&sqlVersionStr})

	if err == sql.ErrNoRows {
		println(0, "\nThis looks like a new installation")

		if _, err = db.Exec("INSERT INTO " + config.DBprefix + "staff " +
			"(username, password_checksum, salt, rank) " +
			"VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3)",
		); err != nil {
			handleError(0, "Failed creating admin user with error: %s\n", customError(err))
			os.Exit(2)
		}

		_, err = execSQL("INSERT INTO "+config.DBprefix+"info (name,value) VALUES('version',?)", versionStr)
		return
	} else if err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}
	if err != nil && !strings.Contains(err.Error(), "Duplicate entry") {
		handleError(0, "failed with error: %s\n", customError(err))
		os.Exit(2)
	}
	if version.CompareString(sqlVersionStr) > 0 {
		printf(0, "Updating version in database from %s to %s\n", sqlVersionStr, version.String())
		execSQL("UPDATE "+config.DBprefix+"info SET value = ? WHERE name = 'version'", versionStr)
	}

}

func initDB(initFile string) error {
	var err error
	if _, err = os.Stat(initFile); err != nil {
		return fmt.Errorf("SQL database initialization file (%s) missing. Please reinstall gochan", initFile)
	}

	sqlBytes, err := ioutil.ReadFile(initFile)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlStr = strings.NewReplacer(
		"DBNAME", config.DBname,
		"DBPREFIX", config.DBprefix,
		"\n", " ").Replace(sqlStr)
	sqlArr := strings.Split(sqlStr, ";")

	for _, statement := range sqlArr {
		if statement != "" && statement != " " {
			if _, err := db.Exec(statement + ";"); err != nil {
				return err
			}
		}
	}
	return nil
}

// checks to see if the given error is a syntax error (used for built-in strings)
func sqlVersionErr(err error) error {
	if err == nil {
		return nil
	}
	errText := err.Error()
	switch config.DBtype {
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
	switch config.DBtype {
	case "mysql":
		fallthrough
	case "sqlite3":
		preparedStr = query
		break
	case "postgres":
		arr := strings.Split(query, "?")
		for i := range arr {
			if i == len(arr)-1 {
				break
			}
			arr[i] += fmt.Sprintf("$%d", i+1)
		}
		preparedStr = strings.Join(arr, "")
		break
	}
	stmt, err := db.Prepare(preparedStr)
	return stmt, sqlVersionErr(err)
}

/*
 * Automatically escapes the given values and caches the statement
 * Example:
 * var intVal int
 * var stringVal string
 * result, err := execSQL("INSERT INTO tablename (intval,stringval) VALUES(?,?)", intVal, stringVal)
 */
func execSQL(query string, values ...interface{}) (sql.Result, error) {
	stmt, err := prepareSQL(query)
	defer closeHandle(stmt)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(values...)
}

/*
 * Gets a row from the db with the values in values[] and fills the respective pointers in out[]
 * Automatically escapes the given values and caches the query
 * Example:
 * id := 32
 * var intVal int
 * var stringVal string
 * err := queryRowSQL("SELECT intval,stringval FROM table WHERE id = ?",
 * 	[]interface{}{&id},
 * 	[]interface{}{&intVal, &stringVal}
 * )
 */
func queryRowSQL(query string, values []interface{}, out []interface{}) error {
	stmt, err := prepareSQL(query)
	defer closeHandle(stmt)
	if err != nil {
		return err
	}
	return stmt.QueryRow(values...).Scan(out...)
}

/*
 * Gets all rows from the db with the values in values[] and fills the respective pointers in out[]
 * Automatically escapes the given values and caches the query
 * Example:
 * rows, err := querySQL("SELECT * FROM table")
 * if err == nil {
 * 	for rows.Next() {
 * 		var intVal int
 * 		var stringVal string
 * 		rows.Scan(&intVal, &stringVal)
 * 		// do something with intVal and stringVal
 * 	}
 * }
 */
func querySQL(query string, a ...interface{}) (*sql.Rows, error) {
	stmt, err := prepareSQL(query)
	defer closeHandle(stmt)
	if err != nil {
		return nil, err
	}
	return stmt.Query(a...)
}

func getSQLDateTime() string {
	return time.Now().Format(mysqlDatetimeFormat)
}

func getSpecificSQLDateTime(t time.Time) string {
	return t.Format(mysqlDatetimeFormat)
}
