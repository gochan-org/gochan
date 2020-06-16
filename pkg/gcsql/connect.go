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
	db       *sql.DB
	dbDriver string
	//FatalSQLFlags is used to log a fatal sql error and then close gochan
	FatalSQLFlags   = gclog.LErrorLog | gclog.LStdLog | gclog.LFatal
	nilTimestamp    string
	sqlReplacer     *strings.Replacer // used during SQL string preparation
	tcpHostIsolator = regexp.MustCompile(`\b(tcp\()?([^\(\)]*)\b`)
)

// ConnectToDB initializes the database connection and exits if there are any errors
func ConnectToDB(host string, dbType string, dbName string, username string, password string, prefix string) {
	var connStr string
	sqlReplacer = strings.NewReplacer(
		"DBNAME", dbName,
		"DBPREFIX", prefix,
		"\n", " ")
	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Initializing server...")

	addrMatches := tcpHostIsolator.FindAllStringSubmatch(host, -1)
	if len(addrMatches) > 0 && len(addrMatches[0]) > 2 {
		host = addrMatches[0][2]
	}

	switch dbType {
	case "mysql":
		connStr = fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&collation=utf8mb4_unicode_ci",
			username, password, host, dbName)
		nilTimestamp = "0000-00-00 00:00:00"
	case "postgres":
		connStr = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
			username, password, host, dbName)
		nilTimestamp = "0001-01-01 00:00:00"
	default:
		gclog.Printf(FatalSQLFlags,
			`Invalid DBtype %q in gochan.json, valid values are "mysql" and "postgres" (sqlite3 is no longer supported for stability reasons)`, dbType)
	}
	dbDriver = dbType
	var err error
	if db, err = sql.Open(dbType, connStr); err != nil {
		gclog.Print(FatalSQLFlags, "Failed to connect to the database: ", err.Error())
	}
	//TEMP
	// var temp = "oldDB" + dbType + ".sql"
	// runSQLFile(gcutil.FindResource(temp,
	// 	"/usr/local/share/gochan/"+temp,
	// 	"/usr/share/gochan/"+temp))
	// var temp2 = "olddbdummydata.sql"
	// runSQLFile(gcutil.FindResource(temp2,
	// 	"/usr/local/share/gochan/"+temp2,
	// 	"/usr/share/gochan/"+temp2))
	// err = migratePreApril2020Database(dbType)
	// os.Exit(0)
	//END TEMP
	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Connected to database...")
}

func initDB(initFile string) error {
	filePath := gcutil.FindResource(initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return fmt.Errorf(
			"SQL database initialization file (%s) missing. Please reinstall gochan", initFile)
	}
	return RunSQLFile(filePath)
}

//RunSQLFile cuts a given sql file into individual statements and runs it.
func RunSQLFile(path string) error {
	sqlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(sqlReplacer.Replace(sqlStr), ";")

	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = db.Exec(statement); err != nil {
				if config.Config.DebugMode {
					gclog.Printf(gclog.LStdLog, "Error excecuting sql: %s\n", err.Error())
					gclog.Printf(gclog.LStdLog, "Length sql: %d\n", len(statement))
					gclog.Printf(gclog.LStdLog, "Statement: %s\n", statement)
					fmt.Printf("%08b", []byte(statement))
				}
				return err
			}
		}
	}
	return nil
}
