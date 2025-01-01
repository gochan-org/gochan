package pre2021

import (
	"errors"
	"runtime/debug"
	"strings"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

type migrationBoard struct {
	oldID int
	gcsql.Board
}

func (m *Pre2021Migrator) migrateBoardsInPlace() error {
	return nil
}

func (m *Pre2021Migrator) createSectionIfNotExist(sectionCheck *gcsql.Section) (int, error) {
	// to be used when not migrating in place, otherwise the section table should be altered
	section, err := gcsql.GetSectionFromName(sectionCheck.Name)
	if errors.Is(err, gcsql.ErrSectionDoesNotExist) {
		// section doesn't exist, create it
		section, err = gcsql.NewSection(sectionCheck.Name, sectionCheck.Abbreviation, true, 0)
		if err != nil {
			return 0, err
		}
	}
	return section.ID, nil
}

func (m *Pre2021Migrator) migrateSectionsToNewDB() error {
	// creates sections in the new db if they don't exist, and also creates a migration section that
	// boards will be set to, to be moved to the correct section by the admin after migration
	rows, err := m.db.QuerySQL(sectionsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var section gcsql.Section
		if err = rows.Scan(
			&section.ID,
			&section.Position,
			&section.Hidden,
			&section.Name,
			&section.Abbreviation,
		); err != nil {
			return err
		}
		if _, err = m.createSectionIfNotExist(&section); err != nil {
			return err
		}
	}
	if err = rows.Close(); err != nil {
		return err
	}
	m.migrationSectionID, err = m.createSectionIfNotExist(&gcsql.Section{
		Name:         "Migrated Boards",
		Abbreviation: "mb",
		Hidden:       true,
	})

	return err
}

func (m *Pre2021Migrator) migrateBoardsToNewDB() error {
	if m.boards == nil {
		m.boards = make(map[string]migrationBoard)
	}
	errEv := common.LogError()
	defer errEv.Discard()

	err := m.migrateSectionsToNewDB()
	if err != nil {
		errEv.Err(err).Msg("Failed to migrate sections")
	}

	// get all boards from new db
	if err = gcsql.ResetBoardSectionArrays(); err != nil {
		return nil
	}

	// get boards from old db
	rows, err := m.db.QuerySQL(boardsQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to query old database boards")
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var board migrationBoard
		var maxPages int
		if err = rows.Scan(
			&board.ID, &board.NavbarPosition, &board.Dir, &board.Title, &board.Subtitle,
			&board.Description, &board.SectionID, &board.MaxFilesize, &maxPages, &board.DefaultStyle, &board.Locked,
			&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
			&board.MaxMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan row into board")
			return err
		}
		board.MaxThreads = maxPages * config.GetBoardConfig(board.Dir).ThreadsPerPage
		found := false
		for _, newBoard := range gcsql.AllBoards {
			if _, ok := m.boards[board.Dir]; !ok {
				m.boards[board.Dir] = board
			}
			if newBoard.Dir == board.Dir {
				common.LogWarning().Str("board", board.Dir).Msg("Board already exists in new db, moving on")
				found = true
				break
			}
		}
		m.boards[board.Dir] = board
		if found {
			continue
		}
		// create new board using the board data from the old db
		// omitting things like ID and creation date since we don't really care
		if err = gcsql.CreateBoard(&board.Board, false); err != nil {
			errEv.Err(err).Caller().Str("board", board.Dir).Msg("Failed to create board")
			return err
		}
		common.LogInfo().Str("board", board.Dir).Msg("Board successfully created")
	}
	return nil
}

func (m *Pre2021Migrator) MigrateBoards() error {
	defer func() {
		if r := recover(); r != nil {
			stackTrace := debug.Stack()
			traceLines := strings.Split(string(stackTrace), "\n")
			zlArr := zerolog.Arr()
			for _, line := range traceLines {
				zlArr.Str(line)
			}
			common.LogFatal().Caller().
				Interface("recover", r).
				Array("stackTrace", zlArr).
				Msg("Recovered from panic in MigrateBoards")
		}
	}()
	if m.IsMigratingInPlace() {
		return m.migrateBoardsInPlace()
	}
	return m.migrateBoardsToNewDB()
}
