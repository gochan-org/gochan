package pre2021

import (
	"errors"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func (*Pre2021Migrator) migrateAnnouncementsInPlace() error {
	return common.NewMigrationError("pre2021", "migrateAnnouncementsInPlace not implemented")
}

func (m *Pre2021Migrator) migrateAnnouncementsToNewDB() error {
	errEv := common.LogError()
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
