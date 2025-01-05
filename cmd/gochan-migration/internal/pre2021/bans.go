package pre2021

import (
	"database/sql"
	"errors"
	"net"
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
	banID    int
	staffID  int
	filterID int
}

func (m *Pre2021Migrator) migrateBansInPlace() error {
	return common.NewMigrationError("pre2021", "migrateBansInPlace not yet implemented")
}

func (m *Pre2021Migrator) migrateBan(tx *sql.Tx, ban *migrationBan, boardID *int, errEv *zerolog.Event) error {
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
	if err := gcsql.NewIPBanTx(tx, migratedBan); err != nil {
		errEv.Err(err).Caller().
			Int("oldID", ban.oldID).Msg("Failed to migrate ban")
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

	rows, err := m.db.QuerySQL(bansQuery)
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

		ban.staffID, err = gcsql.GetStaffID(ban.staff)
		if errors.Is(err, gcsql.ErrUnrecognizedUsername) {
			// username not found after staff were migrated, use a stand-in account to be updated by the admin later
			migrationUser, err := m.getMigrationUser(errEv)
			if err != nil {
				return err
			}
			common.LogWarning().
				Str("username", ban.staff).
				Str("migrationUser", migrationUser.Username).
				Msg("Ban staff not found in migrated staff table, using migration user instead")
			ban.staffID = migrationUser.ID
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
	}

	return tx.Commit()
}

func (m *Pre2021Migrator) MigrateBans() error {
	if m.IsMigratingInPlace() {
		return m.migrateBansInPlace()
	}
	return m.migrateBansToNewDB()
}
