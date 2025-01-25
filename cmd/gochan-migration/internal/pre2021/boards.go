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
	oldID        int
	gcsql.Board
}

type migrationSection struct {
	oldID int
	gcsql.Section
}

func (m *Pre2021Migrator) migrateSectionsInPlace() error {
	_, err := m.db.ExecSQL(`ALTER TABLE DBPREFIXsections RENAME COLUMN list_order TO position`)
	if err != nil {
		common.LogError().Caller().Msg("Failed to rename list_order column to position")
	}
	return err
}

func (m *Pre2021Migrator) migrateBoardsInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()
	err := m.migrateSectionsInPlace()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to migrate sections")
		return err
	}

	for _, statement := range alterStatements {
		if strings.Contains(statement, "CONSTRAINT") && m.db.SQLDriver() == "sqlite3" {
			// skip constraints in SQLite since they can't be added after table creation
			continue
		}
		_, err = m.db.ExecSQL(statement)
		if err != nil {
			errEv.Err(err).Caller().
				Str("statement", statement).
				Msg("Failed to execute alter statement")
			return err
		}
	}

	return nil
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
				// section already exists, update values
				m.sections[s].oldID = section.ID
				m.sections[s].Abbreviation = section.Abbreviation
				m.sections[s].Hidden = section.Hidden
				m.sections[s].Position = section.Position
				common.LogInfo().
					Int("sectionID", section.ID).
					Int("oldSectionID", m.sections[s].oldID).
					Str("sectionName", section.Name).
					Str("sectionAbbreviation", section.Abbreviation).
					Msg("Section already exists in new db, updating values")
				if err = m.sections[s].UpdateValues(); err != nil {
					errEv.Err(err).Caller().Str("sectionName", section.Name).Msg("Failed to update pre-existing section values")
				}
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
	m.boards = nil
	errEv := common.LogError()
	defer errEv.Discard()

	// get all boards from new db
	err := gcsql.ResetBoardSectionArrays()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to reset board section arrays")
		return nil
	}

	if err = m.migrateSectionsToNewDB(); err != nil {
		// error should already be logged by migrateSectionsToNewDB
		return err
	}

	allBoards, err := gcsql.GetAllBoards(false)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get all boards from new db")
		return err
	}
	for _, board := range allBoards {
		m.boards = append(m.boards, migrationBoard{
			oldSectionID: -1,
			oldID:        -1,
			Board:        board,
		})
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
			&board.oldID, &board.NavbarPosition, &board.Dir, &board.Title, &board.Subtitle, &board.Description,
			&board.SectionID, &board.MaxFilesize, &maxPages, &board.DefaultStyle, &board.Locked, &board.CreatedAt,
			&board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength,
			&board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan row into board")
			return err
		}
		board.MaxThreads = maxPages * config.GetBoardConfig(board.Dir).ThreadsPerPage
		found := false

		for b, newBoard := range m.boards {
			if newBoard.Dir == board.Dir {
				m.boards[b].oldID = board.oldID
				m.boards[b].oldSectionID = board.SectionID
				common.LogInfo().
					Str("board", board.Dir).
					Int("oldBoardID", board.ID).
					Int("migratedBoardID", newBoard.ID).
					Msg("Board already exists in new db, updating values")
				// don't update other values in the array since they don't affect migrating threads or posts
				if _, err = gcsql.ExecSQL(`UPDATE DBPREFIXboards
					SET uri = ?, navbar_position = ?, title = ?, subtitle = ?, description = ?,
					max_file_size = ?, max_threads = ?, default_style = ?, locked = ?,
					anonymous_name = ?, force_anonymous = ?, autosage_after = ?, no_images_after = ?, max_message_length = ?,
					min_message_length = ?, allow_embeds = ?, redirect_to_thread = ?, require_file = ?, enable_catalog = ?
					WHERE id = ?`,
					board.Dir, board.NavbarPosition, board.Title, board.Subtitle, board.Description,
					board.MaxFilesize, board.MaxThreads, board.DefaultStyle, board.Locked,
					board.AnonymousName, board.ForceAnonymous, board.AutosageAfter, board.NoImagesAfter, board.MaxMessageLength,
					board.MinMessageLength, board.AllowEmbeds, board.RedirectToThread, board.RequireFile, board.EnableCatalog,
					newBoard.ID); err != nil {
					errEv.Err(err).Caller().Str("board", board.Dir).Msg("Failed to update board values")
					return err
				}
				found = true
				break
			}
		}

		if found {
			continue
		}

		// create new board using the board data from the old db
		// omitting things like ID and creation date since we don't really care
		if err = gcsql.CreateBoard(&board.Board, board.IsHidden(false)); err != nil {
			errEv.Err(err).Caller().Str("board", board.Dir).Msg("Failed to create board")
			return err
		}
		m.boards = append(m.boards, board)
		common.LogInfo().
			Str("dir", board.Dir).
			Int("boardID", board.ID).
			Msg("Board successfully created")
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
