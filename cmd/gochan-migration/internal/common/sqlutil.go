package common

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	commentRemover = regexp.MustCompile("--.*\n?")
)

// ColumnType returns a string representation of the column's data type. It does not return an error
// if the column does not exist, instead returning an empty string.
func ColumnType(db *gcsql.GCDB, tx *sql.Tx, columnName string, tableName string, criticalCfg *config.SystemCriticalConfig) (string, error) {
	var query string
	var dataType string
	var err error
	var params []any
	tableName = strings.ReplaceAll(tableName, "DBPREFIX", criticalCfg.DBprefix)
	dbName := criticalCfg.DBname
	switch criticalCfg.DBtype {
	case "mysql":
		query = `SELECT DATA_TYPE FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND COLUMN_NAME = ? LIMIT 1`
		params = []any{dbName, tableName, columnName}
	case "postgresql":
		query = `SELECT data_type FROM information_schema.columns
		WHERE (table_schema = ? OR table_schema = 'public')
		AND table_name = ? AND column_name = ? LIMIT 1`
		params = []any{dbName, tableName, columnName}
	case "sqlite3":
		query = `SELECT type FROM  pragma_table_info(?) WHERE name = ?`
		params = []any{tableName, columnName}
	default:
		return "", gcsql.ErrUnsupportedDB
	}
	err = db.QueryRowTxSQL(tx, query, params, []any{&dataType})
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return dataType, err
}

// IsStringType returns true if the given column data type is TEXT or VARCHAR
func IsStringType(dataType string) bool {
	lower := strings.ToLower(dataType)
	return strings.HasPrefix(lower, "varchar") || lower == "text"
}

func RunSQLFile(path string, db *gcsql.GCDB) error {
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := commentRemover.ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(sqlStr, ";")

	for _, statement := range sqlArr {
		statement = strings.TrimSpace(statement)
		if len(statement) > 0 {
			if _, err = db.ExecSQL(statement); err != nil {
				return err
			}
		}
	}
	return nil
}

func InitDB(initFile string, db *gcsql.GCDB) error {
	filePath := gcutil.FindResource(initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return fmt.Errorf(
			"SQL database initialization file (%s) missing. Please reinstall gochan-migration", initFile)
	}

	return RunSQLFile(filePath, db)
}
