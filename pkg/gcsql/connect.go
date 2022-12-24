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

	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = gcdb.db.Exec(statement); err != nil {
				return err
			}
		}
	}
	return nil
}

// TODO: get gochan-migration working so this doesn't have to sit here
func tmpSqlAdjust() error {
	// first update the crappy wordfilter table structure
	var err error
	var query string
	switch gcdb.driver {
	case "mysql":
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
		} else {
			query = ""
		}
	case "postgres":
		query = `ALTER TABLE DBPREFIXwordfilters DROP CONSTRAINT IF EXISTS board_id_fk`
	case "sqlite3":
		_, err = ExecSQL(`PRAGMA foreign_keys = ON`)
		return err
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
