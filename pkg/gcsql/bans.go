package gcsql

import (
	"database/sql"
	"errors"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	ipBanQueryBase = `SELECT
	id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
	is_active, RANGE_START_NTOA, RANGE_END_NTOA, issued_at, appeal_at, expires_at,
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

func NewIPBanTx(tx *sql.Tx, ban *IPBan) error {
	const query = `INSERT INTO DBPREFIXip_ban
	(staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
		is_active, range_start, range_end, appeal_at, expires_at,
		permanent, staff_note, message, can_appeal)
	VALUES(?, ?, ?, ?, ?, ?, PARAM_ATON, PARAM_ATON, ?, ?, ?, ?, ?, ?)`
	if ban.ID > 0 {
		return ErrBanAlreadyInserted
	}
	_, err := ExecTxSQL(tx, query, ban.StaffID, ban.BoardID, ban.BannedForPostID, ban.CopyPostText,
		ban.IsThreadBan, ban.IsActive, ban.RangeStart, ban.RangeEnd, ban.AppealAt,
		ban.ExpiresAt, ban.Permanent, ban.StaffNote, ban.Message, ban.CanAppeal)
	if err != nil {
		return err
	}

	ban.ID, err = getLatestID("DBPREFIXip_ban", tx)
	return err
}

func NewIPBan(ban *IPBan) error {
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = NewIPBanTx(tx, ban); err != nil {
		return err
	}

	return tx.Commit()
}

// CheckIPBan returns the latest active IP ban for the given IP, as well as any
// errors. If the IPBan pointer is nil, the IP has no active bans. Because
// SQLite 3 does not support a native IP type, range bans are not supported if
// DBtype == "sqlite3"
func CheckIPBan(ip string, boardID int) (*IPBan, error) {
	query := ipBanQueryBase + " WHERE "
	if config.GetSystemCriticalConfig().DBtype == "sqlite3" {
		query += "range_start = ? OR range_end = ?"
	} else {
		query += "range_start <= PARAM_ATON AND PARAM_ATON <= range_end"
	}
	query += ` AND (board_id IS NULL OR board_id = ?) AND is_active AND
		(expires_at > CURRENT_TIMESTAMP OR permanent)
	ORDER BY id DESC LIMIT 1`
	var ban IPBan
	err := QueryRowSQL(query, []any{ip, ip, boardID}, []any{
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

func GetIPBanByID(id int) (*IPBan, error) {
	const query = ipBanQueryBase + " WHERE id = ?"
	var ban IPBan
	err := QueryRowSQL(query, []any{id}, []any{
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
		rows, err = QuerySQL(query, boardID)
	} else {
		rows, err = QuerySQL(query)
	}
	if err != nil {
		return nil, err
	}
	var bans []IPBan
	for rows.Next() {
		var ban IPBan
		if err = rows.Scan(
			&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText, &ban.IsThreadBan,
			&ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt,
			&ban.Permanent, &ban.StaffNote, &ban.Message, &ban.CanAppeal,
		); err != nil {
			rows.Close()
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
	const query = `INSERT INTO DBPREFIXip_ban_appeals (ip_ban_id, appeal_text, is_denied) VALUES(?, ?, FALSE)`
	_, err := ExecSQL(query, ipb.ID, msg)
	return err
}

// IsGlobalBan returns true if BoardID is a nil int, meaning they are banned on all boards, as opposed to a specific one
func (ipb IPBan) IsGlobalBan() bool {
	return ipb.BoardID == nil
}

func (ban IPBan) BannedForever() bool {
	return ban.IsActive && ban.Permanent && !ban.CanAppeal
}

func (ipb *IPBan) Deactivate(_ int) error {
	const deactivateQuery = `UPDATE DBPREFIXip_ban SET is_active = FALSE WHERE id = ?`
	const auditInsertQuery = `INSERT INTO DBPREFIXip_ban_audit
		(ip_ban_id, staff_id, is_active, is_thread_ban, expires_at, appeal_at, permanent, staff_note, message, can_appeal)
		SELECT
		id, staff_id, is_active, is_thread_ban, expires_at, appeal_at, permanent, staff_note, message, can_appeal
		FROM DBPREFIXip_ban WHERE id = ?`
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = ExecTxSQL(tx, deactivateQuery, ipb.ID); err != nil {
		return err
	}
	if _, err = ExecTxSQL(tx, auditInsertQuery, ipb.ID); err != nil {
		return err
	}
	return tx.Commit()
}
