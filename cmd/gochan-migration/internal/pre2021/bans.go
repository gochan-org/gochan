package pre2021

import (
	"context"
	"database/sql"
	"errors"
	"net"
	"os"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

type migrationBan struct {
	oldID        int
	allowRead    string
	ip           string
	name         string
	nameIsRegex  bool
	filename     string
	fileChecksum string
	boards       string
	staff        string
	timestamp    time.Time
	expires      time.Time
	permaban     bool
	reason       string
	banType      int
	staffNote    string
	appealAt     time.Time
	canAppeal    bool

	boardIDs []int
	staffID  int
	newID    int
}

type migrationAppeal struct {
	oldID         int
	oldBan        int
	message       string
	timestamp     time.Time
	denied        bool
	staffResponse sql.NullString

	newID  int
	newBan int
}

func (m *Pre2021Migrator) migrateBansInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()
	initSQLPath := gcutil.FindResource("sql/initdb_" + m.db.SQLDriver() + ".sql")
	ba, err := os.ReadFile(initSQLPath)
	if err != nil {
		errEv.Err(err).Caller().
			Str("initDBFile", initSQLPath).
			Msg("Failed to read initdb file")
		return err
	}
	statements := strings.Split(string(ba), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if strings.HasPrefix(stmt, "CREATE TABLE DBPREFIXip_ban") || strings.HasPrefix(stmt, "CREATE TABLE DBPREFIXfilter") {
			_, err = gcsql.Exec(nil, stmt)
			if err != nil {
				errEv.Err(err).Caller().
					Str("statement", stmt).
					Msg("Failed to create table")
				return err
			}
		}
	}
	// since the table names are different, migrateBansToNewDB can be called directly to migrate bans
	return m.migrateBansToNewDB()
}

func (*Pre2021Migrator) migrateBan(tx *sql.Tx, ban *migrationBan, boardID *int, errEv *zerolog.Event) (err error) {
	migratedBan := &gcsql.IPBan{
		BoardID:    boardID,
		RangeStart: ban.ip,
		RangeEnd:   ban.ip,
		IssuedAt:   ban.timestamp,
	}
	migratedBan.CanAppeal = ban.canAppeal
	migratedBan.AppealAt = ban.appealAt
	migratedBan.ExpiresAt = ban.expires
	migratedBan.Permanent = ban.permaban
	migratedBan.Message = ban.reason
	migratedBan.StaffID = ban.staffID
	migratedBan.StaffNote = ban.staffNote
	opts := &gcsql.RequestOptions{Tx: tx}

	if err = gcsql.NewIPBan(migratedBan, opts); err != nil {
		errEv.Err(err).Caller().
			Int("oldID", ban.oldID).Msg("Failed to migrate ban")
		return err
	}

	if err = gcsql.QueryRow(opts, "SELECT MAX(id) FROM DBPREFIXip_bans", nil, []any{&ban.newID}); err != nil {
		errEv.Err(err).Caller().
			Int("oldID", ban.oldID).Msg("Failed to get new ban ID after inserting ban")
		return err
	}

	return nil
}

func (m *Pre2021Migrator) migrateBansToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	tx, err := gcsql.BeginTx()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to start transaction")
		return err
	}
	defer tx.Rollback()

	rows, err := m.db.Query(nil, bansQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get bans")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var ban migrationBan
		if err = rows.Scan(
			&ban.oldID, &ban.allowRead, &ban.ip, &ban.name, &ban.nameIsRegex, &ban.filename, &ban.fileChecksum,
			&ban.boards, &ban.staff, &ban.timestamp, &ban.expires, &ban.permaban, &ban.reason, &ban.banType, &ban.staffNote, &ban.appealAt, &ban.canAppeal,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan ban row")
			return err
		}

		if ban.boards != "" && ban.boards != "*" {
			boardDirs := strings.Split(ban.boards, ",")
			for _, dir := range boardDirs {
				dir = strings.TrimSpace(dir)
				boardID, err := gcsql.GetBoardIDFromDir(dir)
				if err != nil {
					if errors.Is(err, gcsql.ErrBoardDoesNotExist) {
						common.Logger().Warn().Str("board", dir).Msg("Found unrecognized ban board")
						continue
					} else {
						errEv.Err(err).Caller().Str("board", dir).Msg("Failed getting board ID from dir")
						return err
					}
				}
				ban.boardIDs = append(ban.boardIDs, boardID)
			}
		}

		migrationUser, err := m.getMigrationUser(errEv)
		if err != nil {
			return err
		}
		ban.staffID, err = gcsql.GetStaffID(ban.staff)
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// username not found after staff were migrated, use a stand-in account to be updated by the admin later
			common.LogWarning().
				Str("username", ban.staff).
				Str("migrationUser", migrationUser.Username).
				Msg("Ban staff not found in migrated staff table, using migration user instead")
			ban.staffID = migrationUser.ID
		} else if err != nil {
			errEv.Err(err).Caller().Str("username", ban.staff).Msg("Failed to get staff from username")
			return err
		}

		if ban.ip == "" && ban.name == "" && ban.fileChecksum == "" && ban.filename == "" {
			common.LogWarning().Int("banID", ban.oldID).Msg("Found invalid ban (no IP, name, file checksum, or filename set)")
			continue
		}
		if ban.ip != "" {
			if net.ParseIP(ban.ip) == nil {
				gcutil.LogWarning().
					Int("oldID", ban.oldID).
					Str("ip", ban.ip).
					Msg("Found ban with invalid IP address, skipping")
				continue
			}
			if len(ban.boardIDs) == 0 {
				if err = m.migrateBan(tx, &ban, nil, errEv); err != nil {
					return err
				}
			} else {
				for b := range ban.boardIDs {
					if err = m.migrateBan(tx, &ban, &ban.boardIDs[b], errEv); err != nil {
						return err
					}
				}
			}
		}
		if ban.name != "" || ban.fileChecksum != "" || ban.filename != "" {
			filter := &gcsql.Filter{
				StaffID:     &ban.staffID,
				StaffNote:   ban.staffNote,
				IsActive:    true,
				HandleIfAny: true,
				MatchAction: "reject",
				MatchDetail: ban.reason,
			}
			var conditions []gcsql.FilterCondition
			if ban.name != "" {
				nameCondition := gcsql.FilterCondition{
					Field:     "name",
					Search:    ban.name,
					MatchMode: gcsql.ExactMatch,
				}
				if ban.nameIsRegex {
					nameCondition.MatchMode = gcsql.RegexMatch
				}
				conditions = append(conditions, nameCondition)
			}
			if ban.fileChecksum != "" {
				conditions = append(conditions, gcsql.FilterCondition{
					Field:     "checksum",
					MatchMode: gcsql.ExactMatch,
					Search:    ban.fileChecksum,
				})
			}
			if ban.filename != "" {
				filenameCondition := gcsql.FilterCondition{
					Field:     "filename",
					Search:    ban.filename,
					MatchMode: gcsql.ExactMatch,
				}
				if ban.nameIsRegex {
					filenameCondition.MatchMode = gcsql.RegexMatch
				}
				conditions = append(conditions, filenameCondition)
			}
			if err = gcsql.ApplyFilterTx(context.Background(), tx, filter, conditions, ban.boardIDs); err != nil {
				errEv.Err(err).Caller().Int("banID", ban.oldID).Msg("Failed to migrate ban to filter")
				return err
			}
		}
	}

	return tx.Commit()
}

func (m *Pre2021Migrator) migrateAppealsInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()

	err := gcutil.ErrNotImplemented
	errEv.Err(err).Caller().Msg("In-place ban appeal migration not implemented yet")
	return err
}

func (m *Pre2021Migrator) migrateAppealsToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	tx, err := gcsql.BeginContextTx(ctx)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to start appeals migration transaction")
		return err
	}
	opts := &gcsql.RequestOptions{Context: ctx, Tx: tx, Cancel: cancel}

	var appeals []migrationAppeal

	rows, err := m.db.Query(opts, `SELECT id, ban, message, timestamp, denied, staff_response FROM DBPREFIXip_ban_appeals`)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get ban appeals")
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var appeal migrationAppeal
		if err = rows.Scan(&appeal.oldID, &appeal.oldBan, &appeal.message, &appeal.timestamp, &appeal.denied, &appeal.staffResponse); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan ban appeal row")
			return err
		}
		appeals = append(appeals, appeal)
	}

	for a, appeal := range appeals {
		for _, ban := range m.bans {
			if ban.oldID == appeal.oldBan {
				appeals[a].newBan = ban.newID
				break
			}
		}

		if _, err = gcsql.Exec(opts,
			`INSERT INTO DBPREFIXip_ban_appeals (ip_ban_id, appeal_text, is_denied, timestamp) VALUES (?, ?, ?, ?)`,
			appeals[a].newBan, appeal.message, appeal.denied, appeal.timestamp,
		); err != nil {
			errEv.Err(err).Caller().
				Int("oldID", appeal.oldID).Msg("Failed to insert ban appeal")
			return err
		}

		if err = gcsql.QueryRow(opts, "SELECT MAX(id) FROM DBPREFIXip_ban_appeals", nil, []any{&appeals[a].newID}); err != nil {
			errEv.Err(err).Caller().
				Int("oldID", appeal.oldID).Msg("Failed to get new appeal ID after inserting appeal")
			return err
		}
	}

	return nil
}

func (m *Pre2021Migrator) MigrateBans() (err error) {
	if m.IsMigratingInPlace() {
		if err = m.migrateBansInPlace(); err != nil {
			return err
		}
		return m.migrateAppealsInPlace()
	}
	if err = m.migrateBansToNewDB(); err != nil {
		return err
	}

	return m.migrateAppealsToNewDB()
}
