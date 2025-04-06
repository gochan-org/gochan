package gcupdate

import (
	"context"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/rs/zerolog"
)

func updatePostgresDB(ctx context.Context, dbu *GCDatabaseUpdater, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	db := dbu.db
	var query, dataType string

	dataType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXposts", sqlConfig)
	defer func() {
		if a := recover(); a != nil {
			errEv.Caller(4).Interface("panic", a).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()
	if err != nil {
		return err
	}
	if common.IsStringType(dataType) {
		// change ip column to temporary ip_str
		query = `ALTER TABLE DBPREFIXposts RENAME COLUMN ip TO ip_str,`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		// add ip column with INET type, default '127.0.0.1' because it throws an error otherwise
		// because it is non-nil
		query = `ALTER TABLE DBPREFIXposts
		ADD COLUMN IF NOT EXISTS ip INET NOT NULL DEFAULT '127.0.0.1'`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `UPDATE TABLE DBPREFIXposts SET ip = ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `ALTER TABLE DBPREFIXposts DROP COLUMN ip_str`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	dataType, err = common.ColumnType(ctx, db, nil, "ip", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType != "" {
		query = `ALTER TABLE DBPREFIXip_ban
		ADD COLUMN IF NOT EXISTS range_start INET NOT NULL,
		ADD COLUMN IF NOT EXISTS range_end INET NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}

		query = `UPDATE DBPREFIXip_ban SET range_start = ip::INET, SET range_end = ip::INET`
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

	// add is_secure_tripcode column to DBPREFIXposts
	dataType, err = common.ColumnType(ctx, db, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	return nil
}
