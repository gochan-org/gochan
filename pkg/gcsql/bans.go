package gcsql

import "database/sql"

// CheckIPBan returns the latest active IP ban for the given IP, as well as any errors. If the
// IPBan pointer is nil, the IP has no active bans
func CheckIPBan(ip string) (*IPBan, error) {
	const query = `SELECT 
	id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,
	is_active, ip, issued_at, appeal_at, expires_at, permanent, staff_note,
	message, can_appeal
	FROM DBPREFIXip_ban WHERE ip = ? AND is_active AND (expires_at > CURRENT_TIMESTAMP OR permanent)
	ORDER BY id DESC LIMIT 1`
	var ban IPBan
	err := QueryRowSQL(query, interfaceSlice(ip), interfaceSlice(
		&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText, &ban.IsThreadBan,
		&ban.IsActive, &ban.IP, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote,
		&ban.Message, &ban.CanAppeal))
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &ban, nil
}

// IsGlobalBan returns true if BoardID is a nil int, meaning they are banned on all boards, as opposed to a specific one
func (ipb *IPBan) IsGlobalBan() bool {
	return ipb.BoardID == nil
}
