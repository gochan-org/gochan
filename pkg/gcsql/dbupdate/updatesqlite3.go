package dbupdate

import (
	"context"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcsql/migrationutil"
	"github.com/rs/zerolog"
)

func updateSqliteDB(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	defer func() {
		if a := recover(); a != nil {
			errEv.Caller(4).Interface("panic", a).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()

	_, err = gcsql.ExecContextSQL(ctx, nil, `PRAGMA foreign_keys = ON`)
	if err != nil {
		return err
	}

	opts := &gcsql.RequestOptions{Context: ctx}

	// simple alterations first
	dataType, err := migrationutil.ColumnType(ctx, nil, nil, "cyclical", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType != "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXthreads RENAME COLUMN cyclical TO cyclic"); err != nil {
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE"); err != nil {
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "flag", "DBPREFIXposts", sqlConfig); err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "country", "DBPREFIXposts", sqlConfig); err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "expires", "DBPREFIXsessions", sqlConfig); err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXsessions ADD COLUMN expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"); err != nil {
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "data", "DBPREFIXsessions", sqlConfig); err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXsessions ADD COLUMN data VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "range_start", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXip_ban ADD COLUMN range_start VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if _, err = gcsql.Exec(opts, "UPDATE DBPREFIXip_ban SET range_start = ip"); err != nil {
			return err
		}
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "range_end", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXip_ban ADD COLUMN range_end VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			return err
		}
		if _, err = gcsql.Exec(opts, "UPDATE DBPREFIXip_ban SET range_end = ip"); err != nil {
			return err
		}
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE"); err != nil {
			return err
		}
	}

	filtersExist, err := migrationutil.TableExists(ctx, nil, nil, "DBPREFIXfilters", sqlConfig)
	if err != nil {
		return err
	}
	if !filtersExist {
		// update pre-filter tables to make sure they can be migrated to the new filter tables

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN fingerprinter VARCHAR(64) DEFAULT ''"); err != nil {
				return err
			}
		}

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip BOOL NOT NULL DEFAULT FALSE"); err != nil {
				return err
			}
		}

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ban_ip_message", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip_message TEXT"); err != nil {
				return err
			}
		}
	}

	return nil
}
