package gcsql

import (
	"errors"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// selects all columns from DBPREFIXboards
	selectBoardsBaseSQL = `SELECT
	id, section_id, dir, navbar_position, title, subtitle, description,
	max_file_size, default_style, locked, created_at, anonymous_name, force_anonymous,
	autosage_after, no_images_after, max_message_length, allow_embeds, redirect_to_thread,
	require_file, enable_catalog
	FROM DBPREFIXboards `
)

var (
	AllBoards            []Board
	ErrNilBoard          = errors.New("board is nil")
	ErrBoardExists       = errors.New("board already exists")
	ErrBoardDoesNotExist = errors.New("board does not exist")
)

// DoesBoardExistByID returns a bool indicating whether a board with a given id exists
func DoesBoardExistByID(ID int) bool {
	const query = `SELECT COUNT(id) FROM DBPREFIXboards WHERE id = ?`
	var count int
	QueryRowSQL(query, interfaceSlice(ID), interfaceSlice(&count))
	return count > 0
}

// DoesBoardExistByDir returns a bool indicating whether a board with a given directory exists
func DoesBoardExistByDir(dir string) bool {
	const query = `SELECT COUNT(dir) FROM DBPREFIXboards WHERE dir = ?`
	var count int
	QueryRowSQL(query, interfaceSlice(dir), interfaceSlice(&count))
	return count > 0
}

// getAllBoards gets a list of all existing boards
func getAllBoards() ([]Board, error) {
	const query = selectBoardsBaseSQL + "ORDER BY navbar_position ASC, id ASC"
	rows, err := QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boards []Board
	for rows.Next() {
		var board Board
		err = rows.Scan(
			&board.ID, &board.SectionID, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle, &board.Description,
			&board.MaxFilesize, &board.DefaultStyle, &board.Locked, &board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous,
			&board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength, &board.AllowEmbeds, &board.RedirectToThread,
			&board.RequireFile, &board.EnableCatalog)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, nil
}

func GetBoardDir(id int) (string, error) {
	const query = `SELECT dir FROM DBPREFIXboards WHERE id = ?`
	var dir string
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(&dir))
	return dir, err
}

// GetBoardFromID returns the board corresponding to a given id
func GetBoardFromID(id int) (*Board, error) {
	const query = selectBoardsBaseSQL + "WHERE id = ?"
	board := new(Board)
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(
		&board.ID, &board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
		&board.MaxMessageLength, &board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile,
		&board.EnableCatalog,
	))
	return board, err
}

// ResetBoardSectionArrays is run when the board list needs to be changed
// (board/section is added, deleted, etc)
func ResetBoardSectionArrays() error {
	AllBoards = nil
	AllSections = nil

	allBoardsArr, err := getAllBoards()
	if err != nil {
		return err
	}
	AllBoards = append(AllBoards, allBoardsArr...)

	allSectionsArr, err := getAllSections()
	if err != nil {
		return err
	}
	AllSections = append(AllSections, allSectionsArr...)
	return nil
}

// NewBoardSimple creates a new board in the database given the directory, title, subtitle, and description.
// Generic values are used for the other columns to be optionally changed later
func NewBoardSimple(dir string, title string, subtitle string, description string, appendToAllBoards bool) (*Board, error) {
	sectionID, err := getOrCreateDefaultSectionID()
	if err != nil {
		return nil, err
	}
	board := &Board{
		SectionID:        sectionID,
		URI:              dir,
		Dir:              dir,
		NavbarPosition:   3,
		Title:            title,
		Subtitle:         subtitle,
		Description:      description,
		MaxFilesize:      15000,
		MaxThreads:       300,
		DefaultStyle:     config.GetBoardConfig("").DefaultStyle,
		Locked:           false,
		AnonymousName:    "Anonymous",
		ForceAnonymous:   false,
		AutosageAfter:    500,
		NoImagesAfter:    -1,
		MaxMessageLength: 1500,
		MinMessageLength: 0,
		AllowEmbeds:      false,
		RedirectToThread: false,
		RequireFile:      false,
		EnableCatalog:    true,
	}
	// board.ShowID = false
	// board.EnableSpoileredImages = true
	// board.Worksafe = true
	// board.ThreadsPerPage = 20
	// board.Cooldowns = BoardCooldowns{
	// 	NewThread:  30,
	// 	Reply:      7,
	// 	ImageReply: 7,
	// }
	return board, CreateBoard(board, appendToAllBoards)
}

// CreateBoard inserts a new board into the database, using the fields from the given Board pointer.
// It sets board.ID and board.CreatedAt if it is successfull
func CreateBoard(board *Board, appendToAllBoards bool) error {
	const sqlINSERT = `INSERT INTO DBPREFIXboards
	(section_id, uri, dir, navbar_position, title, suttitle,
	description, max_file_size, max_threads, default_style, locked,
	anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length,
	min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog)
	VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	if board == nil {
		return ErrNilBoard
	}
	id, err := getNextFreeID("DBPREFIXboards")
	if err != nil {
		return err
	}
	_, err = ExecSQL(sqlINSERT,
		&board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength,
		&board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog)
	if err != nil {
		return err
	}
	board.ID = id
	board.CreatedAt = time.Now()
	if appendToAllBoards {
		AllBoards = append(AllBoards, *board)
	}
	return nil
}

// createDefaultBoardIfNoneExist creates a default board if no boards exist yet
func createDefaultBoardIfNoneExist() error {
	const query = `SELECT COUNT(id) FROM DBPREFIXboards`
	var count int
	QueryRowSQL(query, interfaceSlice(), interfaceSlice(&count))
	if count > 0 {
		return nil
	}

	// create a default generic /test/ board
	_, err := NewBoardSimple("test", "Testing Board", "Board for testing stuff", "Board for testing stuff", true)
	return err
}

func getBoardIDFromURI(uri string) (int, error) {
	const sql = `SELECT id FROM DBPREFIXboards WHERE uri = ?`
	var id int
	err := QueryRowSQL(sql, interfaceSlice(uri), interfaceSlice(&id))
	return id, err
}

func (board *Board) GetThreads(onlyNotDeleted bool) ([]Thread, error) {
	query := selectThreadsBaseSQL + " WHERE id = ?"
	if onlyNotDeleted {
		query += " AND is_deleted = FALSE"
	}
	rows, err := QuerySQL(query, board.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var threads []Thread
	for rows.Next() {
		var thread Thread
		err = rows.Scan(
			&thread.ID, &thread.BoardID, &thread.Locked, &thread.Stickied, &thread.Anchored,
			&thread.Cyclical, &thread.LastBump, &thread.DeletedAt, &thread.IsDeleted,
		)
		if err != nil {
			return threads, err
		}
		threads = append(threads, thread)
	}
	return threads, nil
}

// IsHidden returns true if the board is in a section that is hidden, otherwise false. If it is in a section
// that is not in the AllSections array, it returns defValueIfMissingSection
func (board *Board) IsHidden(defValueIfMissingSection bool) bool {
	for s := range AllSections {
		if AllSections[s].ID == board.SectionID {
			return AllSections[s].Hidden
		}
	}
	return defValueIfMissingSection // board is not in a valid section (or AllSections needs to be reset)
}

// AbsolutePath returns the full filepath of the board directory
func (board *Board) AbsolutePath(subpath ...string) string {
	return path.Join(config.GetSystemCriticalConfig().DocumentRoot, board.Dir, path.Join(subpath...))
}

// WebPath returns a string that represents the file's path as accessible by a browser
// fileType should be "boardPage", "threadPage", "upload", or "thumb"
func (board *Board) WebPath(fileName, fileType string) string {
	var filePath string
	systemCritical := config.GetSystemCriticalConfig()

	switch fileType {
	case "":
		fallthrough
	case "boardPage":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, fileName)
	case "threadPage":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "res", fileName)
	case "upload":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "src", fileName)
	case "thumb":
		fallthrough
	case "thumbnail":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "thumb", fileName)
	}
	return filePath
}
