package common

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

// AddFilterTables is used for the db version 4 upgrade to create the filter tables from the respective SQL init file
func AddFilterTables(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, sqlConfig *config.SQLConfig) error {
	filePath, err := getInitFilePath("initdb_" + sqlConfig.DBtype + ".sql")
	if err != nil {
		return err
	}
	ba, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	sqlStr := commentRemover.ReplaceAllString(string(ba), " ")
	sqlArr := strings.Split(sqlStr, ";")

	for _, stmtStr := range sqlArr {
		stmtStr = strings.TrimSpace(stmtStr)
		if !strings.HasPrefix(stmtStr, "CREATE TABLE DBPREFIXfilter") {
			continue
		}
		if _, err = db.ExecContextSQL(ctx, tx, stmtStr); err != nil {
			return err
		}
	}
	return nil
}

// MigrateFileBans migrates file checksum and image fingerprint bans to the filter table
func MigrateFileBans(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, cfg *config.SQLConfig) error {
	fileBanTableExists, err := TableExists(ctx, db, tx, "DBPREFIXfilename_ban", cfg)
	if err != nil {
		return err
	}
	if !fileBanTableExists {
		// no filename bans to migrate (database partially migrated?)
		return nil
	}
	rows, err := db.QueryContextSQL(ctx, tx, `SELECT board_id,staff_id,staff_note,issued_at,checksum,fingerprinter,ban_ip,ban_ip_message FROM DBPREFIXfile_ban`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var fBanBoardID *int
	var fBanStaffID int
	var fBanStaffNote string
	var fBanIssuedAt time.Time
	var fBanChecksum string
	var fBanFingerprinter *string
	var fBanBanIP bool
	var fBanBanIPMessage *string

	var matchAction string
	var detail string
	var filterID int
	var field string
	for rows.Next() {
		if err = rows.Scan(
			&fBanBoardID, &fBanStaffID, &fBanStaffNote, &fBanIssuedAt, &fBanChecksum, &fBanFingerprinter, &fBanBanIP, &fBanBanIPMessage,
		); err != nil {
			return err
		}
		if fBanBanIP {
			matchAction = "ban"
		} else {
			matchAction = "reject"
		}
		if fBanBanIPMessage == nil {
			detail = ""
		} else {
			detail = *fBanBanIPMessage
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail, is_active) VALUES(?,?,?,?,?,?)`,
			fBanStaffID, fBanStaffNote, fBanIssuedAt, matchAction, detail, true); err != nil {
			return err
		}
		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}
		if fBanBoardID != nil {
			if _, err = db.ExecContextSQL(ctx, tx,
				`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES(?,?)`, filterID, *fBanBoardID,
			); err != nil {
				return err
			}
		}
		if fBanFingerprinter != nil {
			field = *fBanFingerprinter
		}
		if field == "" {
			field = "checksum"
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,match_mode,search,field) VALUES(?,?,?,?)`, filterID, gcsql.ExactMatch, fBanChecksum, field,
		); err != nil {
			return err
		}
	}
	return rows.Close()
}

// MigrateFilenameBans migrates filename bans to the filter table
func MigrateFilenameBans(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, cfg *config.SQLConfig) error {
	filenameBanTableExists, err := TableExists(ctx, db, tx, "DBPREFIXfilename_ban", cfg)
	if err != nil {
		return err
	}
	if !filenameBanTableExists {
		// no filename bans to migrate (database partially migrated?)
		return nil
	}
	rows, err := db.QueryContextSQL(ctx, tx, `SELECT board_id,staff_id,staff_note,issued_at,filename,is_regex FROM DBPREFIXfilename_ban`)
	if err != nil {
		fmt.Println("query error")
		return err
	}
	defer rows.Close()

	var fnBanBoardID *int
	var fnBanStaffID int
	var fnBanStaffNote string
	var fnBanIssuedAt time.Time
	var fnBanFilename string
	var fnBanIsRegex bool
	var filterID int
	var matchMode gcsql.StringMatchMode
	for rows.Next() {
		if err = rows.Scan(
			&fnBanBoardID, &fnBanStaffID, &fnBanStaffNote, &fnBanIssuedAt, &fnBanFilename, &fnBanIsRegex,
		); err != nil {
			return err
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail, is_active) VALUES(?,?,?,?,?,?)`,
			fnBanStaffID, fnBanStaffNote, fnBanIssuedAt, "reject", "", true,
		); err != nil {
			return err
		}

		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}

		if fnBanBoardID != nil {
			if _, err = db.ExecContextSQL(ctx, tx,
				`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES(?,?)`, filterID, *fnBanBoardID,
			); err != nil {
				return err
			}
		}

		if fnBanIsRegex {
			matchMode = gcsql.RegexMatch
		} else {
			matchMode = gcsql.SubstrMatch
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,match_mode,search,field) VALUES(?,?,?,?)`,
			filterID, matchMode, fnBanFilename, "filename",
		); err != nil {
			return err
		}
	}
	return rows.Close()
}

// MigrateUsernameBans migrates poster name bans to the filter table
func MigrateUsernameBans(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, cfg *config.SQLConfig) error {
	usernameBanTableExists, err := TableExists(ctx, db, tx, "DBPREFIXusername_ban", cfg)
	if err != nil {
		return err
	}
	if !usernameBanTableExists {
		// no name bans to migrate (database partially migrated?)
		return nil
	}
	rows, err := db.QueryContextSQL(ctx, tx, `SELECT board_id,staff_id,staff_note,issued_at,username,is_regex FROM DBPREFIXusername_ban`)
	if err != nil {
		fmt.Println("MigrateUsernameBans rows error")
		return err
	}
	defer rows.Close()

	var unBanBoardID *int
	var unBanStaffID int
	var unBanStaffNote string
	var unBanIssuedAt time.Time
	var unBanUsername string
	var unBanIsRegex bool
	var filterID int
	var matchMode gcsql.StringMatchMode
	for rows.Next() {
		if err = rows.Scan(
			&unBanBoardID, &unBanStaffID, &unBanStaffNote, &unBanIssuedAt, &unBanUsername, &unBanIsRegex,
		); err != nil {
			return err
		}

		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail, is_active) VALUES(?,?,?,?,?,?)`,
			unBanStaffID, unBanStaffNote, unBanIssuedAt, "reject", "", true,
		); err != nil {
			return err
		}

		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}

		if unBanBoardID != nil {
			if _, err = db.ExecContextSQL(ctx, tx,
				`INSERT INTO DBPREFIXfilter_boards(filter_id, board_id) VALUES(?,?)`, filterID, *unBanBoardID,
			); err != nil {
				return err
			}
		}
		if unBanIsRegex {
			matchMode = gcsql.RegexMatch
		} else {
			matchMode = gcsql.SubstrMatch
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,match_mode,search,field) VALUES(?,?,?,?)`,
			filterID, matchMode, unBanUsername, "name",
		); err != nil {
			return err
		}
	}
	return rows.Close()
}

// MigrateWordfilters migrates pre-filter wordfilters to the filter table
func MigrateWordfilters(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, sqlConfig *config.SQLConfig) error {
	wordfiltersTableExists, err := TableExists(ctx, db, tx, "DBPREFIXwordfilters", sqlConfig)
	if err != nil {
		return err
	}
	if !wordfiltersTableExists {
		// no wordfilters to migrate (database partially migrated?)
		return nil
	}
	rows, err := db.QueryContextSQL(ctx, tx, `SELECT board_dirs, staff_id, staff_note, issued_at, search, is_regex, change_to FROM DBPREFIXwordfilters`)
	if err != nil {
		fmt.Println("MigrateWordfilters rows error")
		return err
	}
	defer rows.Close()

	var boardDirsPtr *string
	var boardDirs []string
	var boardID int
	var staffID int
	var staffNote string
	var issuedAt time.Time
	var search string
	var isRegex bool
	var changeTo string
	var filterID int
	var matchMode gcsql.StringMatchMode
	for rows.Next() {
		if err = rows.Scan(&boardDirsPtr, &staffID, &staffNote, &issuedAt, &search, &isRegex, &changeTo); err != nil {
			return err
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail, is_active) VALUES(?,?,?,'replace',?,TRUE)`,
			staffID, staffNote, issuedAt, changeTo,
		); err != nil {
			return err
		}

		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}

		if boardDirsPtr != nil {
			boardDirs = strings.Split(*boardDirsPtr, ",")
			for _, dir := range boardDirs {
				if dir == "" || dir == "*" {
					// treated as "all boards", but handle this in the loop just in case there's something like "a,*,b"
					// if the only value in the string is *, there will be no single board associated with the filter
					continue
				}
				err = db.QueryRowContextSQL(ctx, tx, `SELECT id FROM DBPREFIXboards WHERE dir = ?`, []any{dir}, []any{&boardID})
				if errors.Is(err, sql.ErrNoRows) {
					// board may have been deleted, skip it and don't return an error
					continue
				} else if err != nil {
					return err
				}

				if _, err = db.ExecContextSQL(ctx, tx,
					`INSERT INTO DBPREFIXfilter_boards(filter_id,board_id) VALUES(?,?)`, filterID, boardID,
				); err != nil {
					return err
				}
			}
		}
		if isRegex {
			matchMode = gcsql.RegexMatch
		} else {
			matchMode = gcsql.SubstrMatch
		}

		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id, match_mode, search, field) VALUES(?,?,?,'body')`,
			filterID, matchMode, search,
		); err != nil {
			return err
		}
	}
	return rows.Close()
}
