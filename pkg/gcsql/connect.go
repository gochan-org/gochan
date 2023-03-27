package gcsql

import (
	"os"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
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

// TODO: get gochan-migration working so this doesn't have to sit here
func tmpSqlAdjust() error {
	// first update the crappy wordfilter table structure
	var err error
	var query string
	switch gcdb.driver {
	case "mysql":
		query = `SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
		WHERE CONSTRAINT_NAME = 'wordfilters_board_id_fk'
		AND TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DBPREFIXwordfilters'`
		var numConstraints int
		if err = gcdb.QueryRowSQL(query,
			interfaceSlice(),
			interfaceSlice(&numConstraints)); err != nil {
			return err
		}
		if numConstraints > 0 {
			query = `ALTER TABLE DBPREFIXwordfilters DROP FOREIGN KEY wordfilters_board_id_fk`
		} else {
			query = ""
		}
		query = `SELECT COUNT(*) FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = 'DBPREFIXwordfilters'
		AND COLUMN_NAME = 'board_dirs'`
		var numColumns int
		if err = gcdb.QueryRowSQL(query,
			interfaceSlice(),
			interfaceSlice(&numColumns)); err != nil {
			return err
		}
		if numColumns == 0 {
			query = `ALTER TABLE DBPREFIXwordfilters ADD COLUMN board_dirs varchar(255) DEFAULT '*'`
			if _, err = ExecSQL(query); err != nil {
				return err
			}
		}

		// Yay, collation! Everybody loves MySQL's default collation!
		criticalConfig := config.GetSystemCriticalConfig()
		query = `ALTER DATABASE ` + criticalConfig.DBname + ` CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci`
		if _, err = gcdb.db.Exec(query); err != nil {
			return err
		}

		rows, err := QuerySQL(
			`SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`,
			criticalConfig.DBname)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var tableName string
			err = rows.Scan(&tableName)
			if err != nil {
				return err
			}
			query = `ALTER TABLE ` + tableName + ` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`
			if _, err = gcdb.db.Exec(query); err != nil {
				return err
			}
		}
		err = nil
	case "postgres":
		_, err = ExecSQL(`ALTER TABLE DBPREFIXwordfilters DROP CONSTRAINT IF EXISTS board_id_fk`)
		if err != nil {
			return err
		}
		_, err = ExecSQL(`ALTER TABLE DBPREFIXwordfilters ADD COLUMN IF NOT EXISTS board_dirs varchar(255) DEFAULT '*'`)
	case "sqlite3":
		_, err = ExecSQL(`PRAGMA foreign_keys = ON`)
		if err != nil {
			return err
		}
		_, err = ExecSQL(`ALTER TABLE DBPREFIXwordfilters ADD COLUMN IF NOT EXISTS board_dirs varchar(255) DEFAULT '*'`)
	}

	return err
}
