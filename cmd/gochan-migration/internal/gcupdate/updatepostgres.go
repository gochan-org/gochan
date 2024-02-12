package gcupdate

import (
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func updatePostgresDB(db *gcsql.GCDB, tx *sql.Tx, criticalCfg *config.SystemCriticalConfig) error {
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

	dataType, err := common.ColumnType(db, tx, "ip", "DBPREFIXposts", criticalCfg)
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

	dataType, err = common.ColumnType(db, tx, "ip", "DBPREFIXip_ban", criticalCfg)
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

	// add flag column to DBPREFIXposts
	dataType, err = common.ColumnType(db, tx, "flag", "DBPREFIXposts", criticalCfg)
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
	dataType, err = common.ColumnType(db, tx, "country", "DBPREFIXposts", criticalCfg)
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
	dataType, err = common.ColumnType(db, tx, "fingerprinter", "DBPREFIXfile_ban", criticalCfg)
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
	dataType, err = common.ColumnType(db, tx, "ban_ip", "DBPREFIXfile_ban", criticalCfg)
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
	dataType, err = common.ColumnType(db, tx, "ban_ip_message", "DBPREFIXfile_ban", criticalCfg)
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
