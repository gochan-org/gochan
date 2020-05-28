package gcsql

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
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
	var gErr error
	if db, gErr = sql.Open(dbType, connStr); gErr != nil {
		gclog.Print(fatalSQLFlags, "Failed to connect to the database: ", gErr.Error())
	}

	err := handleVersioning(dbType)
	if err != nil {
		gclog.Print(fatalSQLFlags, "Failed to initialise database: ", err.Error())
	}

	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Finished initializing server...")
}

func initDB(initFile string) *gcutil.GcError {
	filePath := gcutil.FindResource(initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return gcutil.NewError(fmt.Sprintf(
			"SQL database initialization file (%s) missing. Please reinstall gochan", initFile), false)
	}

	sqlBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return gcutil.FromError(err, false)
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(sqlReplacer.Replace(sqlStr), ";")

	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = db.Exec(statement); err != nil {
				if config.Config.DebugMode {
					println("Error excecuting sql:")
					println(err.Error())
					println("Length sql: " + string(len(statement)))
					println(statement)
					fmt.Printf("%08b", []byte(statement))
				}
				return gcutil.FromError(err, false)
			}
		}
	}
	return nil
}
