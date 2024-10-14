package gcupdate

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

type GCDatabaseUpdater struct {
	options *common.MigrationOptions
	db      *gcsql.GCDB
	// if the database version is less than TargetDBVer, it is assumed to be out of date, and the schema needs to be adjusted.
	// It is expected to be set by the build script
	TargetDBVer int
}

func (dbu *GCDatabaseUpdater) Init(options *common.MigrationOptions) error {
	dbu.options = options
	sqlCfg := config.GetSQLConfig()
	var err error
	dbu.db, err = gcsql.Open(&sqlCfg)
	return err
}

func (dbu *GCDatabaseUpdater) IsMigrated() (bool, error) {
	var currentDatabaseVersion int
	err := dbu.db.QueryRowSQL(`SELECT version FROM DBPREFIXdatabase_version WHERE component = 'gochan'`, nil,
		[]any{&currentDatabaseVersion})
	if err != nil {
		return false, err
	}
	if currentDatabaseVersion == dbu.TargetDBVer {
		return true, nil
	}
	if currentDatabaseVersion > dbu.TargetDBVer {
		return false, fmt.Errorf("database layout is ahead of current version (%d), target version: %d",
			currentDatabaseVersion, dbu.TargetDBVer)
	}
	return false, nil
}

func (dbu *GCDatabaseUpdater) MigrateDB() (migrated bool, err error) {
	errEv := common.LogError()

	gcsql.SetDB(dbu.db)
	migrated, err = dbu.IsMigrated()
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
		return migrated, err
	}
	if migrated {
		return migrated, nil
	}

	sqlConfig := config.GetSQLConfig()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var filterTableExists bool
	filterTableExists, err = common.TableExists(ctx, dbu.db, nil, "DBPREFIXfilters", &sqlConfig)
	if err != nil {
		return false, err
	}

	if !filterTableExists {
		// DBPREFIXfilters not found, create it and migrate data from DBPREFIXfile_ban, DBPREFIXfilename_ban, and DBPREFIXusername_ban,
		if err = addFilterTables(ctx, dbu.db, nil, &sqlConfig, errEv); err != nil {
			return false, err
		}
	}

	switch sqlConfig.DBtype {
	case "mysql":
		err = updateMysqlDB(ctx, dbu, &sqlConfig, errEv)
	case "postgres":
		err = updatePostgresDB(ctx, dbu, &sqlConfig, errEv)
	case "sqlite3":
		err = updateSqliteDB(ctx, dbu, &sqlConfig, errEv)
	}
	if err != nil {
		return false, err
	}

	if err = ctx.Err(); err != nil {
		return false, err
	}

	if err = dbu.migrateFilters(ctx, &sqlConfig, errEv); err != nil {
		return false, err
	}

	query := `UPDATE DBPREFIXdatabase_version SET version = ? WHERE component = 'gochan'`
	_, err = dbu.db.ExecContextSQL(ctx, nil, query, dbu.TargetDBVer)
	if err != nil {
		return false, err
	}

	return false, nil
}

func (dbu *GCDatabaseUpdater) migrateFilters(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	var fileBansExist, filenameBansExist, usernameBansExist, wordfiltersExist bool

	fileBansExist, err = common.TableExists(ctx, dbu.db, nil, "DBPREFIXfile_ban", sqlConfig)
	defer func() {
		if err != nil {
			errEv.Err(err).Caller(1).Send()
			errEv.Discard()
		}
	}()
	if err != nil {
		return err
	}

	filenameBansExist, err = common.TableExists(ctx, dbu.db, nil, "DBPREFIXfilename_ban", sqlConfig)
	if err != nil {
		return err
	}

	usernameBansExist, err = common.TableExists(ctx, dbu.db, nil, "DBPREFIXusername_ban", sqlConfig)
	if err != nil {
		return err
	}

	wordfiltersExist, err = common.TableExists(ctx, dbu.db, nil, "DBPREFIXwordfilters", sqlConfig)
	if err != nil {
		return err
	}

	if fileBansExist {
		if err = dbu.migrateFileBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if filenameBansExist {
		if err = dbu.migrateFilenameBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if usernameBansExist {
		if err = dbu.migrateUsernameBans(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	if wordfiltersExist {
		if err = dbu.migrateWordfilters(ctx, sqlConfig, errEv); err != nil {
			return err
		}
	}

	return nil
}

func (dbu *GCDatabaseUpdater) migrateFileBans(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := dbu.db.BeginTx(ctx, nil)
	defer func() {
		if a := recover(); a != nil {
			err = errors.New(fmt.Sprintf("recovered: %v", a))
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
	fingerprinterCol, err = common.ColumnType(ctx, dbu.db, nil, "fingerprinter", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}

	if fingerprinterCol == "" {
		query = strings.ReplaceAll(query, "fingerprinter", "'checksum' AS fingerprinter")
	}

	var banIPCol string
	banIPCol, err = common.ColumnType(ctx, dbu.db, nil, "ban_ip", "DBPREFIXfile_ban", sqlConfig)
	if err != nil {
		return err
	}
	if banIPCol == "" {
		query = strings.ReplaceAll(query, "ban_ip", "FALSE AS ban_ip")
	}

	rows, err := dbu.db.QueryContextSQL(ctx, nil, query)
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

func (dbu *GCDatabaseUpdater) migrateFilenameBans(ctx context.Context, _ *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := dbu.db.BeginTx(ctx, nil)
	defer func() {
		if a := recover(); a != nil {
			err = errors.New(fmt.Sprintf("recovered: %v", a))
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
	rows, err := dbu.db.QueryContextSQL(ctx, nil, query)
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

func (dbu *GCDatabaseUpdater) migrateUsernameBans(ctx context.Context, _ *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := dbu.db.BeginTx(ctx, nil)
	defer func() {
		if a := recover(); a != nil {
			err = errors.New(fmt.Sprintf("recovered: %v", a))
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
	rows, err := dbu.db.QueryContextSQL(ctx, nil, query)
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

func (dbu *GCDatabaseUpdater) migrateWordfilters(ctx context.Context, sqlConfig *config.SQLConfig, errEv *zerolog.Event) (err error) {
	tx, err := dbu.db.BeginTx(ctx, nil)
	defer func() {
		if a := recover(); a != nil {
			err = errors.New(fmt.Sprintf("recovered: %v", a))
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
	boardIDCol, err = common.ColumnType(ctx, dbu.db, nil, "board_id", "DBPREFIXwordfilters", sqlConfig)
	if err != nil {
		return err
	}
	if boardIDCol != "" {
		query = strings.ReplaceAll(query, "board_dirs", "board_id")
	}

	rows, err := dbu.db.QueryContextSQL(ctx, nil, query)
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

func (*GCDatabaseUpdater) MigrateBoards() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigratePosts() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateStaff(_ string) error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateBans() error {
	return gcutil.ErrNotImplemented
}

func (*GCDatabaseUpdater) MigrateAnnouncements() error {
	return gcutil.ErrNotImplemented
}

func (dbu *GCDatabaseUpdater) Close() error {
	if dbu.db != nil {
		return dbu.db.Close()
	}
	return nil
}
