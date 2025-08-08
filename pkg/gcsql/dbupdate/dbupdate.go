package dbupdate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcsql/migrationutil"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

var (
	ErrInvalidVersion = errors.New("database contains database_version table but zero or more than one versions were found")
)

func UpdateDatabase() error {
	gcutil.LogInfo().Msg("Preparing to update the database")
	errEv := gcutil.LogError(nil)

	sqlConfig := config.GetSQLConfig()
	var gochanTablesExist bool

	db, err := gcsql.GetDatabase()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return err
	}

	if sqlConfig.DBprefix == "" {
		gochanTablesExist, err = migrationutil.TableExists(context.Background(), db, nil, "database_version", &sqlConfig)
	} else {
		gochanTablesExist, err = gcsql.DoesGochanPrefixTableExist()
	}
	if err != nil {
		return err
	}
	if !gochanTablesExist {
		return migrationutil.ErrNotInstalled
	}

	updated, err := isUpdated()
	defer func() {
		if a := recover(); a != nil {
			errEv.Caller(4).Interface("panic", a).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()
	if errors.Is(err, sql.ErrNoRows) {
		return gcsql.ErrInvalidVersion
	}
	if err != nil {
		return err
	}
	if updated {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var filterTableExists bool
	filterTableExists, err = migrationutil.TableExists(ctx, db, nil, "DBPREFIXfilters", &sqlConfig)
	if err != nil {
		return err
	}

	if !filterTableExists {
		// DBPREFIXfilters not found, create it and migrate data from DBPREFIXfile_ban, DBPREFIXfilename_ban, and DBPREFIXusername_ban,
		if err = addFilterTables(ctx, nil, &sqlConfig, errEv); err != nil {
			return err
		}
	}

	switch sqlConfig.DBtype {
	case "mysql":
		err = updateMysqlDB(ctx, &sqlConfig, errEv)
	case "postgres":
		err = updatePostgresDB(ctx, &sqlConfig, errEv)
	case "sqlite3":
		err = updateSqliteDB(ctx, &sqlConfig, errEv)
	}
	if err != nil {
		return err
	}

	if err = ctx.Err(); err != nil {
		return err
	}

	if err = updateFilters(ctx, &sqlConfig, errEv); err != nil {
		return err
	}

	query := `UPDATE DBPREFIXdatabase_version SET version = ? WHERE component = 'gochan'`
	_, err = gcsql.ExecContextSQL(ctx, nil, query, gcsql.DatabaseVersion)
	if err != nil {
		return err
	}

	gcutil.LogInfo().
		Int("DBVersion", gcsql.DatabaseVersion).
		Msg("Database updated successfully")
	return nil
}

func isUpdated() (bool, error) {
	var currentDatabaseVersion int
	err := gcsql.QueryRow(nil, "SELECT version FROM DBPREFIXdatabase_version WHERE component = 'gochan'", nil,
		[]any{&currentDatabaseVersion})
	if err != nil {
		return false, err
	}
	if currentDatabaseVersion == gcsql.DatabaseVersion {
		return true, nil
	}
	if currentDatabaseVersion > gcsql.DatabaseVersion {
		return false, fmt.Errorf("database layout is ahead of current version (%d), target version: %d",
			currentDatabaseVersion, gcsql.DatabaseVersion)
	}
	return false, nil
}

func updateFilters(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	var fileBansExist, filenameBansExist, usernameBansExist, wordfiltersExist bool

	fileBansExist, err = migrationutil.TableExists(ctx, nil, nil, "DBPREFIXfile_ban", sqlConfig)
	defer func() {
		if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()
	if err != nil {
		return err
	}

	filenameBansExist, err = migrationutil.TableExists(ctx, nil, nil, "DBPREFIXfilename_ban", sqlConfig)
	if err != nil {
		return err
	}

	usernameBansExist, err = migrationutil.TableExists(ctx, nil, nil, "DBPREFIXusername_ban", sqlConfig)
	if err != nil {
		return err
	}

	wordfiltersExist, err = migrationutil.TableExists(ctx, nil, nil, "DBPREFIXwordfilters", sqlConfig)
	if err != nil {
		return err
	}

	if fileBansExist {
		if err = updateFileBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if filenameBansExist {
		if err = updateFilenameBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if usernameBansExist {
		if err = updateUsernameBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if wordfiltersExist {
		if err = updateWordfilters(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	return nil
}

func updateFileBans(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := gcsql.BeginContextTx(ctx)
	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("recovered: %v", a)
			errEv.Caller(4).Err(err).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
		_ = tx.Rollback()
	}()
	if err != nil {
		return err
	}

	query := "SELECT board_id, staff_id, staff_note, issued_at, checksum, fingerprinter, ban_ip, ban_ip_message FROM DBPREFIXfile_ban"
	var fingerprinterCol string
	fingerprinterCol, err = migrationutil.ColumnType(ctx, nil, nil, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}

	if fingerprinterCol == "" {
		query = strings.ReplaceAll(query, "fingerprinter", "'checksum' AS fingerprinter")
	}

	var banIPCol string
	banIPCol, err = migrationutil.ColumnType(ctx, nil, nil, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if banIPCol == "" {
		query = strings.ReplaceAll(query, "ban_ip", "FALSE AS ban_ip")
	}

	rows, err := gcsql.QueryContextSQL(ctx, nil, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ban FileBan
		if err = rows.Scan(
			&ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum,
			&ban.Fingerprinter, &ban.BanIP, &ban.BanIPMessage,
		); err != nil {
			return err
		}

		filter := &gcsql.Filter{
			StaffID:     &ban.StaffID,
			StaffNote:   ban.StaffNote,
			IssuedAt:    ban.IssuedAt,
			MatchAction: "reject",
			IsActive:    true,
		}
		if ban.BanIP {
			filter.MatchAction = "ban"
			if ban.BanIPMessage != nil {
				filter.MatchDetail = *ban.BanIPMessage
			}
		}
		var boards []int
		if ban.BoardID != nil {
			boards = append(boards, *ban.BoardID)
		}

		condition := gcsql.FilterCondition{MatchMode: gcsql.ExactMatch, Search: ban.Checksum, Field: "checksum"}
		if ban.Fingerprinter != nil {
			condition.Field = *ban.Fingerprinter
			if condition.Field == "" {
				condition.Field = "checksum"
			}
		}

		if err = gcsql.ApplyFilterTx(ctx, tx, filter, []gcsql.FilterCondition{condition}, boards); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}

	return tx.Commit()
}

func updateFilenameBans(ctx context.Context, _ *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := gcsql.BeginContextTx(ctx)
	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("recovered: %v", a)
			errEv.Caller(4).Err(err).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
		_ = tx.Rollback()
	}()
	if err != nil {
		return err
	}

	query := "SELECT board_id, staff_id, staff_note, issued_at, filename, is_regex FROM DBPREFIXfilename_ban"
	rows, err := gcsql.QueryContextSQL(ctx, nil, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ban FilenameBan
		if err = rows.Scan(
			&ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Filename, &ban.IsRegex,
		); err != nil {
			return err
		}

		filter := &gcsql.Filter{
			StaffID:     &ban.StaffID,
			StaffNote:   ban.StaffNote,
			IssuedAt:    ban.IssuedAt,
			MatchAction: "reject",
			IsActive:    true,
			MatchDetail: "File rejected",
		}

		condition := gcsql.FilterCondition{MatchMode: gcsql.ExactMatch, Search: ban.Filename, Field: "filename"}
		if ban.IsRegex {
			condition.MatchMode = gcsql.RegexMatch
		}
		var boards []int
		if ban.BoardID != nil {
			boards = append(boards, *ban.BoardID)
		}
		if err = gcsql.ApplyFilterTx(ctx, tx, filter, []gcsql.FilterCondition{condition}, boards); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}
	return tx.Commit()
}

func updateUsernameBans(ctx context.Context, _ *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := gcsql.BeginContextTx(ctx)
	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("recovered: %v", a)
			errEv.Caller(4).Err(err).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
		_ = tx.Rollback()
	}()
	if err != nil {
		return err
	}

	query := "SELECT board_id, staff_id, staff_note, issued_at, username FROM DBPREFIXusername_ban"
	rows, err := gcsql.QueryContextSQL(ctx, nil, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var ban UsernameBan
		if err = rows.Scan(
			&ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Username,
		); err != nil {
			return err
		}

		filter := &gcsql.Filter{
			StaffID:     &ban.StaffID,
			StaffNote:   ban.StaffNote,
			IssuedAt:    ban.IssuedAt,
			MatchAction: "reject",
			IsActive:    true,
			MatchDetail: "Name rejected",
		}

		condition := gcsql.FilterCondition{MatchMode: gcsql.ExactMatch, Search: ban.Username, Field: "name"}
		if ban.IsRegex {
			condition.MatchMode = gcsql.RegexMatch
		}
		var boards []int
		if ban.BoardID != nil {
			boards = append(boards, *ban.BoardID)
		}
		if err = gcsql.ApplyFilterTx(ctx, tx, filter, []gcsql.FilterCondition{condition}, boards); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}
	return tx.Commit()
}

func updateWordfilters(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := gcsql.BeginContextTx(ctx)
	defer func() {
		if a := recover(); a != nil {
			err = fmt.Errorf("recovered: %v", a)
			errEv.Caller(4).Err(err).Send()
			errEv.Discard()
		} else if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
		_ = tx.Rollback()
	}()
	if err != nil {
		return err
	}

	query := "SELECT board_dirs, staff_id, staff_note, issued_at, search, is_regex, change_to FROM DBPREFIXwordfilters"
	var boardIDCol string
	boardIDCol, err = migrationutil.ColumnType(ctx, nil, nil, "board_id", "DBPREFIXwordfilters", sqlConfig)
	if err != nil {
		return err
	}
	if boardIDCol != "" {
		query = strings.ReplaceAll(query, "board_dirs", "board_id")
	}

	rows, err := gcsql.QueryContextSQL(ctx, nil, query)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var wf Wordfilter
		var boards []int
		if boardIDCol != "" {
			if err = rows.Scan(
				&wf.BoardID, &wf.StaffID, &wf.StaffNote, &wf.IssuedAt, &wf.Search, &wf.IsRegex, &wf.ChangeTo,
			); err != nil {
				return err
			}
			if wf.BoardID != nil {
				boards = append(boards, *wf.BoardID)
			}
		} else {
			if err = rows.Scan(
				&wf.BoardDirs, &wf.StaffID, &wf.StaffNote, &wf.IssuedAt, &wf.Search, &wf.IsRegex, &wf.ChangeTo,
			); err != nil {
				return err
			}
			if wf.BoardDirs != nil {
				boardDirsVal := *wf.BoardDirs
				if boardDirsVal != "" && boardDirsVal != "*" {
					boardDirs := strings.Split(*wf.BoardDirs, ",")
					for _, boardDir := range boardDirs {
						boardID, err := gcsql.GetBoardIDFromDir(strings.TrimSpace(boardDir))
						if err != nil {
							errEv.Str("boardDir", boardDir)
							return err
						}
						boards = append(boards, boardID)
					}
				}
			}
		}

		filter := &gcsql.Filter{
			StaffID:     &wf.StaffID,
			StaffNote:   wf.StaffNote,
			IssuedAt:    wf.IssuedAt,
			MatchAction: "replace",
			IsActive:    true,
			MatchDetail: wf.ChangeTo,
		}
		condition := gcsql.FilterCondition{MatchMode: gcsql.ExactMatch, Search: wf.Search, Field: "body"}
		if wf.IsRegex {
			condition.MatchMode = gcsql.RegexMatch
		}
		if err = gcsql.ApplyFilterTx(ctx, tx, filter, []gcsql.FilterCondition{condition}, boards); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}

	return tx.Commit()
}
