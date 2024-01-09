package gcupdate

import (
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func updatePostgresDB(db *gcsql.GCDB, tx *sql.Tx, dbName string, dbType string) error {
	query := `ALTER TABLE DBPREFIXwordfilters
	DROP CONSTRAINT IF EXISTS board_id_fk`
	_, err := db.ExecSQL(query)
	if err != nil {
		return err
	}
	query = `ALTER TABLE DBPREFIXwordfilters
	ADD COLUMN IF NOT EXISTS board_dirs varchar(255) DEFAULT '*'`
	if _, err = db.ExecTxSQL(tx, query); err != nil {
		return err
	}

	dataType, err := common.ColumnType(db, tx, "ip", "DBPREFIXposts", dbName, dbType)
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// change ip column to temporary ip_str
		query = `ALTER TABLE DBPREFIXposts RENAME COLUMN ip TO ip_str,`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		// add ip column with INET type, default '127.0.0.1' because it throws an error otherwise
		// because it is non-nil
		query = `ALTER TABLE DBPREFIXposts
		ADD COLUMN IF NOT EXISTS ip INET NOT NULL DEFAULT '127.0.0.1'`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `UPDATE TABLE DBPREFIXposts SET ip = ip_str`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts DROP COLUMN ip_str`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	dataType, err = common.ColumnType(db, tx, "ip", "DBPREFIXip_ban", dbName, dbType)
	if err != nil {
		return err
	}
	if dataType != "" {
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start INET NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end INET NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `UPDATE DBPREFIXip_ban SET range_start = ip::INET, SET range_end = ip::INET`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}
	return nil
}
