package gcupdate

import (
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

func updateMysqlDB(db *gcsql.GCDB, tx *sql.Tx, criticalCfg *config.SystemCriticalConfig) error {
	var numConstraints int
	var err error
	dbName := criticalCfg.DBname
	query := `SELECT COUNT(*) FROM information_schema.TABLE_CONSTRAINTS
	WHERE CONSTRAINT_NAME = 'wordfilters_board_id_fk'
	AND TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'DBPREFIXwordfilters'`

	if err = db.QueryRowTxSQL(tx, query, nil, []any{&numConstraints}); err != nil {
		return err
	}
	if numConstraints > 0 {
		query = `ALTER TABLE DBPREFIXwordfilters DROP FOREIGN KEY wordfilters_board_id_fk`
	} else {
		query = ""
	}
	dataType, err := common.ColumnType(db, tx, "board_dirs", "DBPREFIXwordfilters", criticalCfg)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXwordfilters ADD COLUMN board_dirs varchar(255) DEFAULT '*'`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// Yay, collation! Everybody loves MySQL's default collation!
	query = `ALTER DATABASE ` + dbName + ` CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci`
	if _, err = tx.Exec(query); err != nil {
		return err
	}

	query = `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`
	rows, err := db.QuerySQL(query, dbName)
	if err != nil {
		return err
	}
	defer func() {
		rows.Close()
	}()
	var tableName string
	for rows.Next() {
		err = rows.Scan(&tableName)
		if err != nil {
			return err
		}
		query = `ALTER TABLE ` + tableName + ` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`
		if _, err = tx.Exec(query); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}
	dataType, err = common.ColumnType(db, tx, "ip", "DBPREFIXip_ban", criticalCfg)
	if err != nil {
		return err
	}
	if dataType != "" {
		// add range_start and range_end columns
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start VARBINARY(16) NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end VARBINARY(16) NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
		// convert ban IP string to IP range
		if rows, err = db.QuerySQL(`SELECT id, ip FROM DBPREFIXip_ban`); err != nil {
			return err
		}
		var rangeStart, rangeEnd string
		for rows.Next() {
			var id int
			var ipOrCIDR string
			if err = rows.Scan(&id, &ipOrCIDR); err != nil {
				return err
			}
			if rangeStart, rangeEnd, err = gcutil.ParseIPRange(ipOrCIDR); err != nil {
				return err
			}
			query = `UPDATE DBPREFIXip_ban
			SET range_start = INET6_ATON(?), range_end = INET6_ATON(?) WHERE id = ?`
			if _, err = db.ExecTxSQL(tx, query, rangeStart, rangeEnd, id); err != nil {
				return err
			}
			query = `ALTER TABLE DBPREFIXip_ban DROP COLUMN IF EXISTS ip`
			if _, err = db.ExecTxSQL(tx, query); err != nil {
				return err
			}
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}

	// Convert DBPREFIXposts.ip to from varchar to varbinary
	dataType, err = common.ColumnType(db, tx, "ip", "DBPREFIXposts", criticalCfg)
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// rename `ip` to a temporary column to then be removed
		query = `ALTER TABLE DBPREFIXposts CHANGE ip ip_str varchar(45)`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		// convert post IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXposts SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts DROP COLUMN IF EXISTS ip_str`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}
	}

	// Convert DBPREFIXreports.ip to from varchar to varbinary
	dataType, err = common.ColumnType(db, tx, "ip", "DBPREFIXreports", criticalCfg)
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// rename `ip` to a temporary column to then be removed
		query = `ALTER TABLE DBPREFIXreports CHANGE ip ip_str varchar(45)`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXreports
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		// convert report IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXreports SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecTxSQL(tx, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXreports DROP COLUMN IF EXISTS ip_str`
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

	return nil
}
