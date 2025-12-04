package gcsql

import (
	"database/sql"
	"errors"
	"strconv"
)

const (
	ipBanQueryBase = `SELECT
	id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
	is_active, INET6_NTOA(range_start), INET6_NTOA(range_end), issued_at, appeal_at, expires_at,
	permanent, staff_note, message, can_appeal
	FROM DBPREFIXip_ban`
)

var (
	ErrBanAlreadyInserted = errors.New("ban already submitted")
)

type Ban interface {
	IsGlobalBan() bool
	Deactivate(int) error
}

func NewIPBan(ban *IPBan, requestOpts ...*RequestOptions) error {
	const query = `INSERT INTO DBPREFIXip_ban
	(staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
		is_active, range_start, range_end, appeal_at, expires_at,
		permanent, staff_note, message, can_appeal)
	VALUES(?, ?, ?, ?, ?, ?, INET6_ATON(?), INET6_ATON(?), ?, ?, ?, ?, ?, ?)`
	opts := setupOptions(requestOpts...)
	shouldCommit := opts.Tx == nil
	var err error
	if shouldCommit {
		opts.Tx, err = BeginTx()
		if err != nil {
			return err
		}
		defer opts.Tx.Rollback()
	}

	if ban.ID > 0 {
		return ErrBanAlreadyInserted
	}
	if _, err = Exec(opts, query, ban.StaffID, ban.BoardID, ban.BannedForPostID, ban.CopyPostText,
		ban.IsThreadBan, ban.IsActive, ban.RangeStart, ban.RangeEnd, ban.AppealAt,
		ban.ExpiresAt, ban.Permanent, ban.StaffNote, ban.Message, ban.CanAppeal,
	); err != nil {
		return err
	}

	ban.ID, err = getLatestID(opts, "DBPREFIXip_ban")
	if err != nil {
		return err
	}
	if shouldCommit {
		return opts.Tx.Commit()
	}

	return nil
}

// CheckIPBan returns the latest active IP ban for the given IP, as well as any
// errors. If the IPBan pointer is nil, the IP has no active bans.
func CheckIPBan(ip string, boardID int) (*IPBan, error) {
	query := ipBanQueryBase + ` WHERE ip_cmp(?, range_start) >= 0 AND ip_cmp(?, range_end) <= 0
		AND (board_id IS NULL OR board_id = ?) AND is_active
		AND (expires_at > CURRENT_TIMESTAMP OR permanent)
		ORDER BY id DESC LIMIT 1`
	var ban IPBan
	err := QueryRow(nil, query, []any{ip, ip, boardID}, []any{
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText,
		&ban.IsThreadBan, &ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt,
		&ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message,
		&ban.CanAppeal})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &ban, nil
}

func GetIPBanByID(opts *RequestOptions, id int) (*IPBan, error) {
	const query = ipBanQueryBase + " WHERE id = ?"
	var ban IPBan

	if opts == nil {
		opts = setupOptionsWithTimeout(opts)
		defer opts.Cancel()
	}

	err := QueryRow(opts, query, []any{id}, []any{
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText,
		&ban.IsThreadBan, &ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt,
		&ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message,
		&ban.CanAppeal})
	if err != nil {
		return nil, err
	}
	return &ban, err
}

func GetIPBans(boardID int, limit int, onlyActive bool) ([]IPBan, error) {
	query := ipBanQueryBase
	if boardID > 0 {
		query += " WHERE board_id = ?"
	}
	query += " ORDER BY issued_at DESC LIMIT " + strconv.Itoa(limit)
	var rows *sql.Rows
	var err error
	if boardID > 0 {
		rows, err = Query(nil, query, boardID)
	} else {
		rows, err = Query(nil, query)
	}
	if err != nil {
		return nil, err
	}
	var bans []IPBan
	defer rows.Close()
	for rows.Next() {
		var ban IPBan
		if err = rows.Scan(
			&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText, &ban.IsThreadBan,
			&ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt,
			&ban.Permanent, &ban.StaffNote, &ban.Message, &ban.CanAppeal,
		); err != nil {
			return nil, err
		}
		if onlyActive && !ban.IsActive {
			continue
		}
		bans = append(bans, ban)
	}
	return bans, rows.Close()
}

func (ipb *IPBan) Appeal(msg string) error {
	const insertAppealSQL = "INSERT INTO DBPREFIXip_ban_appeals (ip_ban_id, appeal_text, is_denied) VALUES(?, ?, FALSE)"
	const getLatestIDSQL = "SELECT MAX(id) FROM DBPREFIXip_ban_appeals"
	const insertAppealAuditSQL = "INSERT INTO DBPREFIXip_ban_appeals_audit(appeal_id, appeal_text, is_denied) SELECT id, appeal_text, is_denied FROM DBPREFIXip_ban_appeals WHERE id = ?"
	_, err := Exec(nil, insertAppealSQL, ipb.ID, msg)
	if err != nil {
		return err
	}
	var appealID int
	err = QueryRow(nil, getLatestIDSQL, []any{}, []any{&appealID})
	if err != nil {
		return err
	}
	_, err = Exec(nil, insertAppealAuditSQL, appealID)

	return err
}

// IsGlobalBan returns true if BoardID is a nil int, meaning they are banned on all boards, as opposed to a specific one
func (ipb IPBan) IsGlobalBan() bool {
	return ipb.BoardID == nil
}

func (ban IPBan) BannedForever() bool {
	return ban.IsActive && ban.Permanent && !ban.CanAppeal
}

func DeactivateBan(banID int, staffID int, opts ...*RequestOptions) error {
	const updateBanSQL = "UPDATE DBPREFIXip_ban SET is_active = FALSE WHERE id = ?"
	const insertAuditSQL = `INSERT INTO DBPREFIXip_ban_audit
			(ip_ban_id, staff_id, is_active, is_thread_ban, expires_at, appeal_at, permanent, staff_note, message, can_appeal)
			VALUES(?, ?, FALSE, ?, ?, ?, ?, ?, ?, ?)`

	opt := setupOptionsWithTimeout(opts...)
	ban, err := GetIPBanByID(opt, banID)
	if err != nil {
		return err
	}
	if !ban.IsActive {
		return ErrBanNotActive
	}

	if _, err = Exec(opt, updateBanSQL, banID); err != nil {
		return err
	}

	if _, err = Exec(opt, insertAuditSQL, ban.ID, staffID, ban.IsThreadBan, ban.ExpiresAt, ban.AppealAt,
		ban.Permanent, ban.StaffNote, ban.Message, ban.CanAppeal); err != nil {
		return err
	}
	return nil

}
