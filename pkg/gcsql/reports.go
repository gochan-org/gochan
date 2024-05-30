package gcsql

import (
	"context"
	"time"
)

// CreateReport inserts a new report into the database and returns a Report pointer and any
// errors encountered
func CreateReport(postID int, ip string, reason string) (*Report, error) {
	insertSQL := `INSERT INTO DBPREFIXreports (post_id, ip, reason, is_cleared) VALUES(?, PARAM_ATON, ?, FALSE)`
	currentTime := time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	result, err := ExecContextSQL(ctx, tx, insertSQL, postID, ip, reason)
	if err != nil {
		return nil, err
	}
	reportID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	insertSQL = `INSERT INTO DBPREFIXreports_audit (report_id, timestamp, is_cleared) VALUES(?, ?, FALSE)`
	if _, err = ExecContextSQL(ctx, tx, insertSQL, reportID, currentTime); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &Report{
		ID:               int(reportID),
		HandledByStaffID: -1,
		PostID:           postID,
		IP:               ip,
		Reason:           reason,
		IsCleared:        false,
	}, nil
}

// ClearReport dismisses the report with the given `id`. If `block` is true, future reports of the post will
// be ignored. It returns a boolean value representing whether or not any reports matched,
// as well as any errors encountered
func ClearReport(id int, staffID int, block bool) (bool, error) {
	sql := `UPDATE DBPREFIXreports SET is_cleared = ?, handled_by_staff_id = ? WHERE id = ?`
	isCleared := 1
	if block {
		isCleared = 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return false, err
	}

	result, err := ExecContextSQL(ctx, tx, sql, isCleared, staffID, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return affected > 0, err
	}
	sql = `UPDATE DBPREFIXreports_audit SET is_cleared = ?, handled_by_staff_id = ? WHERE report_id = ?`
	_, err = ExecContextSQL(ctx, tx, sql, isCleared, staffID, id)
	if err != nil {
		return affected > 0, err
	}

	return affected > 0, tx.Commit()
}

// CheckPostReports checks to see if the given post ID has already been reported, and if a report of the post has been
// dismissed with prejudice (so that more reports of that post can't be made)
func CheckPostReports(postID int, reason string) (bool, bool, error) {
	sql := `SELECT COUNT(*), MAX(is_cleared) FROM DBPREFIXreports
		WHERE post_id = ? AND (reason = ? OR is_cleared = 2)`
	var num int
	var isCleared interface{}
	err := QueryRowTimeoutSQL(nil, sql, []any{postID, reason}, []any{&num, &isCleared})
	isClearedInt, _ := isCleared.(int64)
	return num > 0, isClearedInt == 2, err
}

// GetReports returns a Report array and any errors encountered. If `includeCleared` is true,
// the array will include reports that have already been dismissed
func GetReports(includeCleared bool) ([]Report, error) {
	sql := `SELECT id,handled_by_staff_id,post_id,IP_NTOA,reason,is_cleared FROM DBPREFIXreports`
	if !includeCleared {
		sql += ` WHERE is_cleared = FALSE`
	}

	rows, cancel, err := QueryTimeoutSQL(nil, sql)
	if err != nil {
		return nil, err
	}
	defer func() {
		cancel()
		rows.Close()
	}()
	var reports []Report
	for rows.Next() {
		var report Report
		var staffID interface{}
		err = rows.Scan(&report.ID, &staffID, &report.PostID, &report.IP, &report.Reason, &report.IsCleared)
		if err != nil {
			return nil, err
		}

		staffID64, _ := (staffID.(int64))
		report.HandledByStaffID = int(staffID64)
		reports = append(reports, report)
	}
	return reports, rows.Close()
}
