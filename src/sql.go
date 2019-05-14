package main

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	// _ "github.com/lib/pq"
)

const (
	nilTimestamp        = "0000-00-00 00:00:00"
	mysqlDatetimeFormat = "2006-01-02 15:04:05"
)

var (
	db *sql.DB
)

func connectToSQLServer() {
	var err error

	println(0, "Initializing server...")
	db, err = sql.Open("mysql", config.DBusername+":"+config.DBpassword+"@"+config.DBhost+"/"+config.DBname+"?parseTime=true&collation=utf8mb4_unicode_ci")
	if err != nil {
		handleError(0, "Failed to connect to the database: %s\n", customError(err))
		os.Exit(2)
	}

	if err = initDB(); err != nil {
		handleError(0, "Failed initializing DB: %s\n", customError(err))
		os.Exit(2)
	}

	var sqlVersionStr string
	err = queryRowSQL("SELECT `value` FROM `"+config.DBprefix+"info` WHERE `name` = 'version'",
		[]interface{}{}, []interface{}{&sqlVersionStr})

	if err == sql.ErrNoRows {
		println(0, "\nThis looks like a new installation")

		if _, err = db.Exec("INSERT INTO `" + config.DBname + "`.`" + config.DBprefix + "staff` " +
			"(`username`, `password_checksum`, `salt`, `rank`) " +
			"VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3)",
		); err != nil {
			handleError(0, "Failed creating admin user with error: %s\n", customError(err))
			os.Exit(2)
		}

		_, err = execSQL("INSERT INTO `"+config.DBprefix+"info` (`name`,`value`) VALUES('version',?)", versionStr)
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
		execSQL("UPDATE `"+config.DBprefix+"info` SET `value` = ? WHERE `name` = 'version'", versionStr)
	}

}

func initDB() error {
	var err error
	if _, err = os.Stat("initdb.sql"); err != nil {
		return errors.New("SQL database initialization file (initdb.sql) missing. Please reinstall gochan")
	}

	sqlBytes, err := ioutil.ReadFile("initdb.sql")
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

/*
 * Automatically escapes the given values and caches the statement
 * Example:
 * var intVal int
 * var stringVal string
 * result, err := execSQL("INSERT INTO `tablename` (`intval`,`stringval`) VALUES(?,?)", intVal, stringVal)
 */
func execSQL(query string, values ...interface{}) (sql.Result, error) {
	stmt, err := db.Prepare(query)
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
 * err := queryRowSQL("SELECT `intval`,`stringval` FROM `table` WHERE `id` = ?",
 * 	[]interface{}{&id},
 * 	[]interface{}{&intVal, &stringVal}
 * )
 */
func queryRowSQL(query string, values []interface{}, out []interface{}) error {
	stmt, err := db.Prepare(query)
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
 * rows, err := querySQL("SELECT * FROM `table`")
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
	stmt, err := db.Prepare(query)
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

func checkTableExists(tableName string) bool {
	rows, err := querySQL("SELECT * FROM information_schema.tables WHERE `TABLE_SCHEMA` = ? AND `TABLE_NAME` = ? LIMIT 1",
		config.DBname, tableName)
	return err == nil && rows.Next() == true
}
