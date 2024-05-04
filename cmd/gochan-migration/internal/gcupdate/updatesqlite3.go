package gcupdate

import (
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func updateSqliteDB(db *gcsql.GCDB, tx *sql.Tx, sqlConfig *config.SQLConfig) error {
	var query string
	_, err := db.ExecSQL(`PRAGMA foreign_keys = ON`)
	if err != nil {
		return err
	}
	dataType, err := common.ColumnType(db, tx, "DBPREFIXwordfilters", "board_dirs", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXwordfilters ADD COLUMN board_dirs varchar(255) DEFAULT '*'`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// Add range_start column to DBPREFIXIp_ban if it doesn't exist
	dataType, err = common.ColumnType(db, tx, "DBPREFIXip_ban", "range_start", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXip_ban ADD COLUMN range_start VARCHAR(45) NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// Add range_start column if it doesn't exist
	dataType, err = common.ColumnType(db, tx, "DBPREFIXip_ban", "range_end", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXip_ban ADD COLUMN range_end VARCHAR(45) NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// add flag column to DBPREFIXposts
	dataType, err = common.ColumnType(db, tx, "flag", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// add country column to DBPREFIXposts
	dataType, err = common.ColumnType(db, tx, "country", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// add fingerprinter column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(db, tx, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN fingerprinter VARCHAR(64)`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// add ban_ip column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(db, tx, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip BOOL NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// add ban_ip_message column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(db, tx, "ban_ip_message", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip_message TEXT`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	return nil
}
