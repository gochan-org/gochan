package gcupdate

import (
	"context"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/rs/zerolog"
)

func updateSqliteDB(ctx context.Context, dbu *GCDatabaseUpdater, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	db := dbu.db
	var query string

	_, err = db.ExecContextSQL(ctx, nil, `PRAGMA foreign_keys = ON`)
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

	var dataType string
	// Add range_start column to DBPREFIXIp_ban if it doesn't exist
	dataType, err = common.ColumnType(ctx, db, nil, "DBPREFIXip_ban", "range_start", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXip_ban ADD COLUMN range_start VARCHAR(45) NOT NULL`
		if _, err = db.ExecTxSQL(nil, query); err != nil {
			return err
		}
	}

	// Add range_start column if it doesn't exist
	dataType, err = common.ColumnType(ctx, db, nil, "DBPREFIXip_ban", "range_end", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXip_ban ADD COLUMN range_end VARCHAR(45) NOT NULL`
		if _, err = db.ExecTxSQL(nil, query); err != nil {
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

	// add fingerprinter column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(ctx, db, nil, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN fingerprinter VARCHAR(64)`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// add ban_ip column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(ctx, db, nil, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip BOOL NOT NULL`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// add ban_ip_message column to DBPREFIXfile_ban
	dataType, err = common.ColumnType(ctx, db, nil, "ban_ip_message", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip_message TEXT`
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

	// add spoilered column to DBPREFIXthreads
	dataType, err = common.ColumnType(ctx, db, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			return err
		}
	}

	// rename DBPREFIXposts.cyclical to cyclic
	dataType, err = common.ColumnType(ctx, db, nil, "cyclic", "DBPREFIXposts", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		query = `ALTER TABLE DBPREFIXposts CHANGE cyclical cyclic BOOL NOT NULL DEFAULT FALSE`
		if _, err = db.ExecContextSQL(ctx, nil, query); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	return nil
}
