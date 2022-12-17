package gcsql

import (
	"database/sql"
	"strconv"
)

// GetAppeals returns an array of appeals, optionally limiting them to a specific ban
func GetAppeals(banID int, limit int) ([]IPBanAppeal, error) {
	query := `SELECT id, staff_id, ip_ban_id, appeal_text, staff_response, is_denied FROM DBPREFIXip_ban_appeals`
	if banID > 0 {
		query += " WHERE ip_ban_id = ?"
	}
	if limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}
	var rows *sql.Rows
	var err error
	if banID > 0 {
		rows, err = QuerySQL(query, banID)
	} else {
		rows, err = QuerySQL(query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var appeals []IPBanAppeal
	for rows.Next() {
		var appeal IPBanAppeal
		var staffID *int
		var staffResponse *string
		err = rows.Scan(&appeal.ID, &staffID, &appeal.IPBanID, &appeal.AppealText, &staffResponse, &appeal.IsDenied)
		if err != nil {
			return nil, err
		}
		if staffID != nil {
			appeal.StaffID = *staffID
		}
		if staffResponse != nil {
			appeal.StaffResponse = *staffResponse
		}
		appeals = append(appeals, appeal)
	}
	return appeals, nil
}

// ApproveAppeal deactivates the ban that the appeal was submitted for
func ApproveAppeal(appealID int, staffID int) error {
	const deactivateQuery = `UPDATE DBPREFIXip_ban SET is_active = FALSE WHERE id = (
		SELECT ip_ban_id FROM DBPREFIXip_ban_appeals WHERE id = ?)`
	const deactivateAppealQuery = `INSERT INTO DBPREFIXip_ban_audit (
		ip_ban_id, timestamp, staff_id, is_active, is_thread_ban, permanent, staff_note, message, can_appeal)
		VALUES((SELECT ip_ban_id FROM DBPREFIXip_ban_appeals WHERE id = ?),
		CURRENT_TIMESTAMP, ?, FALSE, FALSE, FALSE, '', '', TRUE)`
	const deleteAppealQuery = `DELETE FROM DBPREFIXip_ban_appeals WHERE id = ?`
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := PrepareSQL(deactivateQuery, tx)
	if err != nil {
		return err
	}
	defer func() {
		stmt.Close()
	}()
	if _, err = stmt.Exec(appealID); err != nil {
		return err
	}
	if stmt, err = PrepareSQL(deactivateAppealQuery, tx); err != nil {
		return err
	}
	if _, err = stmt.Exec(appealID, staffID); err != nil {
		return err
	}
	if stmt, err = PrepareSQL(deleteAppealQuery, tx); err != nil {
		return err
	}
	if _, err = stmt.Exec(appealID); err != nil {
		return err
	}
	return tx.Commit()
}
