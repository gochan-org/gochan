package pre2021

import (
	"errors"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

type boardTable struct {
	id               int
	listOrder        int
	dir              string
	boardType        int
	uploadType       int
	title            string
	subtitle         string
	description      string
	section          int
	maxFileSize      int
	maxPages         int
	defaultStyle     string
	locked           bool
	createdOn        time.Time
	anonymous        string
	forcedAnon       bool
	maxAge           int
	autosageAfter    int
	noImagesAfter    int
	maxMessageLength int
	embedsAllowed    bool
	redirectToThread bool
	requireFile      bool
	enableCatalog    bool
}

func (m *Pre2021Migrator) migrateBoardsInPlace() error {
	return nil
}

func (m *Pre2021Migrator) createSectionIfNotExist(sectionCheck *gcsql.Section) (int, error) {
	// to be used when not migrating in place, otherwise the section table should be altered
	section, err := gcsql.GetSectionFromName(sectionCheck.Name)
	if errors.Is(err, gcsql.ErrSectionDoesNotExist) {
		// section doesn't exist, create it
		section, err = gcsql.NewSection(sectionCheck.Name, section.Abbreviation, true, 0)
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
	if m.oldBoards == nil {
		m.oldBoards = make(map[string]boardTable)
	}
	if m.newBoards == nil {
		m.newBoards = make(map[string]boardTable)
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
		var board boardTable
		if err = rows.Scan(
			&board.id, &board.listOrder, &board.dir, &board.boardType, &board.uploadType, &board.title, &board.subtitle,
			&board.description, &board.section, &board.maxFileSize, &board.maxPages, &board.defaultStyle, &board.locked,
			&board.createdOn, &board.anonymous, &board.forcedAnon, &board.maxAge, &board.autosageAfter, &board.noImagesAfter,
			&board.maxMessageLength, &board.embedsAllowed, &board.redirectToThread, &board.requireFile, &board.enableCatalog,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan row into board")
			return err
		}
		found := false
		for _, newBoard := range gcsql.AllBoards {
			if _, ok := m.oldBoards[board.dir]; !ok {
				m.oldBoards[board.dir] = board
			}

			if newBoard.Dir == board.dir {
				common.LogWarning().Str("board", board.dir).Msg("Board already exists in new db, moving on")
				found = true
				break
			}
		}
		if found {
			continue
		}
		// create new board using the board data from the old db
		// omitting things like ID and creation date since we don't really care
		if err = gcsql.CreateBoard(&gcsql.Board{
			Dir:              board.dir,
			Title:            board.title,
			Subtitle:         board.subtitle,
			Description:      board.description,
			SectionID:        board.section,
			MaxFilesize:      board.maxFileSize,
			DefaultStyle:     board.defaultStyle,
			Locked:           board.locked,
			AnonymousName:    board.anonymous,
			ForceAnonymous:   board.forcedAnon,
			AutosageAfter:    board.autosageAfter,
			NoImagesAfter:    board.noImagesAfter,
			MaxMessageLength: board.maxMessageLength,
			AllowEmbeds:      board.embedsAllowed,
			RedirectToThread: board.redirectToThread,
			RequireFile:      board.requireFile,
			EnableCatalog:    board.enableCatalog,
		}, false); err != nil {
			errEv.Err(err).Caller().Str("board", board.dir).Msg("Failed to create board")
			return err
		}
		m.newBoards[board.dir] = board
		common.LogInfo().Str("board", board.dir).Msg("Board successfully migrated")
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
