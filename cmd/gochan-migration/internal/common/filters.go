package common

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

// Used for db version 4 upgrade to create the filter tables from the respective SQL init file
func AddFilterTables(db *gcsql.GCDB, ctx context.Context, tx *sql.Tx, sqlConfig *config.SQLConfig) error {
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

func MigrateFileBans(db *gcsql.GCDB, ctx context.Context, tx *sql.Tx, cfg *config.SQLConfig) error {
	rows, err := db.QueryContextSQL(ctx, nil, `SELECT board_id,staff_id,staff_note,issued_at,checksum,fingerprinter,ban_ip,ban_ip_message FROM DBPREFIXfile_ban`)
	if err != nil {
		return err
	}

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
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail) VALUES(?,?,?,?,?)`,
			fBanStaffID, fBanStaffNote, fBanIssuedAt, matchAction, detail); err != nil {
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
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,is_regex,search,field) VALUES(?,?,?,?)`, filterID, false, fBanChecksum, field,
		); err != nil {
			return err
		}
	}
	return nil
}

func MigrateFilenameBans(db *gcsql.GCDB, ctx context.Context, tx *sql.Tx, cfg *config.SQLConfig) error {
	rows, err := db.QueryContextSQL(ctx, nil, `SELECT board_id,staff_id,staff_note,issued_at,filename,is_regex FROM DBPREFIXfilename_ban`)
	if err != nil {
		return err
	}

	var fnBanBoardID *int
	var fnBanStaffID int
	var fnBanStaffNote string
	var fnBanIssuedAt time.Time
	var fnBanFilename string
	var fnBanIsRegex bool
	var filterID int
	for rows.Next() {
		if err = rows.Scan(
			&fnBanBoardID, &fnBanStaffID, &fnBanStaffNote, &fnBanIssuedAt, &fnBanFilename, &fnBanIsRegex,
		); err != nil {
			return err
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail) VALUES(?,?,?,?,?)`,
			fnBanStaffID, fnBanStaffNote, fnBanIssuedAt, "reject", "",
		); err != nil {
			return err
		}
		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,is_regex,search,field) VALUES(?,?,?,?)`,
			filterID, fnBanIsRegex, fnBanFilename, "filename",
		); err != nil {
			return err
		}
	}
	return nil
}

func MigrateUsernameBans(db *gcsql.GCDB, ctx context.Context, tx *sql.Tx, cfg *config.SQLConfig) error {
	rows, err := db.QueryContextSQL(ctx, nil, `SELECT board_id,staff_id,staff_note,issued_at,username,is_regex FROM DBPREFIXusername_ban`)
	if err != nil {
		return err
	}

	var unBanBoardID *int
	var unBanStaffID int
	var unBanStaffNote string
	var unBanIssuedAt time.Time
	var unBanUsername string
	var unBanIsRegex bool
	var filterID int
	for rows.Next() {
		if err = rows.Scan(
			&unBanBoardID, &unBanStaffID, &unBanStaffNote, &unBanIssuedAt, &unBanUsername, &unBanIsRegex,
		); err != nil {
			return err
		}

		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilters(staff_id, staff_note, issued_at, match_action, match_detail) VALUES(?,?,?,?,?)`,
			unBanStaffID, unBanStaffNote, unBanIssuedAt, "reject", "",
		); err != nil {
			return err
		}
		if err = db.QueryRowContextSQL(ctx, tx, `SELECT MAX(id) FROM DBPREFIXfilters`, nil, []any{&filterID}); err != nil {
			return err
		}
		if _, err = db.ExecContextSQL(ctx, tx,
			`INSERT INTO DBPREFIXfilter_conditions(filter_id,is_regex,search,field) VALUES(?,?,?,?)`,
			filterID, unBanIsRegex, unBanUsername, "name",
		); err != nil {
			return err
		}
	}

	return nil
}
