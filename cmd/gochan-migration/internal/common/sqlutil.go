package common

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path"
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
func ColumnType(db *gcsql.GCDB, tx *sql.Tx, columnName string, tableName string, sqlConfig *config.SQLConfig) (string, error) {
	var query string
	var dataType string
	var err error
	var params []any
	tableName = strings.ReplaceAll(tableName, "DBPREFIX", sqlConfig.DBprefix)
	dbName := sqlConfig.DBname
	switch sqlConfig.DBtype {
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

// TableExists returns true if the given table exists in the given database, and an error if one occured
func TableExists(db *gcsql.GCDB, tx *sql.Tx, tableName string, sqlConfig *config.SQLConfig) (bool, error) {
	tableName = strings.ReplaceAll(tableName, "DBPREFIX", sqlConfig.DBprefix)
	dbName := sqlConfig.DBname
	var query string
	var params []any
	switch sqlConfig.DBtype {
	case "mysql":
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`
		params = []any{dbName, tableName}
	case "postgresql":
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE table_catalog = ? AND table_name = ?`
		params = []any{dbName, tableName}
	case "sqlite3":
		query = `SELECT COUNT(*) FROM sqlite_master WHERE name = ? AND type = 'table'`
		params = []any{tableName}
	default:
		return false, gcsql.ErrUnsupportedDB
	}
	var count int
	err := db.QueryRowTxSQL(tx, query, params, []any{&count})
	return count == 1, err
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

func getInitFilePath(initFile string) (string, error) {
	filePath := gcutil.FindResource(initFile,
		path.Join("./sql", initFile),
		path.Join("/usr/local/share/gochan", initFile),
		path.Join("/usr/share/gochan", initFile))
	if filePath == "" {
		return "", fmt.Errorf("missing SQL database initialization file (%s), please reinstall gochan", initFile)
	}
	return filePath, nil
}

func InitDB(initFile string, db *gcsql.GCDB) error {
	filePath, err := getInitFilePath(initFile)
	if err != nil {
		return err
	}

	return RunSQLFile(filePath, db)
}
