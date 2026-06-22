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
			errEv.Discard()
		}
	}()

	_, err = gcsql.ExecContextSQL(ctx, nil, `PRAGMA foreign_keys = ON`)
	if err != nil {
		errEv.Err(err).Caller().Send()
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
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_secure_tripcode", "DBPREFIXposts", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN is_secure_tripcode BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "flag", "DBPREFIXposts", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN flag VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "country", "DBPREFIXposts", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXposts ADD COLUMN country VARCHAR(80) NOT NULL DEFAULT ''"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "expires", "DBPREFIXsessions", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXsessions ADD COLUMN expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "data", "DBPREFIXsessions", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXsessions ADD COLUMN data VARCHAR(45) NOT NULL DEFAULT ''"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	banRangeStringToBinaryStmts := []string{
		// used for if the column exists but is varchar(45) instead of varbinary(16)
		"DROP VIEW IF EXISTS DBPREFIXv_appeal_messages",
		`CREATE TEMPORARY TABLE tmp_ip_ban AS SELECT id, staff_id, board_id, banned_for_post_id,
			copy_post_text, is_thread_ban, is_active, issued_at, appeal_at, expires_at, permanent,
			staff_note, message, can_appeal FROM DBPREFIXip_ban`,
		"ALTER TABLE tmp_ip_ban ADD COLUMN range_start VARBINARY(16) NOT NULL DEFAULT ''",
		"ALTER TABLE tmp_ip_ban ADD COLUMN range_end VARBINARY(16) NOT NULL DEFAULT ''",
		"UPDATE tmp_ip_ban SET range_start = INET6_ATON((SELECT range_start FROM DBPREFIXip_ban WHERE DBPREFIXip_ban.id = tmp_ip_ban.id))",
		"UPDATE tmp_ip_ban SET range_end = INET6_ATON((SELECT range_end FROM DBPREFIXip_ban WHERE DBPREFIXip_ban.id = tmp_ip_ban.id))",
		"DROP TABLE DBPREFIXip_ban",
		"CREATE TABLE DBPREFIXip_ban AS SELECT * FROM tmp_ip_ban",
		"DROP TABLE tmp_ip_ban",
	}
	banRangeIpToBinaryStmts := []string{
		// used for if the ban table uses a single ip column (assumed to be varchar(45)) instead of range_start and range_end
		"DROP VIEW IF EXISTS DBPREFIXv_appeal_messages",
		`CREATE TEMPORARY TABLE tmp_ip_ban AS SELECT id, staff_id, board_id, banned_for_post_id,
			copy_post_text, is_thread_ban, is_active, issued_at, appeal_at, expires_at, permanent,
			staff_note, message, can_appeal FROM DBPREFIXip_ban`,
		"ALTER TABLE tmp_ip_ban ADD COLUMN range_start VARBINARY(16) NOT NULL DEFAULT ''",
		"ALTER TABLE tmp_ip_ban ADD COLUMN range_end VARBINARY(16) NOT NULL DEFAULT ''",
		"UPDATE tmp_ip_ban SET range_start = INET6_ATON((SELECT range_start FROM DBPREFIXip_ban WHERE DBPREFIXip_ban.id = tmp_ip_ban.id))",
		"UPDATE tmp_ip_ban SET range_end = INET6_ATON((SELECT range_end FROM DBPREFIXip_ban WHERE DBPREFIXip_ban.id = tmp_ip_ban.id))",
		"DROP TABLE DBPREFIXip_ban",
		"CREATE TABLE DBPREFIXip_ban AS SELECT * FROM tmp_ip_ban",
		"DROP TABLE tmp_ip_ban",
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "range_start", "DBPREFIXip_ban", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	switch dataType {
	case "VARCHAR(45)":
		// column stored as the old type (string), need to convert to VARBINARY(16) with sqlite3-inet6 extensions
		for _, stmt := range banRangeStringToBinaryStmts {
			if _, err = gcsql.Exec(opts, stmt); err != nil {
				errEv.Err(err).Caller().Str("failedStmt", stmt).Send()
				return err
			}
		}
	case "":
		// column doesn't exist, create it
		for _, stmt := range banRangeIpToBinaryStmts {
			if _, err = gcsql.Exec(opts, stmt); err != nil {
				errEv.Err(err).Caller().Str("failedStmt", stmt).Send()
				return err
			}
		}
	}

	dataType, err = migrationutil.ColumnType(ctx, nil, nil, "is_spoilered", "DBPREFIXthreads", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "" {
		if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXthreads ADD COLUMN is_spoilered BOOL NOT NULL DEFAULT FALSE"); err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
	}

	filtersExist, err := migrationutil.TableExists(ctx, nil, nil, "DBPREFIXfilters", sqlConfig)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if !filtersExist {
		// update pre-filter tables to make sure they can be migrated to the new filter tables

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN fingerprinter VARCHAR(64) DEFAULT ''"); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
		}

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip BOOL NOT NULL DEFAULT FALSE"); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
		}

		dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ban_ip_message", "DBPREFIXfile_ban", sqlConfig)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		if dataType == "" {
			if _, err = gcsql.Exec(opts, "ALTER TABLE DBPREFIXfile_ban ADD COLUMN ban_ip_message TEXT"); err != nil {
				errEv.Err(err).Caller().Send()
				return err
			}
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXposts", sqlConfig); err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}
	if dataType == "VARCHAR(45)" {
		ipStmts := []string{
			"DROP VIEW IF EXISTS DBPREFIXv_appeal_messages",
			`CREATE TEMPORARY TABLE tmp_posts AS SELECT id, thread_id, is_top_post, created_on, name, tripcode,
				is_secure_tripcode, is_role_signature, email, subject, message, message_raw, password, deleted_at,
				is_deleted, banned_message, flag, country FROM DBPREFIXposts`,
			"ALTER TABLE tmp_posts ADD COLUMN ip VARBINARY(16) NOT NULL DEFAULT ''",
			"UPDATE tmp_posts SET ip = INET6_ATON((SELECT ip FROM DBPREFIXposts WHERE DBPREFIXposts.id = tmp_posts.id))",
			"DROP TABLE DBPREFIXposts",
			"CREATE TABLE DBPREFIXposts AS SELECT * FROM tmp_posts",
			"DROP TABLE tmp_posts",
		}
		for _, stmt := range ipStmts {
			if _, err = gcsql.Exec(opts, stmt); err != nil {
				errEv.Err(err).Caller().Str("failedStmt", stmt).Send()
				return err
			}
		}
	}

	if dataType, err = migrationutil.ColumnType(ctx, nil, nil, "ip", "DBPREFIXreports", sqlConfig); err != nil {
		return err
	}
	if dataType == "VARCHAR(45)" {
		ipStmts := []string{
			"CREATE TEMPORARY TABLE tmp_reports AS SELECT id, handled_by_staff_id, post_id, reason, is_cleared, CURRENT_TIMESTAMP as timestamp FROM DBPREFIXreports",
			"ALTER TABLE tmp_reports ADD COLUMN ip VARBINARY(16) NOT NULL DEFAULT ''",
			"UPDATE tmp_reports SET ip = INET6_ATON((SELECT ip FROM DBPREFIXreports WHERE tmp_reports.id = DBPREFIXreports.id))",
			"DROP TABLE DBPREFIXreports",
			"CREATE TABLE DBPREFIXreports AS SELECT * FROM tmp_reports",
			"DROP TABLE tmp_reports",
		}
		for _, stmt := range ipStmts {
			if _, err = gcsql.Exec(opts, stmt); err != nil {
				errEv.Err(err).Caller().Str("failedStmt", stmt).Send()
				return err
			}
		}
	}

	return nil
}
