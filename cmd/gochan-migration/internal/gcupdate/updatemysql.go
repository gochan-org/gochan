package gcupdate

import (
	"context"
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func updateMysqlDB(ctx context.Context, dbu *GCDatabaseUpdater, sqlConfig *config.SQLConfig, errEv *zerolog.Event) error {
	var query string
	var dataType string
	var err error
	defer func() {
		if err != nil {
			errEv.Err(err).Caller(1).Send()
		}
	}()
	dbName := sqlConfig.DBname
	db := dbu.db

	// fix default collation
	query = `ALTER DATABASE ` + dbName + ` CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci`
	if _, err = db.GetBaseDB().ExecContext(ctx, query); err != nil {
		return err
	}

	var rows *sql.Rows
	query = `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ?`
	rows, err = db.QueryContextSQL(ctx, nil, query, dbName)
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
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}

	dataType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType != "" {
		// add range_start and range_end columns
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start VARBINARY(16) NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		// convert ban IP string to IP range
		if rows, err = db.QueryContextSQL(ctx, nil, "SELECT id, ip FROM DBPREFIXip_ban"); err != nil {
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
			if _, err = db.ExecContextSQL(ctx, nil, query, rangeStart, rangeEnd, id); err != nil {
				return err
			}
			query = `ALTER TABLE DBPREFIXip_ban DROP COLUMN IF EXISTS ip`
			if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
				return err
			}
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}

	// Convert DBPREFIXposts.ip to from varchar to varbinary
	dataType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// rename `ip` to a temporary column to then be removed
		query = "ALTER TABLE DBPREFIXposts CHANGE ip ip_str varchar(45)"
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		// convert post IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXposts SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts DROP COLUMN IF EXISTS ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// Convert DBPREFIXreports.ip to from varchar to varbinary
	dataType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXreports", sqlConfig)
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// rename `ip` to a temporary column to then be removed
		query = "ALTER TABLE DBPREFIXreports CHANGE ip ip_str varchar(45)"
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXreports
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		// convert report IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXreports SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXreports DROP COLUMN IF EXISTS ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// add flag column to DBPREFIXposts
	dataType, err = common.ColumnType(ctx, db, nil, "flag", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// add country column to DBPREFIXposts
	dataType, err = common.ColumnType(ctx, db, nil, "country", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	return nil
}
