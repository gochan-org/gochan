package gcsql

import (
	"database/sql"
	"errors"
	"regexp"
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

func NewIPBan(ban *IPBan) error {
	const query = `INSERT INTO DBPREFIXip_ban
	(staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
		is_active, range_start, range_end, appeal_at, expires_at,
		permanent, staff_note, message, can_appeal)
	VALUES(?, ?, ?, ?, ?, ?, PARAM_ATON, PARAM_ATON, ?, ?, ?, ?, ?, ?)`
	if ban.ID > 0 {
		return ErrBanAlreadyInserted
	}
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return err
	}
	defer stmt.Close()
	if _, err = stmt.Exec(
		ban.StaffID, ban.BoardID, ban.BannedForPostID, ban.CopyPostText,
		ban.IsThreadBan, ban.IsActive, ban.RangeStart, ban.RangeEnd, ban.AppealAt,
		ban.ExpiresAt, ban.Permanent, ban.StaffNote, ban.Message, ban.CanAppeal,
	); err != nil {
		return err
	}
	ban.ID, err = getLatestID("DBPREFIXip_ban", tx)
	if err != nil {
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
	err := QueryRowSQL(query, interfaceSlice(ip, ip, boardID), interfaceSlice(
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText,
		&ban.IsThreadBan, &ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt,
		&ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message,
		&ban.CanAppeal))
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
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText,
		&ban.IsThreadBan, &ban.IsActive, &ban.RangeStart, &ban.RangeEnd, &ban.IssuedAt,
		&ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message,
		&ban.CanAppeal))
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

func checkUsernameOrFilename(usernameFilename string, check string, boardID int) (*filenameOrUsernameBanBase, error) {
	query := `SELECT
	id, board_id, staff_id, staff_note, issued_at, ` + usernameFilename + `, is_regex
	FROM DBPREFIX` + usernameFilename + `_ban WHERE (` + usernameFilename + ` = ? OR is_regex) AND (board_id IS NULL OR board_id = ?)`
	rows, err := QuerySQL(query, check, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ban filenameOrUsernameBanBase
		err = rows.Scan(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.check, &ban.IsRegex)
		if err == sql.ErrNoRows {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		if ban.IsRegex {
			match, err := regexp.MatchString(ban.check, check)
			if err != nil {
				return nil, err
			}
			if match {
				return &ban, nil
			}
		} else if ban.check == check {
			return &ban, nil
		}
	}
	return nil, nil
}

func CheckNameBan(name string, boardID int) (*UsernameBan, error) {
	banBase, err := checkUsernameOrFilename("username", name, boardID)
	if err != nil {
		return nil, err
	}
	if banBase == nil {
		return nil, nil
	}
	return &UsernameBan{
		Username:                  banBase.check,
		filenameOrUsernameBanBase: *banBase,
	}, nil
}

func NewNameBan(name string, isRegex bool, boardID int, staffID int, staffNote string) (*UsernameBan, error) {
	const query = `INSERT INTO DBPREFIXusername_ban
	(board_id, staff_id, staff_note, username, is_regex)
	VALUES(?,?,?,?,?)`
	var ban UsernameBan
	if boardID > 0 {
		ban.BoardID = new(int)
		*ban.BoardID = boardID
	}

	tx, err := BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	if _, err = stmt.Exec(ban.BoardID, staffID, staffNote, name, isRegex); err != nil {
		return nil, err
	}
	if ban.ID, err = getLatestID("DBPREFIXusername_ban", tx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	ban.StaffID = staffID
	ban.StaffNote = staffNote
	ban.Username = name
	ban.IsRegex = isRegex
	return &ban, nil
}

func GetNameBans(boardID int, limit int) ([]UsernameBan, error) {
	query := `SELECT
	id, board_id, staff_id, staff_note, issued_at, username, is_regex
	FROM DBPREFIXusername_ban`
	limitStr := ""
	if limit > 0 {
		limitStr = " LIMIT " + strconv.Itoa(limit)
	}
	var rows *sql.Rows
	var err error
	if boardID > 0 {
		query += " WHERE board_id = ?"
		rows, err = QuerySQL(query+limitStr, boardID)
	} else {
		rows, err = QuerySQL(query + limitStr)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bans []UsernameBan
	for rows.Next() {
		var ban UsernameBan
		if err = rows.Scan(
			&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Username,
			&ban.IsRegex,
		); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

func DeleteNameBan(id int) error {
	const query = `DELETE FROM DBPREFIXusername_ban WHERE id = ?`
	_, err := ExecSQL(query, id)
	return err
}

func GetFileBans(boardID int, limit int) ([]FileBan, error) {
	query := `SELECT id, board_id, staff_id, staff_note, issued_at, checksum, fingerprinter, ban_ip, ban_ip_message FROM DBPREFIXfile_ban`
	limitStr := ""
	if limit > 0 {
		limitStr = " LIMIT " + strconv.Itoa(limit)
	}
	var rows *sql.Rows
	var err error
	if boardID > 0 {
		query += " WHERE board_id = ?"
		rows, err = QuerySQL(query+limitStr, boardID)
	} else {
		rows, err = QuerySQL(query + limitStr)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bans []FileBan
	for rows.Next() {
		var ban FileBan
		if err = rows.Scan(
			&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt,
			&ban.Checksum, &ban.Fingerprinter, &ban.BanIP, &ban.BanIPMessage,
		); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

func GetFilenameBans(boardID int, limit int) ([]FilenameBan, error) {
	query := `SELECT id, board_id, staff_id, staff_note, issued_at, filename, is_regex FROM DBPREFIXfilename_ban`
	limitStr := ""
	if limit > 0 {
		limitStr = " LIMIT " + strconv.Itoa(limit)
	}
	var rows *sql.Rows
	var err error
	if boardID > 0 {
		query += " WHERE board_id = ?"
		rows, err = QuerySQL(query+limitStr, boardID)
	} else {
		rows, err = QuerySQL(query + limitStr)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bans []FilenameBan
	for rows.Next() {
		var ban FilenameBan
		if err = rows.Scan(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Filename, &ban.IsRegex); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

func NewFilenameBan(filename string, isRegex bool, boardID int, staffID int, staffNote string) (*FilenameBan, error) {
	const query = `INSERT INTO DBPREFIXfilename_ban (board_id, staff_id, staff_note, filename, is_regex) VALUES(?,?,?,?,?)`
	var ban FilenameBan
	if boardID > 0 {
		ban.BoardID = new(int)
		*ban.BoardID = boardID
	}

	tx, err := BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	if _, err = stmt.Exec(ban.BoardID, staffID, staffNote, filename, isRegex); err != nil {
		return nil, err
	}
	if ban.ID, err = getLatestID("DBPREFIXfilename_ban", tx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}

	ban.StaffID = staffID
	ban.StaffNote = staffNote
	ban.Filename = filename
	ban.IsRegex = isRegex
	return &ban, nil
}

func (ub filenameOrUsernameBanBase) IsGlobalBan() bool {
	return ub.BoardID == nil
}

func (fnb *FilenameBan) Deactivate(_ int) error {
	const deleteQuery = `DELETE FROM DBPREFIXfilename_ban WHERE id = ?`
	_, err := ExecSQL(deleteQuery, fnb.ID)
	return err
}

func (fnb *UsernameBan) Deactivate(_ int) error {
	const deleteQuery = `DELETE FROM DBPREFIXusername_ban WHERE id = ?`
	_, err := ExecSQL(deleteQuery, fnb.ID)
	return err
}

func CheckFilenameBan(filename string, boardID int) (*FilenameBan, error) {
	banBase, err := checkUsernameOrFilename("filename", filename, boardID)
	if err != nil {
		return nil, err
	}
	if banBase == nil {
		return nil, nil
	}
	return &FilenameBan{
		Filename:                  banBase.check,
		filenameOrUsernameBanBase: *banBase,
	}, nil
}

// CheckFileChecksumBan checks to see if the given checksum is banned on the given boardID, or on all boards.
// It returns the ban info (or nil if it is not banned) and any errors
func CheckFileChecksumBan(checksum string, boardID int) (*FileBan, error) {
	const query = `SELECT
	id, board_id, staff_id, staff_note, issued_at, checksum
	FROM DBPREFIXfile_ban
	WHERE checksum = ? AND (board_id IS NULL OR board_id = ?) ORDER BY id DESC LIMIT 1`
	var ban FileBan
	err := QueryRowSQL(query, interfaceSlice(checksum, boardID), interfaceSlice(
		&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum,
	))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &ban, err
}

func GetChecksumBans(boardID int, limit int) ([]FileBan, error) {
	query := `SELECT
	id, board_id, staff_id, staff_note, issued_at, checksum
	FROM DBPREFIXfile_ban`
	if boardID > 0 {
		query += " WHERE board_id = ?"
	}
	query += " LIMIT " + strconv.Itoa(limit)
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
	defer rows.Close()
	var bans []FileBan
	for rows.Next() {
		var ban FileBan
		if err = rows.Scan(
			&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum,
		); err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

func NewFileChecksumBan(checksum string, boardID int, staffID int, staffNote string) (*FileBan, error) {
	const query = `INSERT INTO DBPREFIXfile_ban
	(board_id, staff_id, staff_note, checksum) VALUES(?,?,?,?)`
	var ban FileBan
	var err error

	if boardID > 0 {
		ban.BoardID = new(int)
		*ban.BoardID = boardID
	}
	tx, err := BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	stmt, err := PrepareSQL(query, tx)
	if err != nil {
		return nil, err
	}
	if _, err = stmt.Exec(ban.BoardID, staffID, staffNote, checksum); err != nil {
		return nil, err
	}
	if ban.ID, err = getLatestID("DBPREFIXfile_ban", tx); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	ban.StaffID = staffID
	ban.StaffNote = staffNote
	ban.Checksum = checksum
	return &ban, nil
}

func (fb *FileBan) IsGlobalBan() bool {
	return fb.BoardID == nil
}

func (fb FileBan) Deactivate(_ int) error {
	const deleteQuery = `DELETE FROM DBPREFIXfile_ban WHERE id = ?`
	_, err := ExecSQL(deleteQuery, fb.ID)
	return err
}

// DeleteFileBanByID deletes the ban, given the id column value
func DeleteFileBanByID(id int) error {
	_, err := ExecSQL("DELETE FROM DBPREFIXfile_ban WHERE id = ?", id)
	return err
}
