package dbupdate

import (
	"context"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcsql/migrationutil"
	"github.com/rs/zerolog"
)

func updatePostgresDB(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	defer func() {
		if a := recover(); a != nil {
			errEv.Caller(4).Interface("panic", a).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()

	dataType, err := migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	var query string
	if migrationutil.IsStringType(dataType) {
		alterPostsIP := []string{
			"ALTER TABLE DBPREFIXposts RENAME COLUMN ip TO ip_str",
			"ALTER TABLE DBPREFIXposts ADD COLUMN IF NOT EXISTS ip INET",
			"UPDATE DBPREFIXposts SET ip = ip_str",
			"ALTER TABLE DBPREFIXposts DROP COLUMN ip_str",
			"ALTER TABLE DBPREFIXposts ALTER COLUMN ip SET NOT NULL",
		}
		for _, query = range alterPostsIP {
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				return err
			}
		}

		// change ip column to temporary ip_str
		query = `ALTER TABLE DBPREFIXposts RENAME COLUMN ip TO ip_str,`
		if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// Convert DBPREFIXreports.ip to from varchar to inet
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXreports", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if migrationutil.IsStringType(dataType) {
		alterReportsIP := []string{
			"ALTER TABLE DBPREFIXreports CHANGE ip ip_str varchar(45)",
			"ALTER TABLE DBPREFIXreports ADD COLUMN ip INET",
			"UPDATE DBPREFIXreports SET ip = ip_str",
			"ALTER TABLE DBPREFIXreports ALTER COLUMN ip SET NOT NULL",
			"ALTER TABLE DBPREFIXreports DROP COLUMN ip_str",
		}
		for _, query := range alterReportsIP {
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
		}
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType != "" {
		alterIPBanIP := []string{
			`ALTER TABLE DBPREFIXip_ban
				ADD COLUMN IF NOT EXISTS range_start INET,
				ADD COLUMN IF NOT EXISTS range_end INET`,
			"UPDATE DBPREFIXip_ban SET range_start = ip::INET, range_end = ip::INET",
			`ALTER TABLE DBPREFIXip_ban
				ALTER COLUMN range_start SET NOT NULL,
				ALTER COLUMN range_end SET NOT NULL`,
			"ALTER TABLE DBPREFIXip_ban DROP COLUMN ip",
		}
		for _, query = range alterIPBanIP {
			if _, err = gcsql.ExecContextSQL(ctx, nil, query); err != nil {
				return err
			}
		}
	}

	// add flag column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "flag", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// add country column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "country", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	// add is_secure_tripcode column to DBPREFIXposts
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE"); err != nil {
			return err
		}
	}

	// add spoilered column to DBPREFIXthreads
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE"); err != nil {
			return err
		}
	}

	// rename DBPREFIXthreads.cyclical to cyclic if cyclical exists
	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "cyclic", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXthreads RENAME cyclical TO cyclic"); err != nil {
			return err
		}
	}

	return nil
}
