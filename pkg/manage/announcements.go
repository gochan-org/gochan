package manage

import (
	"errors"
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	errMissingAnnouncementMessage = errors.New("missing message field in announcement")
)

type announcementWithName struct {
	ID        uint      `json:"no"`
	Staff     string    `json:"name"`
	Subject   string    `json:"sub"`
	Message   string    `json:"com"`
	Timestamp time.Time `json:"-"`
}

func getAllAnnouncements() ([]announcementWithName, error) {
	querySQL := `SELECT id, staff, subject, message, timestamp from DBPREFIXannouncements NATURAL JOIN (
		SELECT id AS staff_id, username AS staff FROM DBPREFIXstaff) S1`
	rows, err := gcsql.QuerySQL(querySQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var announcements []announcementWithName
	for rows.Next() {
		var announcement announcementWithName
		if err = rows.Scan(&announcement.ID, &announcement.Staff, &announcement.Subject, &announcement.Message, &announcement.Timestamp); err != nil {
			rows.Close()
			return announcements, err
		}
		announcements = append(announcements, announcement)
	}
	return announcements, rows.Close()
}
