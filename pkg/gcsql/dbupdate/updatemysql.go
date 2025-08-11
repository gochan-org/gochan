package dbupdate

import (
	"context"
	"database/sql"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcsql/migrationutil"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func updateMysqlDB(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
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

	// fix default collation
	db, err := gcsql.GetDatabase()
	if err != nil {
		return err
	}
	query := `ALTER DATABASE ` + dbName + ` CHARACTER SET = utf8mb4 COLLATE = utf8mb4_unicode_ci`
	// use the base sql.DB, since our database automatically prepares all queries, and ALTER DATABASE
	// will cause a "This command is not supported in the prepared statement protocol yet" error
	if _, err = db.GetBaseDB().ExecContext(ctx, query); err != nil {
		return err
	}

	var rows *sql.Rows
	query = `SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE'`
	rows, err = gcsql.QueryContextSQL(ctx, nil, query, dbName)
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
		if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}

	// split DBPREFIXposts.ip into range_start and range_end as varbinary
	dataType, err := migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType != "" {
		// add range_start and range_end columns
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start VARBINARY(16) NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end VARBINARY(16) NOT NULL`
		if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		// convert ban IP string to IP range
		if rows, err = gcsql.QueryContextSQL(ctx, nil, "SELECT id, ip FROM DBPREFIXip_ban"); err != nil {
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
			if _, err = gcsql.ExecContextSQL(ctx, nil, query, rangeStart, rangeEnd, id); err != nil {
				return err
			}
			query = `ALTER TABLE DBPREFIXip_ban DROP COLUMN IF EXISTS ip`
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				return err
			}
		}
		if err = rows.Close(); err != nil {
			return err
		}
	}

	// Convert DBPREFIXposts.ip to from varchar to varbinary
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if migrationutil.IsStringType(dataType) {
		alterPostsIP := []string{
			"ALTER TABLE DBPREFIXposts CHANGE ip ip_str VARCHAR(45)",
			"ALTER TABLE DBPREFIXposts ADD COLUMN ip VARBINARY(16)",
			"UPDATE DBPREFIXposts SET ip = INET6_ATON(ip_str)",
			"ALTER TABLE DBPREFIXposts CHANGE ip ip VARBINARY(16) NOT NULL",
			"ALTER TABLE DBPREFIXposts DROP COLUMN ip_str",
		}
		for _, query = range alterPostsIP {
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				return err
			}
		}
	}

	// Convert DBPREFIXreports.ip to from varchar to varbinary
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXreports", sqlConfig)
	if err != nil {
		return err
	}
	if migrationutil.IsStringType(dataType) {
		alterReportsIP := []string{
			"ALTER TABLE DBPREFIXreports CHANGE ip ip_str VARCHAR(45)",
			"ALTER TABLE DBPREFIXreports ADD COLUMN ip VARBINARY(16)",
			"UPDATE DBPREFIXreports SET ip = INET6_ATON(ip_str)",
			"ALTER TABLE DBPREFIXreports CHANGE ip ip VARBINARY(45) NOT NULL",
			"ALTER TABLE DBPREFIXreports DROP COLUMN ip_str",
		}

		for _, query := range alterReportsIP {
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				errEv.Str("query", query)
				return err
			}
		}
	}

	// add flag column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "flag", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add country column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "country", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add is_secure_tripcode column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// add spoilered column to DBPREFIXthreads
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	// rename DBPREFIXposts.cyclical to cyclic
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "cyclical", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	cyclicType, err := migrationutil.ColumnType(ctx, nil, nil, "cyclic", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" && cyclicType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXthreads ADD COLUMN cyclic BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	} else if dataType != "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXthreads CHANGE cyclical cyclic BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	return nil
}
