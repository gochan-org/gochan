package gcsql

import (
	"database/sql"
	"regexp"
)

type Ban interface {
	IsGlobalBan() bool
}

// CheckIPBan returns the latest active IP ban for the given IP, as well as any errors. If the
// IPBan pointer is nil, the IP has no active bans
func CheckIPBan(ip string, boardID int) (*IPBan, error) {
	const query = `SELECT 
	id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
	is_active, ip, issued_at, appeal_at, expires_at, permanent, staff_note,
	message, can_appeal
		FROM DBPREFIXip_ban WHERE ip = ? AND (board_id IS NULL OR board_id = ?) AND
		is_active AND (expires_at > CURRENT_TIMESTAMP OR permanent)
	ORDER BY id DESC LIMIT 1`
	var ban IPBan
	err := QueryRowSQL(query, interfaceSlice(ip, boardID), interfaceSlice(
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText, &ban.IsThreadBan,
		&ban.IsActive, &ban.IP, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote,
		&ban.Message, &ban.CanAppeal))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ban, nil
}

// IsGlobalBan returns true if BoardID is a nil int, meaning they are banned on all boards, as opposed to a specific one
func (ipb IPBan) IsGlobalBan() bool {
	return ipb.BoardID == nil
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

func (ub filenameOrUsernameBanBase) IsGlobalBan() bool {
	return ub.BoardID == nil
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
