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
	errEv := common.LogError()
	defer errEv.Discard()
	// populate m.sections with all sections from the new db
	currentAllSections, err := gcsql.GetAllSections(false)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get all sections from new db")
		return err
	}

	for _, section := range currentAllSections {
		m.sections = append(m.sections, migrationSection{
			oldID:   -1,
			Section: section,
		})
	}

	rows, err := m.db.QuerySQL(sectionsQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to query old database sections")
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var section gcsql.Section
		if err = rows.Scan(&section.ID, &section.Position, &section.Hidden, &section.Name, &section.Abbreviation); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan row into section")
			return err
		}
		var found bool
		for s, newSection := range m.sections {
			if section.Name == newSection.Name {
				m.sections[s].oldID = section.ID

				found = true
				break
			}
		}
		if !found {
			migratedSection, err := gcsql.NewSection(section.Name, section.Abbreviation, section.Hidden, section.Position)
			if err != nil {
				errEv.Err(err).Caller().Str("sectionName", section.Name).Msg("Failed to migrate section")
				return err
			}
			m.sections = append(m.sections, migrationSection{
				Section: *migratedSection,
			})
		}
	}
	if err = rows.Close(); err != nil {
		errEv.Caller().Msg("Failed to close section rows")
		return err
	}
	return nil
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
		// error already logged
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
		if !found {
			for _, section := range m.sections {
				if section.oldID == board.oldSectionID {
					board.SectionID = section.ID
					break
				}
			}
		}

		m.boards[board.Dir] = board
		if found {
			// TODO: update board title, subtitle, section etc. in new db
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
	if err = gcsql.ResetBoardSectionArrays(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to reset board and section arrays")
		return err
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
