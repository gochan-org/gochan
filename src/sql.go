package main

import (
	"database/sql"
	"io/ioutil"
	"os"
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

	db, err = sql.Open("mysql", config.DBusername+":"+config.DBpassword+"@"+config.DBhost+"/"+config.DBname+"?parseTime=true&collation=utf8mb4_unicode_ci")
	if err != nil {
		handleError(0, "Failed to connect to the database: %s\n", customError(err))
		os.Exit(2)
	}

	// check if initialsetupdb.sql still exists
	if _, err = os.Stat("initialsetupdb.sql"); err != nil {
		handleError(0, "Initial setup file (initialsetupdb.sql) missing. Please reinstall gochan")
		os.Exit(2)
	}

	// read the initial setup sql file into a string
	initialSQLBytes, err := ioutil.ReadFile("initialsetupdb.sql")
	if err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}
	initialSQLStr := string(initialSQLBytes)

	printf(0, "Starting initial setup...")
	initialSQLStr += "\nINSERT INTO `DBNAME`.`DBPREFIXstaff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3);"
	initialSQLStr = strings.NewReplacer("DBNAME", config.DBname, "DBPREFIX", config.DBprefix).Replace(initialSQLStr)

	initialSQLArr := strings.Split(initialSQLStr, ";")

	for _, statement := range initialSQLArr {
		if statement != "" {
			if _, err := db.Exec(statement); err != nil {
				handleError(0, "failed with error: %s\n", customError(err))
				os.Exit(2)
			}
		}
	}
	println(0, "complete.")
}

func execSQL(query string, values ...interface{}) (sql.Result, error) {
	stmt, err := db.Prepare(query)
	defer closeStatement(stmt)
	if err != nil {
		return nil, err
	}
	return stmt.Exec(values...)
}

func queryRowSQL(query string, values []interface{}, out []interface{}) error {
	stmt, err := db.Prepare(query)
	defer closeStatement(stmt)
	if err != nil {
		return err
	}
	return stmt.QueryRow(values...).Scan(out...)
}

func querySQL(query string, a ...interface{}) (*sql.Rows, error) {
	stmt, err := db.Prepare(query)
	defer closeStatement(stmt)
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
