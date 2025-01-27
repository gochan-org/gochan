package pre2021

import (
	"errors"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func (m *Pre2021Migrator) migrateAnnouncementsInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()

	if _, err := gcsql.ExecSQL(announcementsAlterStatement); err != nil {
		errEv.Err(err).Caller().Msg("Failed to alter announcements table")
		return err
	}

	var staffIDs []int
	rows, err := m.db.QuerySQL("SELECT id FROM DBPREFIXstaff")
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get staff IDs")
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan staff ID")
			return err
		}
		staffIDs = append(staffIDs, id)
	}
	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close staff ID rows")
		return err
	}

	m.migrationUser, err = m.getMigrationUser(errEv)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get migration user")
		return err
	}

	rows, err = m.db.QuerySQL("SELECT poster FROM DBPREFIXannouncements")
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get announcements")
		return err
	}
	defer rows.Close()

	var announcementPosters []string
	for rows.Next() {
		var poster string
		if err = rows.Scan(&poster); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan announcement row")
			return err
		}
		announcementPosters = append(announcementPosters, poster)
	}
	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close announcement rows")
		return err
	}
	for _, poster := range announcementPosters {
		id, err := gcsql.GetStaffID(poster)
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// user doesn't exist, use migration user
			common.LogWarning().Str("staff", poster).Msg("Staff username not found in database")
			id = m.migrationUser.ID
		} else if err != nil {
			errEv.Err(err).Caller().Str("staff", poster).Msg("Failed to get staff ID")
			return err
		}

		if _, err = gcsql.ExecSQL("UPDATE DBPREFIXannouncements SET staff_id = ? WHERE poster = ?", id, poster); err != nil {
			errEv.Err(err).Caller().Str("staff", poster).Msg("Failed to update announcement poster")
			return err
		}
	}

	return nil
}

func (m *Pre2021Migrator) migrateAnnouncementsToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	rows, err := m.db.QuerySQL(announcementsQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get announcements")
		return err
	}
	defer rows.Close()

	if _, err = m.getMigrationUser(errEv); err != nil {
		return err
	}

	for rows.Next() {
		var id int
		var subject, message, staff string
		var timestamp time.Time
		if err = rows.Scan(&id, &subject, &message, &staff, &timestamp); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan announcement row")
			return err
		}
		staffID, err := gcsql.GetStaffID(staff)
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// user doesn't exist, use migration user
			common.LogWarning().Str("staff", staff).Msg("Staff username not found in database")
			message += "\n(originally by " + staff + ")"
		} else if err != nil {
			errEv.Err(err).Caller().Str("staff", staff).Msg("Failed to get staff ID")
			return err
		}
		if _, err = gcsql.ExecSQL(
			"INSERT INTO DBPREFIXannouncements(staff_id,subject,message,timestamp) values(?,?,?,?)",
			staffID, subject, message, timestamp,
		); err != nil {
			errEv.Err(err).Caller().Str("staff", staff).Msg("Failed to migrate announcement")
			return err
		}
	}

	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close announcement rows")
		return err
	}
	return nil
}

func (m *Pre2021Migrator) MigrateAnnouncements() error {
	if m.IsMigratingInPlace() {
		return m.migrateAnnouncementsInPlace()
	}
	return m.migrateAnnouncementsToNewDB()
}
