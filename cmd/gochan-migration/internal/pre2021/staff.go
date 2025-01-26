package pre2021

import (
	"errors"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func (*Pre2021Migrator) migrateStaffInPlace() error {
	err := common.NewMigrationError("pre2021", "migrateStaff not yet implemented")
	common.LogError().Err(err).Caller().Msg("Failed to migrate staff")
	return err
}

func (m *Pre2021Migrator) getMigrationUser(errEv *zerolog.Event) (*gcsql.Staff, error) {
	if m.migrationUser != nil {
		return m.migrationUser, nil
	}

	user := &gcsql.Staff{
		Username: "pre2021-migration" + gcutil.RandomString(15),
		AddedOn:  time.Now(),
	}
	_, err := gcsql.ExecSQL("INSERT INTO DBPREFIXstaff(username,password_checksum,global_rank,is_active) values(?,'',0,0)", user.Username)
	if err != nil {
		errEv.Err(err).Caller().Str("username", user.Username).Msg("Failed to create migration user")
		return nil, err
	}

	if err = gcsql.QueryRowSQL("SELECT id FROM DBPREFIXstaff WHERE username = ?", []any{user.Username}, []any{&user.ID}); err != nil {
		errEv.Err(err).Caller().Str("username", user.Username).Msg("Failed to get migration user ID")
		return nil, err
	}
	m.migrationUser = user
	return user, nil
}

func (m *Pre2021Migrator) migrateStaffToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	_, err := m.getMigrationUser(errEv)
	if err != nil {
		return err
	}

	rows, err := m.db.QuerySQL(staffQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get ban rows")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var username string
		var rank int
		var boards string
		var addedOn, lastActive time.Time

		if err = rows.Scan(&username, &rank, &boards, &addedOn, &lastActive); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan staff row")
			return err
		}
		_, err = gcsql.GetStaffByUsername(username, false)
		if err == nil {
			// found staff
			gcutil.LogInfo().Str("username", username).Int("rank", rank).Msg("Found matching staff account")
		}
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// staff doesn't exist, create it (with invalid checksum to be updated by the admin)
			if _, err2 := gcsql.ExecSQL(
				"INSERT INTO DBPREFIXstaff(username,password_checksum,global_rank,added_on,last_login,is_active) values(?,'',?,?,?,1)",
				username, rank, addedOn, lastActive,
			); err2 != nil {
				errEv.Err(err2).Caller().
					Str("username", username).Int("rank", rank).
					Msg("Failed to migrate staff account")
				return err
			}
			gcutil.LogInfo().Str("username", username).Int("rank", rank).Msg("Successfully migrated staff account")
		} else if err != nil {
			errEv.Err(err).Caller().Str("username", username).Msg("Failed to get staff account info")
			return err
		}
		staffID, err := gcsql.GetStaffID(username)
		if err != nil {
			errEv.Err(err).Caller().Str("username", username).Msg("Failed to get staff account ID")
			return err
		}
		if boards != "" && boards != "*" {
			boardsArr := strings.Split(boards, ",")
			for _, board := range boardsArr {
				board = strings.TrimSpace(board)
				boardID, err := gcsql.GetBoardIDFromDir(board)
				if err != nil {
					errEv.Err(err).Caller().
						Str("username", username).
						Str("board", board).
						Msg("Failed to get board ID")
					return err
				}
				if _, err = gcsql.ExecSQL("INSERT INTO DBPREFIXboard_staff(board_id,staff_id) VALUES(?,?)", boardID, staffID); err != nil {
					errEv.Err(err).Caller().
						Str("username", username).
						Str("board", board).
						Msg("Failed to apply staff board info")
					return err
				}
			}
		}
	}

	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close staff rows")
		return err
	}
	return nil
}

func (m *Pre2021Migrator) MigrateStaff() error {
	if m.IsMigratingInPlace() {
		return m.migrateStaffInPlace()
	}
	return m.migrateStaffToNewDB()
}
