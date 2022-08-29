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
	gclog.Print(gclog.LStdLog, "Connected to database")
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

// RunSQLFile cuts a given sql file into individual statements and runs it.
func RunSQLFile(path string) error {
	sqlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(gcdb.replacer.Replace(sqlStr), ";")

	debugMode := config.GetDebugMode()
	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = gcdb.db.Exec(statement); err != nil {
				if debugMode {
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

// TODO: get gocha-migration working so this doesn't have to sit here
func tmpSqlAdjust() error {
	// first update the crappy wordfilter table structure
	var err error
	var query string
	if gcdb.driver == "mysql" {
		query = `SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
		WHERE CONSTRAINT_NAME = 'wordfilters_staff_id_fk'
		AND TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DBPREFIXwordfilters'`
		numConstraints := 3
		if err = gcdb.QueryRowSQL(query,
			interfaceSlice(),
			interfaceSlice(&numConstraints)); err != nil {
			return err
		}
		if numConstraints > 0 {
			query = `ALTER TABLE DBPREFIXwordfilters DROP FOREIGN KEY IF EXISTS wordfilters_board_id_fk`
		}
	} else {
		query = `ALTER TABLE DBPREFIXwordfilters DROP CONSTRAINT IF EXISTS board_id_fk`
	}
	if _, err = gcdb.ExecSQL(query); err != nil {
		return err
	}
	query = `ALTER TABLE DBPREFIXwordfilters DROP COLUMN IF EXISTS board_id`
	if _, err = gcdb.ExecSQL(query); err != nil {
		return err
	}
	query = `ALTER TABLE DBPREFIXwordfilters ADD COLUMN IF NOT EXISTS board_dirs varchar(255) DEFAULT '*'`
	_, err = gcdb.ExecSQL(query)
	return err
}
