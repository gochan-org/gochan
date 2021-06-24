package gcsql

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	gcdb *GCDB
	//FatalSQLFlags is used to log a fatal sql error and then close gochan
	FatalSQLFlags   = gclog.LErrorLog | gclog.LStdLog | gclog.LFatal
	tcpHostIsolator = regexp.MustCompile(`\b(tcp\()?([^\(\)]*)\b`)
)

// ConnectToDB initializes the database connection and exits if there are any errors
func ConnectToDB(host, driver, dbName, username, password, prefix string) {
	var err error
	if gcdb, err = Open(host, driver, dbName, username, password, prefix); err != nil {
		gclog.Print(FatalSQLFlags, "Failed to connect to the database: ", err.Error())
		return
	}
	gclog.Print(gclog.LStdLog|gclog.LErrorLog, "Connected to database")
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
	sqlArr := strings.Split(gcdb.replacer.Replace(sqlStr), ";")

	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = gcdb.db.Exec(statement); err != nil {
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
