package gcsql

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	db            *sql.DB
	dbDriver      string
	fatalSQLFlags = gclog.LErrorLog | gclog.LStdLog | gclog.LFatal
	nilTimestamp  string
	sqlReplacer   *strings.Replacer // used during SQL string preparation
)

// ConnectToDB initializes the database connection and exits if there are any errors
func ConnectToDB(host string, dbType string, dbName string, username string, password string, prefix string) {
	var err error
	var connStr string
	sqlReplacer = strings.NewReplacer(
		"DBNAME", dbName,
		"DBPREFIX", prefix,
		"\n", " ")
	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Initializing server...")

	switch dbType {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@%s/%s?parseTime=true&collation=utf8mb4_unicode_ci",
			username, password, host, dbName)
		nilTimestamp = "0000-00-00 00:00:00"
	case "postgres":
		connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
			username, password, host, dbName)
		nilTimestamp = "0001-01-01 00:00:00"
	case "sqlite3":
		gclog.Print(gclog.LStdLog, "sqlite3 support is still flaky, consider using mysql or postgres")
		connStr = fmt.Sprintf("file:%s?mode=rwc&_auth&_auth_user=%s&_auth_pass=%s&cache=shared",
			host, username, password)
		nilTimestamp = "0001-01-01 00:00:00+00:00"
	default:
		gclog.Printf(fatalSQLFlags,
			`Invalid DBtype %q in gochan.json, valid values are "mysql", "postgres", and "sqlite3"`, dbType)
	}
	dbDriver = dbType
	if db, err = sql.Open(dbType, connStr); err != nil {
		gclog.Print(fatalSQLFlags, "Failed to connect to the database: ", err.Error())
	}

	if err = initDB("initdb_" + dbType + ".sql"); err != nil {
		gclog.Print(fatalSQLFlags, "Failed initializing DB: ", err.Error())
	}

	// Create generic "Main" section if one doesn't already exist
	if _, err = GetOrCreateDefaultSectionID(); err != nil {
		gclog.Print(fatalSQLFlags, "Failed initializing DB: ", err.Error())
	}
	//TODO fix new install thing once it works with existing database
	// var sqlVersionStr string
	// isNewInstall := false
	// if err = queryRowSQL("SELECT value FROM DBPREFIXinfo WHERE name = 'version'",
	// 	[]interface{}{}, []interface{}{&sqlVersionStr},
	// ); err == sql.ErrNoRows {
	// 	isNewInstall = true
	// } else if err != nil {
	// 	gclog.Print(lErrorLog|lStdLog|lFatal, "Failed initializing DB: ", err.Error())
	// }

	err = CreateDefaultBoardIfNoneExist()
	if err != nil {
		gclog.Print(fatalSQLFlags, "Failed creating default board: ", err.Error())
	}
	err = CreateDefaultAdminIfNoStaff()
	if err != nil {
		gclog.Print(fatalSQLFlags, "Failed creating default admin account: ", err.Error())
	}
	//fix versioning thing
}

func initDB(initFile string) error {
	var err error
	filePath := gcutil.FindResource(initFile,
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
