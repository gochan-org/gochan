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

type migrationStaff struct {
	gcsql.Staff
	boards string
	oldID  int
}

func (m *Pre2021Migrator) getMigrationUser(errEv *zerolog.Event) (*gcsql.Staff, error) {
	if m.migrationUser != nil {
		return m.migrationUser, nil
	}

	user := &gcsql.Staff{
		Username: "pre2021-migration" + gcutil.RandomString(8),
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
	m.staff = append(m.staff, migrationStaff{Staff: *user})
	return user, nil
}

func (m *Pre2021Migrator) MigrateStaff() error {
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
		var staff migrationStaff
		if err = rows.Scan(&staff.oldID, &staff.Username, &staff.Rank, &staff.boards, &staff.AddedOn, &staff.LastLogin); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan staff row")
			return err
		}
		m.staff = append(m.staff, staff)
	}
	for _, staff := range m.staff {
		newStaff, err := gcsql.GetStaffByUsername(staff.Username, false)
		if err == nil {
			// found staff
			gcutil.LogInfo().Str("username", staff.Username).Int("rank", staff.Rank).Msg("Found matching staff account")
			staff.ID = newStaff.ID
		} else if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// staff doesn't exist, create it (with invalid checksum to be updated by the admin)
			if _, err := gcsql.ExecSQL(
				"INSERT INTO DBPREFIXstaff(username,password_checksum,global_rank,added_on,last_login,is_active) values(?,'',?,?,?,1)",
				staff.Username, staff.Rank, staff.AddedOn, staff.LastLogin,
			); err != nil {
				errEv.Err(err).Caller().Str("username", staff.Username).Int("rank", staff.Rank).Msg("Failed to migrate staff account")
				return err
			}
			if staff.ID, err = gcsql.GetStaffID(staff.Username); err != nil {
				errEv.Err(err).Caller().Str("username", staff.Username).Msg("Failed to get staff account ID")
				return err
			}
			gcutil.LogInfo().Str("username", staff.Username).Int("rank", staff.Rank).Msg("Successfully migrated staff account")
		} else {
			errEv.Err(err).Caller().Str("username", staff.Username).Msg("Failed to get staff account info")
			return err
		}

		if staff.boards != "" && staff.boards != "*" {
			boardsArr := strings.Split(staff.boards, ",")
			for _, board := range boardsArr {
				board = strings.TrimSpace(board)
				boardID, err := gcsql.GetBoardIDFromDir(board)
				if err != nil {
					errEv.Err(err).Caller().
						Str("username", staff.Username).
						Str("board", board).
						Msg("Failed to get board ID")
					return err
				}
				if _, err = gcsql.ExecSQL("INSERT INTO DBPREFIXboard_staff(board_id,staff_id) VALUES(?,?)", boardID, staff.ID); err != nil {
					errEv.Err(err).Caller().
						Str("username", staff.Username).
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
