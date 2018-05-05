package main

import (
	"database/sql"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	nilTimestamp        = "0000-00-00 00:00:00"
	mysqlDatetimeFormat = "2006-01-02 15:04:05"
)

var (
	db          *sql.DB
	dbConnected = false
)

func connectToSQLServer() {
	var err error

	db, err = sql.Open("mysql", config.DBusername+":"+config.DBpassword+"@"+config.DBhost+"/"+config.DBname+"?parseTime=true&collation=utf8mb4_unicode_ci")
	if err != nil {
		printf(0, "Failed to connect to the database: ")
		handleError(0, customError(err))
		os.Exit(2)
	}

	// get the number of tables in the database. If the number > 1, we can assume that initial setup has already been run
	var numRows int
	err = db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ?", config.DBname).Scan(&numRows)
	if err == sql.ErrNoRows {
		numRows = 0
	} else if err != nil {
		printf(0, "Failed retrieving list of tables in database: ")
		handleError(0, customError(err))
		os.Exit(2)
	}
	// Detect that there are at least the number of tables that we are setting up.
	// If there are fewer than that, then we either half-way set up, or there's other tables in our database.
	if numRows >= 16 {
		// the initial setup has already been run
		needsInitialSetup = false
		dbConnected = true
		println(0, "complete.")
		return
	}

	// check if initialsetupdb.sql still exists
	if _, err = os.Stat("initialsetupdb.sql"); err != nil {
		println(0, "Initial setup file (initialsetupdb.sql) missing. Please reinstall gochan")
		errorLog.Fatal("Initial setup file (initialsetupdb.sql) missing. Please reinstall gochan")
	}

	// read the initial setup sql file into a string
	initialSQLBytes, err := ioutil.ReadFile("initialsetupdb.sql")
	if err != nil {
		printf(0, "failed: ")
		handleError(0, customError(err))
		os.Exit(2)
	}
	initialSQLStr := string(initialSQLBytes)

	printf(0, "Starting initial setup...")
	initialSQLStr = strings.Replace(initialSQLStr, "DBNAME", config.DBname, -1)
	initialSQLStr = strings.Replace(initialSQLStr, "DBPREFIX", config.DBprefix, -1)
	initialSQLStr += "\nINSERT INTO `" + config.DBname + "`.`" + config.DBprefix + "staff` (`username`, `password_checksum`, `salt`, `rank`) VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3);"
	initialSQLArr := strings.Split(initialSQLStr, ";")

	for _, statement := range initialSQLArr {
		if statement != "" {
			if _, err := db.Exec(statement); err != nil {
				println(0, "failed, see log for details.")
				errorLog.Fatal("Error executing initialsetupdb.sql: " + customError(err))
				return
			}
		}
	}
	println(0, "complete.")
	needsInitialSetup = false
	dbConnected = true

}

func getSQLDateTime() string {
	return time.Now().Format(mysqlDatetimeFormat)
}

func getSpecificSQLDateTime(t time.Time) string {
	return t.Format(mysqlDatetimeFormat)
}
