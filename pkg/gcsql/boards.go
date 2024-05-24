package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// selects all columns from DBPREFIXboards
	selectBoardsBaseSQL = `SELECT
	DBPREFIXboards.id, section_id, uri, dir, navbar_position, title, subtitle, description,
	max_file_size, max_threads, default_style, locked, created_at, anonymous_name, force_anonymous,
	autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread,
	require_file, enable_catalog
	FROM DBPREFIXboards
	INNER JOIN (
		SELECT id, hidden FROM DBPREFIXsections
	) s ON DBPREFIXboards.section_id = s.id `
)

var (
	// AllBoards provides a quick and simple way to access a list of all boards in non-hidden sections
	// without having to do any SQL queries. It and AllSections are updated by ResetBoardSectionArrays
	AllBoards []Board

	ErrNilBoard          = errors.New("board is nil")
	ErrBoardExists       = errors.New("board already exists")
	ErrBoardDoesNotExist = errors.New("board does not exist")
	ErrBoardIsLocked     = errors.New("board is locked")
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

// GetAllBoards gets a list of all existing boards
func GetAllBoards(onlyNonHidden bool) ([]Board, error) {
	query := selectBoardsBaseSQL
	if onlyNonHidden {
		query += " WHERE s.hidden = FALSE"
	}
	query += " ORDER BY navbar_position ASC, DBPREFIXboards.id ASC"
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	rows, err := QueryContextSQL(ctx, nil, query)
	if err != nil {
		return nil, err
	}
	var boards []Board
	for rows.Next() {
		var board Board
		if err = rows.Scan(
			&board.ID, &board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
			&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
			&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
			&board.MaxMessageLength, &board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile,
			&board.EnableCatalog,
		); err != nil {
			rows.Close()
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, rows.Close()
}

func GetBoardDir(id int) (string, error) {
	const query = `SELECT dir FROM DBPREFIXboards WHERE id = ?`
	var dir string
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(&dir))
	return dir, err
}

// GetBoardFromPostID gets the boardURI that a given postid exists on
func GetBoardDirFromPostID(postID int) (string, error) {
	const query = `SELECT board.uri FROM DBPREFIXboards as board
	JOIN (
		SELECT threads.board_id FROM DBPREFIXthreads as threads
		JOIN DBPREFIXposts as posts ON posts.thread_id = threads.id
		WHERE posts.id = ?
	) as threads ON threads.board_id = board.id`
	var boardURI string
	err := QueryRowSQL(query, interfaceSlice(postID), interfaceSlice(&boardURI))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrBoardDoesNotExist
	}
	return boardURI, err
}

func getBoardBase(where string, whereParameters []interface{}) (*Board, error) {
	query := selectBoardsBaseSQL + where
	board := new(Board)
	err := QueryRowSQL(query, whereParameters, interfaceSlice(
		&board.ID, &board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
		&board.MaxMessageLength, &board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile,
		&board.EnableCatalog,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrBoardDoesNotExist
	}
	return board, err
}

// GetBoardFromID returns the board corresponding to a given id
func GetBoardFromID(id int) (*Board, error) {
	return getBoardBase("WHERE DBPREFIXboards.id = ?", interfaceSlice(id))
}

// GetBoardFromDir returns the board corresponding to a given dir
func GetBoardFromDir(dir string) (*Board, error) {
	return getBoardBase("WHERE DBPREFIXboards.dir = ?", interfaceSlice(dir))
}

// GetIDFromDir returns the id of the board with the given dir value
func GetBoardIDFromDir(dir string) (id int, err error) {
	const query = `SELECT id FROM DBPREFIXboards WHERE dir = ?`
	err = QueryRowSQL(query, interfaceSlice(dir), interfaceSlice(&id))
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrBoardDoesNotExist
	}
	return id, err
}

// GetBoardURIs gets a list of all existing board URIs
func GetBoardURIs() (URIS []string, err error) {
	const sql = `SELECT uri FROM DBPREFIXboards`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uris []string
	for rows.Next() {
		var uri string
		if err = rows.Scan(&uri); err != nil {
			return nil, err
		}
		uris = append(uris, uri)
	}
	return uris, nil
}

// ResetBoardSectionArrays is run when the board list needs to be changed
// (board/section is added, deleted, etc)
func ResetBoardSectionArrays() error {
	allBoardsArr, err := GetAllBoards(true)
	if err != nil {
		return err
	}
	AllBoards = nil
	AllBoards = append(AllBoards, allBoardsArr...)
	for _, board := range AllBoards {
		if err = config.UpdateBoardConfig(board.Dir); err != nil {
			return fmt.Errorf("unable to update board config for /%s/: %s", board.Dir, err.Error())
		}
	}

	allSectionsArr, err := GetAllSections(true)
	if err != nil {
		return err
	}
	AllSections = nil
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
	(section_id, uri, dir, navbar_position, title, subtitle,
	description, max_file_size, max_threads, default_style, locked,
	anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length,
	min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog)
	VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`
	if board == nil {
		return ErrNilBoard
	}
	if DoesBoardExistByDir(board.Dir) {
		return ErrBoardExists
	}
	if board.Dir == "" {
		return errors.New("board dir string must not be empty")
	}
	if board.URI == "" {
		board.URI = board.Dir
	}
	if board.Title == "" {
		return errors.New("board title string must not be empty")
	}
	_, err := ExecSQL(sqlINSERT,
		&board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength,
		&board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog)
	if err != nil {
		return err
	}
	if err = QueryRowSQL(
		`SELECT id FROM DBPREFIXboards WHERE dir = ?`,
		interfaceSlice(board.Dir), interfaceSlice(&board.ID),
	); err != nil {
		return err
	}
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

func (board *Board) Delete() error {
	const query = `DELETE FROM DBPREFIXboards WHERE id = ?`
	_, err := ExecSQL(query, board.ID)
	if err != nil {
		return err
	}
	config.DeleteBoardConfig(board.Dir)
	return nil
}

// DeleteOldThreads deletes old threads that exceed the limit set by board.MaxThreads and returns the posts in those
// threads
func (board *Board) DeleteOldThreads() ([]int, error) {
	if board.MaxThreads < 1 {
		return nil, nil
	}
	tx, err := BeginTx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	rows, err := QueryTxSQL(tx, `SELECT id FROM DBPREFIXthreads WHERE board_id = ? AND is_deleted = FALSE AND stickied = FALSE ORDER BY last_bump DESC`,
		board.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threadIDs []interface{}
	var id int
	var threadsProccessed int
	for rows.Next() {
		threadsProccessed++
		if threadsProccessed <= board.MaxThreads {
			continue
		}
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		threadIDs = append(threadIDs, id)
	}
	if threadIDs == nil {
		// no threads to trim
		return nil, nil
	}
	idSetStr := createArrayPlaceholder(threadIDs)

	if _, err = ExecTxSQL(tx, `UPDATE DBPREFIXthreads SET is_deleted = TRUE WHERE id in `+idSetStr,
		threadIDs...); err != nil {
		return nil, err
	}

	if rows, err = QueryTxSQL(tx, `SELECT id FROM DBPREFIXposts WHERE thread_id in `+idSetStr,
		threadIDs...); err != nil {
		return nil, err
	}
	defer rows.Close()

	var postIDs []int
	for rows.Next() {
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		postIDs = append(postIDs, id)
	}

	if _, err = ExecTxSQL(tx, `UPDATE DBPREFIXposts SET is_deleted = TRUE WHERE thread_id in `+idSetStr,
		threadIDs...); err != nil {
		return nil, err
	}

	return postIDs, tx.Commit()
}

func (board *Board) GetThreads(onlyNotDeleted bool, orderLastByBump bool, stickiedFirst bool) ([]Thread, error) {
	query := selectThreadsBaseSQL + " WHERE board_id = ?"
	if onlyNotDeleted {
		query += " AND is_deleted = FALSE"
	}
	if orderLastByBump || stickiedFirst {
		query += " ORDER BY "
	}
	if stickiedFirst {
		query += "stickied DESC"
		if orderLastByBump {
			query += ", "
		}
	}
	if orderLastByBump {
		query += " last_bump DESC"
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

// ModifyInDB updates the board dataa in the database with new values
func (board *Board) ModifyInDB() error {
	const query = `UPDATE DBPREFIXboards SET
		section_id = ?,
		navbar_position = ?,
		title = ?,
		subtitle = ?,
		description = ?,
		max_file_size = ?,
		max_threads = ?,
		default_style = ?,
		locked = ?,
		anonymous_name = ?,
		force_anonymous = ?,
		autosage_after = ?,
		no_images_after = ?,
		max_message_length = ?,
		min_message_length = ?,
		allow_embeds = ?,
		redirect_to_thread = ?,
		require_file = ?,
		enable_catalog = ?
		WHERE id = ?`
	_, err := ExecSQL(query,
		board.SectionID, board.NavbarPosition, board.Title, board.Subtitle, board.Description,
		board.MaxFilesize, board.MaxThreads, board.DefaultStyle, board.Locked, board.AnonymousName,
		board.ForceAnonymous, board.AutosageAfter, board.NoImagesAfter, board.MaxMessageLength,
		board.MinMessageLength, board.AllowEmbeds, board.RedirectToThread, board.RequireFile, board.EnableCatalog,
		board.ID)
	if err != nil {
		return err
	}
	return ResetBoardSectionArrays()
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
