package gcsql

import (
	"os"
	"regexp"
	"strings"
)

var (
	tcpHostIsolator = regexp.MustCompile(`\b(tcp\()?([^\(\)]*)\b`)
)

// ConnectToDB initializes the database connection and exits if there are any errors
func ConnectToDB(host, driver, dbName, username, password, prefix string) error {
	var err error
	gcdb, err = Open(host, driver, dbName, username, password, prefix)
	return err
}

// RunSQLFile cuts a given sql file into individual statements and runs it.
func RunSQLFile(path string) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := regexp.MustCompile("--.*\n?").ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(gcdb.replacer.Replace(sqlStr), ";")

	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = ExecTxSQL(tx, statement); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
