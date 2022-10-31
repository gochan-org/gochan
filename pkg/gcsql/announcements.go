package gcsql

func GetAllAccouncements() ([]Announcement, error) {
	const query = `SELECT id, staff_id, subject, message, timestamp FROM DBPREFIXannouncements ORDER BY TIMESTAMP DESC`
	rows, err := QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var announcements []Announcement
	for rows.Next() {
		var announcement Announcement
		if err = rows.Scan(
			&announcement.ID, &announcement.StaffID, &announcement.Subject, &announcement.Message, &announcement.Timestamp,
		); err != nil {
			return announcements, err
		}
		announcements = append(announcements, announcement)
	}
	return announcements, nil
}
