package gcsql

import (
	"database/sql"
	"errors"
	"strconv"
)

var (
	ErrAppealDoesNotExist = errors.New("appeal does not exist or has already been processed")
	ErrBanNotActive       = errors.New("ban is not active")
)

// AppealsQueryOptions holds options for getting a list of appeals, including SQL request options
type AppealsQueryOptions struct {
	*RequestOptions
	// BanID limits results to appeals for a specific ban if greater than 0
	BanID int
	// Active is used to optionally limit results to only those with active/inactive bans
	Active BooleanFilter
	// Unexpired is used to optionally limit results to only those with unexpired/expired bans
	Unexpired BooleanFilter
	// OrderDescending specifies whether results should be in descending order
	OrderDescending bool
	// Limit specifies the maximum number of results to return if greater than 0, otherwise no limit is applied
	Limit int
}

// GetAppeals returns an array of appeals, optionally limiting them to a specific ban or ordering them in descending order
func GetAppeals(options ...AppealsQueryOptions) ([]Appeal, error) {
	var opts AppealsQueryOptions
	if len(options) > 0 {
		opts = options[0]
	} else {
		opts = AppealsQueryOptions{
			Active:          OnlyTrue,
			Unexpired:       OnlyTrue,
			OrderDescending: true,
		}
	}
	opts.RequestOptions = setupOptionsWithTimeout(opts.RequestOptions)

	query := `SELECT id, staff_id, staff_username, ip_ban_id, appeal_text, is_denied, is_ban_active, ban_expires_at, timestamp FROM DBPREFIXv_appeals`
	if opts.BanID > 0 {
		query += " WHERE ip_ban_id = ?"
	}
	query += opts.Active.whereClause("is_ban_active", false)

	switch opts.Unexpired {
	case OnlyTrue:
		query += " AND (ban_expires_at > CURRENT_TIMESTAMP OR permanent = TRUE)"
	case OnlyFalse:
		query += " AND (ban_expires_at <= CURRENT_TIMESTAMP AND permanent = FALSE)"
	}

	if opts.OrderDescending {
		query += " ORDER BY id DESC"
	} else {
		query += " ORDER BY id ASC"
	}
	if opts.Limit > 0 {
		query += " LIMIT " + strconv.Itoa(opts.Limit)
	}

	var rows *sql.Rows
	var err error
	if opts.BanID > 0 {
		rows, err = Query(opts.RequestOptions, query, opts.BanID)
	} else {
		rows, err = Query(opts.RequestOptions, query)
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
		if err = rows.Scan(
			&appeal.ID, &staffID, &staffUsername, &appeal.IPBanID, &appeal.AppealText, &appeal.IsDenied,
			&appeal.IsBanActive, &appeal.BanExpiresAt, &appeal.Timestamp,
		); err != nil {
			return nil, err
		}
		if staffID != nil {
			appeal.StaffID = *staffID
			appeal.StaffUsername = *staffUsername
		}
		appeals = append(appeals, appeal)
	}
	return appeals, nil
}

// ApproveAppeal deactivates the ban that the appeal was submitted for
func ApproveAppeal(appealID int, staffID int) error {
	const checkAppealSQL = "SELECT ip_ban_id, is_ban_active FROM DBPREFIXv_appeals WHERE id = ? AND is_denied = FALSE"
	const insertAppealAudit = `INSERT INTO DBPREFIXip_ban_appeals_audit (appeal_id, appeal_text, staff_id, is_denied)
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
