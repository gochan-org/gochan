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
	var sqlVersion string
	var newInstall bool

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

	printf(0, "Starting initial setup...")

	initialSQLStr := string(initialSQLBytes)
	initialSQLStr = strings.NewReplacer("DBNAME", config.DBname, "DBPREFIX", config.DBprefix).Replace(initialSQLStr)
	initialSQLArr := strings.Split(initialSQLStr, ";")
	for _, statement := range initialSQLArr {
		if statement != "" && statement != "\n" && strings.Index(statement, "--") != 0 {
			if _, err := db.Exec(statement); err != nil {
				handleError(0, "failed: %s\n", customError(err))
				os.Exit(2)
			}
		}
	}

	sqlVersion = ""
	err = queryRowSQL("SELECT `value` FROM `"+config.DBprefix+"info` WHERE `name` = 'version'",
		[]interface{}{}, []interface{}{&version})
	if err == sql.ErrNoRows {
		newInstall = true
	} else if err != nil {
		handleError(0, "failed: %s\n", customError(err))
		os.Exit(2)
	}

	if newInstall {
		printf(0, "\nThis looks like a new install or one that needs updating, setting up the database...")
		if _, err = db.Exec(
			"INSERT INTO `" + config.DBname + "`.`" + config.DBprefix + "staff` " +
				"(`username`, `password_checksum`, `salt`, `rank`) " +
				"VALUES ('admin', '" + bcryptSum("password") + "', 'abc', 3)",
		); err != nil {
			handleError(0, "failed with error: %s\n", customError(err))
			os.Exit(2)
		}
	}

	if sqlVersion != version {
		_, err = execSQL("INSERT INTO `"+config.DBprefix+"info` (`name`,`value`) VALUES('version',?)", version)
		if err != nil && !strings.Contains(err.Error(), "Duplicate entry") {
			handleError(0, "failed with error: %s\n", customError(err))
			os.Exit(2)
		}
	}
	checkDeprecatedSchema()
	println(0, "complete.")
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
	defer closeStatement(stmt)
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
	defer closeStatement(stmt)
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

// checkDeprecatedSchema checks the tables for outdated columns and column values
// and causes gochan to quit with an error message specific to the needed change
func checkDeprecatedSchema() {
	var hasColumn int
	var err error

	execSQL("ALTER TABLE `"+config.DBprefix+"banlist` CHANGE COLUMN `id` `id` INT(11) UNSIGNED NOT NULL AUTO_INCREMENT", nil)

	if err = queryRowSQL(
		"SELECT COUNT(*) FROM information_schema.COlUMNS WHERE `TABLE_SCHEMA` = '"+config.DBname+"' AND TABLE_NAME = '"+config.DBprefix+"banlist' AND COLUMN_NAME = 'appeal_message'",
		[]interface{}{}, []interface{}{&hasColumn},
	); err != nil {
		println(0, "error checking for deprecated column: "+err.Error())
		os.Exit(2)
		return
	}
	if hasColumn > 0 {
		// Running them one at a time, in case we get errors from individual queries
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` CHANGE COLUMN `banned_by` `staff` VARCHAR(50) NOT NULL", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `type` TINYINT UNSIGNED NOT NULL DEFAULT '3'", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `name_is_regex` TINYINT(1) DEFAULT '0'", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `filename` VARCHAR(255) NOT NULL DEFAULT ''", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `file_checksum` VARCHAR(255) NOT NULL DEFAULT ''", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `permaban` TINYINT(1) DEFAULT '0'", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` ADD COLUMN `can_appeal` TINYINT(1) DEFAULT '1'", nil)
		execSQL("ALTER TABLE `"+config.DBprefix+"banlist` DROP COLUMN `message`", nil)

		println(0, "The column `appeal_message` in table "+config.DBprefix+"banlist is deprecated. A new table , `"+config.DBprefix+"appeals` has been created for it, and the banlist table will be modified accordingly.")
		println(0, "Just to be safe, you may want to check both tables to make sure everything is good.")

		rows, err := querySQL("SELECT `id`,`appeal_message` FROM `" + config.DBprefix + "banlist`")
		if err != nil {
			println(0, "Error updating banlist schema: "+err.Error())
			os.Exit(2)
			return
		}

		for rows.Next() {
			var id int
			var appeal_message string
			rows.Scan(&id, &appeal_message)
			if appeal_message != "" {
				execSQL("INSERT INTO `"+config.DBprefix+"appeals` (`ban`,`message`) VALUES(?,?)", &id, &appeal_message)
			}
			execSQL("ALTER TABLE `" + config.DBprefix + "banlist` DROP COLUMN `appeal_message`")
		}
	}
}
