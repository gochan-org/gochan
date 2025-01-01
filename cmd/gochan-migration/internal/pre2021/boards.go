package pre2021

import (
	"runtime/debug"
	"strings"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

type migrationBoard struct {
	oldSectionID int
	gcsql.Board
}

type migrationSection struct {
	oldID int
	gcsql.Section
}

func (m *Pre2021Migrator) migrateSectionsInPlace() error {
	return common.NewMigrationError("pre2021", "migrateSectionsInPlace not implemented")
}

func (m *Pre2021Migrator) migrateBoardsInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()
	err := m.migrateSectionsInPlace()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to migrate sections")
		return err
	}
	err = common.NewMigrationError("pre2021", "migrateBoardsInPlace not implemented")
	errEv.Err(err).Caller().Msg("Failed to migrate boards")
	return err
}

func (m *Pre2021Migrator) migrateSectionsToNewDB() error {
	// creates sections in the new db if they don't exist, and also creates a migration section that
	// boards will be set to, to be moved to the correct section by the admin after migration
	rows, err := m.db.QuerySQL(sectionsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()
	errEv := common.LogError()
	defer errEv.Discard()
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
		m.sections = append(m.sections, migrationSection{
			oldID:   section.ID,
			Section: section,
		})

		for _, newSection := range gcsql.AllSections {
			if newSection.Name == section.Name || newSection.Abbreviation == section.Abbreviation {
				common.LogWarning().Str("section", section.Name).Msg("Section already exists in new db, moving on")
				m.sections[len(m.sections)-1].ID = newSection.ID
				break
			}
		}
		if _, err = gcsql.NewSection(section.Name, section.Abbreviation, false, section.Position); err != nil {
			errEv.Err(err).Caller().
				Str("sectionName", section.Name).
				Msg("Failed to create section")
			return err
		}
	}
	if err = rows.Close(); err != nil {
		errEv.Caller().Msg("Failed to close section rows")
		return err
	}
	return err
}

func (m *Pre2021Migrator) migrateBoardsToNewDB() error {
	if m.boards == nil {
		m.boards = make(map[string]migrationBoard)
	}
	errEv := common.LogError()
	defer errEv.Discard()

	// get all boards from new db
	err := gcsql.ResetBoardSectionArrays()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to reset board section arrays")
		return nil
	}

	if err = m.migrateSectionsToNewDB(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to migrate sections")
		return err
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
