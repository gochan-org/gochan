package gcupdate

import (
	"context"
	"database/sql"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func updateMysqlDB(ctx context.Context, dbu *GCDatabaseUpdater, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	var query string
	var cyclicalType string
	defer func() {
		if a := recover(); a != nil {
			errEv.Caller(4).Interface("panic", a).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()
	dbName := sqlConfig.DBname
	db := dbu.db

	// fix default collation
	query = `ALTER DATABASE ` + dbName + ` CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci`
	if _, err = db.GetBaseDB().ExecContext(ctx, query); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}

	var rows *sql.Rows
	query = `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'`
	rows, err = db.QueryContextSQL(ctx, nil, query, dbName)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	defer func() {
		rows.Close()
	}()
	var tableName string
	for rows.Next() {
		err = rows.Scan(&tableName)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		query = `ALTER TABLE ` + tableName + ` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}

	cyclicalType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if cyclicalType != "" {
		// add range_start and range_end columns
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start VARBINARY(16) NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		// convert ban IP string to IP range
		if rows, err = db.QueryContextSQL(ctx, nil, "SELECT id, ip FROM DBPREFIXip_ban"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		var rangeStart, rangeEnd string
		for rows.Next() {
			var id int
			var ipOrCIDR string
			if err = rows.Scan(&id, &ipOrCIDR); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
			if rangeStart, rangeEnd, err = gcutil.ParseIPRange(ipOrCIDR); err != nil {
				return err
			}
			query = `UPDATE DBPREFIXip_ban
			SET range_start = INET6_ATON(?), range_end = INET6_ATON(?) WHERE id = ?`
			if _, err = db.ExecContextSQL(ctx, nil, query, rangeStart, rangeEnd, id); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
			query = `ALTER TABLE DBPREFIXip_ban DROP COLUMN IF EXISTS ip`
			if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
		}
		if err = rows.Close(); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// Convert DBPREFIXposts.ip to from varchar to varbinary
	cyclicalType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if common.IsStringType(cyclicalType) {
		// rename `ip` to a temporary column to then be removed
		query = "ALTER TABLE DBPREFIXposts CHANGE ip ip_str varchar(45)"
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		query = `ALTER TABLE DBPREFIXposts
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		// convert post IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXposts SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		query = `ALTER TABLE DBPREFIXposts DROP COLUMN IF EXISTS ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// Convert DBPREFIXreports.ip to from varchar to varbinary
	cyclicalType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXreports", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if common.IsStringType(cyclicalType) {
		// rename `ip` to a temporary column to then be removed
		query = "ALTER TABLE DBPREFIXreports CHANGE ip ip_str varchar(45)"
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		query = `ALTER TABLE DBPREFIXreports
		ADD COLUMN IF NOT EXISTS ip VARBINARY(16) NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		// convert report IP VARCHAR(45) to VARBINARY(16)
		query = `UPDATE DBPREFIXreports SET ip = INET6_ATON(ip_str)`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}

		query = `ALTER TABLE DBPREFIXreports DROP COLUMN IF EXISTS ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add flag column to DBPREFIXposts
	cyclicalType, err = common.ColumnType(ctx, db, nil, "flag", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if cyclicalType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add country column to DBPREFIXposts
	cyclicalType, err = common.ColumnType(ctx, db, nil, "country", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if cyclicalType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add is_secure_tripcode column to DBPREFIXposts
	cyclicalType, err = common.ColumnType(ctx, db, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if cyclicalType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add spoilered column to DBPREFIXthreads
	cyclicalType, err = common.ColumnType(ctx, db, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if cyclicalType == "" {
		query = `ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// rename DBPREFIXposts.cyclical to cyclic
	cyclicalType, err = common.ColumnType(ctx, db, nil, "cyclical", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	cyclicType, err := common.ColumnType(ctx, db, nil, "cyclic", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if cyclicalType == "" && cyclicType == "" {
		query = `ALTER TABLE DBPREFIXthreads ADD COLUMN cyclic BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	} else if cyclicalType != "" {
		query = `ALTER TABLE DBPREFIXthreads CHANGE cyclical cyclic BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	return nil
}
