package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

var (
	ErrAppealDoesNotExist = errors.New("appeal does not exist or has already been processed")
	ErrBanNotActive       = errors.New("ban is not active")
)

// GetAppeals returns an array of appeals, optionally limiting them to a specific ban or ordering them in descending order
func GetAppeals(banID int, limit int, orderDesc ...bool) ([]Appeal, error) {
	query := `SELECT id, staff_id, staff_username, ip_ban_id, appeal_text, staff_response, is_denied, timestamp FROM DBPREFIXv_appeals`
	if banID > 0 {
		query += " WHERE ip_ban_id = ?"
	}
	if len(orderDesc) > 0 && orderDesc[0] {
		query += " ORDER BY id DESC"
	} else {
		query += " ORDER BY id ASC"
	}
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error
	if banID > 0 {
		rows, err = QueryContextSQL(ctx, nil, query, banID)
	} else {
		rows, err = QueryContextSQL(ctx, nil, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var appeals []Appeal
	for rows.Next() {
		var appeal Appeal
		var staffID *int
		var staffUsername *string
		var staffResponse *string
		if err = rows.Scan(
			&appeal.ID, &staffID, &staffUsername, &appeal.IPBanID, &appeal.AppealText, &staffResponse,
			&appeal.IsDenied, &appeal.Timestamp,
		); err != nil {
			return nil, err
		}
		if staffID != nil {
			appeal.StaffID = *staffID
			appeal.StaffUsername = *staffUsername
		}
		if staffResponse != nil {
			appeal.StaffResponse = *staffResponse
		}
		appeals = append(appeals, appeal)
	}
	return appeals, nil
}

// GetAppealCount returns the number of pending ban appeals
func GetAppealCount() (int, error) {
	query := `SELECT COUNT(*) FROM DBPREFIXip_ban_appeals WHERE is_denied = FALSE`
	var count int
	err := QueryRowTimeoutSQL(nil, query, []any{}, []any{&count})
	return count, err
}

// ApproveAppeal deactivates the ban that the appeal was submitted for
func ApproveAppeal(appealID int, staffID int) error {
	const checkAppealSQL = "SELECT ip_ban_id, is_ban_active FROM DBPREFIXv_appeals WHERE id = ? AND is_denied = FALSE"
	const insertAppealAudit = `INSERT INTO DBPREFIXip_ban_appeals_audit (appeal_id, appeal_text, staff_id, staff_response, is_denied)
		VALUES(?, (SELECT appeal_text FROM DBPREFIXip_ban_appeals WHERE id = ?), ?, 'Appeal approved, ban deactivated.', FALSE)`

	opts := setupOptionsWithTimeout()
	defer opts.Cancel()
	var err error
	opts.Tx, err = BeginContextTx(opts.Context)
	if err != nil {
		return err
	}
	defer opts.Tx.Rollback()
	var banID int
	var isBanActive bool

	err = QueryRow(opts, checkAppealSQL, []any{appealID}, []any{&banID, &isBanActive})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAppealDoesNotExist
		}
		return err
	}
	if !isBanActive {
		return ErrBanNotActive
	}

	if err = DeactivateBan(banID, staffID, opts); err != nil {
		return err
	}

	_, err = Exec(opts, insertAppealAudit, appealID, appealID, staffID)
	if err != nil {
		return err
	}

	return opts.Tx.Commit()
}

// view: DBPREFIXv_appeals
type Appeal struct {
	IPBanAppeal
	StaffUsername string
	Timestamp     time.Time
}
