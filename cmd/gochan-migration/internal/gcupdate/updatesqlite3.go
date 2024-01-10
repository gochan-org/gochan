package gcupdate

import (
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func updateSqliteDB(db *gcsql.GCDB, tx *sql.Tx, criticalCfg *config.SystemCriticalConfig) error {
	_, err := db.ExecSQL(`PRAGMA foreign_keys = ON`)
	if err != nil {
		return err
	}
	dataType, err := common.ColumnType(db, tx, "DBPREFIXwordfilters", "board_dirs", criticalCfg)
	if err != nil {
		return err
	}
	if dataType == "" {
		query := `ALTER TABLE DBPREFIXwordfilters ADD COLUMN board_dirs varchar(255) DEFAULT '*'`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}
	return nil
}
