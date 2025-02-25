package pre2021

import (
	"errors"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type migrationAnnouncement struct {
	gcsql.Announcement
	oldPoster string
}

func (m *Pre2021Migrator) MigrateAnnouncements() error {
	errEv := common.LogError()
	defer errEv.Discard()

	rows, err := m.db.Query(nil, announcementsQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get announcements")
		return err
	}
	defer rows.Close()

	if _, err = m.getMigrationUser(errEv); err != nil {
		return err
	}

	var oldAnnouncements []migrationAnnouncement

	for rows.Next() {
		var announcement migrationAnnouncement

		if err = rows.Scan(&announcement.ID, &announcement.Subject, &announcement.Message, &announcement.oldPoster, &announcement.Timestamp); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan announcement row")
			return err
		}
		oldAnnouncements = append(oldAnnouncements, announcement)
	}
	for _, announcement := range oldAnnouncements {
		announcement.StaffID, err = gcsql.GetStaffID(announcement.oldPoster)
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// user doesn't exist, use migration user
			common.LogWarning().Str("staff", announcement.oldPoster).Msg("Staff username not found in database")
			announcement.Message += "\n(originally by " + announcement.oldPoster + ")"
			announcement.StaffID = m.migrationUser.ID
		} else if err != nil {
			errEv.Err(err).Caller().Str("staff", announcement.oldPoster).Msg("Failed to get staff ID")
			return err
		}
		if _, err = gcsql.Exec(nil,
			"INSERT INTO DBPREFIXannouncements(staff_id,subject,message,timestamp) values(?,?,?,?)",
			announcement.StaffID, announcement.Subject, announcement.Message, announcement.Timestamp,
		); err != nil {
			errEv.Err(err).Caller().Str("staff", announcement.oldPoster).Msg("Failed to migrate announcement")
			return err
		}
	}

	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close announcement rows")
		return err
	}
	return nil
}
