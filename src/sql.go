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
	sqlReplacer  *strings.Replacer // used during SQL string preparation
)

func connectToSQLServer() {
	var err error
	var connStr string
	sqlReplacer = strings.NewReplacer(
		"DBNAME", config.DBname,
		"DBPREFIX", config.DBprefix,
		"\n", " ")
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
		println(0, "sqlite3 support is still flaky, consider using mysql or postgres")
		connStr = fmt.Sprintf("file:%s?mode=rwc&_auth&_auth_user=%s&_auth_pass=%s&cache=shared",
			config.DBhost, config.DBusername, config.DBpassword)
		nilTimestamp = "0001-01-01 00:00:00+00:00"
	default:
		handleError(0, "Invalid DBtype '%s' in gochan.json, valid values are 'mysql', 'postgres', and 'sqlite3'", config.DBtype)
		os.Exit(2)
	}

	if db, err = sql.Open(config.DBtype, connStr); err != nil {
		handleError(0, "Failed to connect to the database: %s\n", customError(err))
		os.Exit(2)
	}

	if err = initDB("initdb_" + config.DBtype + ".sql"); err != nil {
		println(0, "Failed initializing DB:", err)
		os.Exit(2)
	}

	var truncateStr string
	switch config.DBtype {
	case "mysql":
		fallthrough
	case "postgres":
		truncateStr = "TRUNCATE TABLE DBPREFIXsessions"
	case "sqlite3":
		truncateStr = "DELETE FROM DBPREFIXsessions"
	}

	if _, err = execSQL(truncateStr); err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}

	var sqlVersionStr string
	isNewInstall := false
	if err = queryRowSQL(
		"SELECT value FROM DBPREFIXinfo WHERE name = 'version'",
		[]interface{}{}, []interface{}{&sqlVersionStr},
	); err == sql.ErrNoRows {
		isNewInstall = true
	} else if err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}

	var numBoards, numStaff int
	rows, err := querySQL("SELECT COUNT(*) FROM DBPREFIXboards UNION ALL SELECT COUNT(*) FROM DBPREFIXstaff")
	if err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}
	rows.Next()
	rows.Scan(&numBoards)
	rows.Next()
	rows.Scan(&numStaff)

	if numBoards == 0 && numStaff == 0 {
		println(0, "This looks like a new installation. Creating /test/ and a new staff member.\nUsername: admin\nPassword: password")

		if _, err = execSQL(
			"INSERT INTO DBPREFIXstaff (username,password_checksum,rank) VALUES(?,?,?)",
			"admin", bcryptSum("password"), 3,
		); err != nil {
			handleError(0, "Failed creating admin user with error: %s\n", customError(err))
			os.Exit(2)
		}

		firstBoard := Board{
			Dir:         "test",
			Title:       "Testing board",
			Subtitle:    "Board for testing",
			Description: "Board for testing"}
		firstBoard.SetDefaults()
		firstBoard.Build(true, true)
		if !isNewInstall {
			return
		}

		if _, err = execSQL(
			"INSERT INTO DBPREFIXinfo (name,value) VALUES('version',?)",
			versionStr); err != nil {
			handleError(0, "failed: %s\n", err.Error())
		}
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
		execSQL("UPDATE DBPREFIXinfo SET value = ? WHERE name = 'version'", versionStr)
	}

}

func initDB(initFile string) error {
	var err error
	filePath := findResource(initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return fmt.Errorf("SQL database initialization file (%s) missing. Please reinstall gochan", initFile)
	}

	sqlBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(sqlReplacer.Replace(sqlStr), ";")

	for _, statement := range sqlArr {
		if statement != "" && statement != " " {
			if _, err = db.Exec(statement); err != nil {
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
