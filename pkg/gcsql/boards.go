package gcsql

import (
	"database/sql"
	"net/http"
	"strconv"
)

// UpdateID takes a board struct and sets the database id according to the dir that is already set
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. (Just bad design in general, try to avoid directly mutating state like this)
func (board *Board) UpdateID() error {
	const query = `SELECT id FROM DBPREFIXboards WHERE dir = ?`
	return QueryRowSQL(query, interfaceSlice(board.Dir), interfaceSlice(&board.ID))
}

// ChangeFromRequest takes values from a HTTP request
func (board *Board) ChangeFromRequest(request *http.Request, dbUpdate bool) error {
	if request.FormValue("docreate") != "" {
		// prevent directory changes if the board already exists
		board.Dir = request.FormValue("dir")
	}
	board.Title = request.FormValue("title")
	board.Subtitle = request.FormValue("subtitle")
	board.Description = request.FormValue("description")
	board.Type, _ = strconv.Atoi(request.FormValue("boardtype"))
	board.UploadType, _ = strconv.Atoi(request.FormValue("uploadtype"))
	board.Section, _ = strconv.Atoi(request.FormValue("section"))
	board.MaxFilesize, _ = strconv.Atoi(request.FormValue("maxfilesize"))
	board.MaxPages, _ = strconv.Atoi(request.FormValue("maxpages"))
	board.DefaultStyle = request.FormValue("defaultstyle")
	board.Locked = len(request.Form["locked"]) > 0
	board.Anonymous = request.FormValue("anonname")
	board.ForcedAnon = len(request.Form["forcedanon"]) > 0
	board.MaxAge, _ = strconv.Atoi(request.FormValue("maxage"))
	board.AutosageAfter, _ = strconv.Atoi(request.FormValue("autosageafter"))
	board.NoImagesAfter, _ = strconv.Atoi(request.FormValue("nouploadsafter"))
	board.MaxMessageLength, _ = strconv.Atoi(request.FormValue("maxmessagelength"))
	board.EmbedsAllowed = len(request.Form["embedsallowed"]) > 0
	board.RedirectToThread = len(request.Form["redirecttothread"]) > 0
	board.ShowID = len(request.Form["showid"]) > 0
	board.RequireFile = len(request.Form["requirefile"]) > 0
	board.EnableCatalog = len(request.Form["enablecatalog"]) > 0
	board.EnableSpoileredImages = len(request.Form["enablefilespoilers"]) > 0
	board.EnableSpoileredThreads = len(request.Form["enablethreadspoilers"]) > 0
	board.Worksafe = len(request.Form["worksafe"]) > 0
	board.Cooldowns.NewThread, _ = strconv.Atoi(request.FormValue("threadcooldown"))
	board.Cooldowns.Reply, _ = strconv.Atoi(request.FormValue("replycooldown"))
	board.Cooldowns.ImageReply, _ = strconv.Atoi(request.FormValue("imagecooldown"))
	board.ThreadsPerPage, _ = strconv.Atoi(request.FormValue("threadsperpage"))
	if !dbUpdate {
		return nil
	}
	id, err := getBoardIDFromURI(board.Dir)
	if err != nil {
		return err
	}
	const query = `UPDATE DBPREFIXboards SET 
	section_id = ?,navbar_position = ?,
	title = ?,subtitle = ?,description = ?,max_file_size = ?,default_style = ?,
	locked = ?,anonymous_name = ?,force_anonymous = ?,autosage_after = ?,no_images_after = ?,
	max_message_length = ?,allow_embeds = ?,redirect_to_thread = ?,require_file = ?,
	enable_catalog = ? WHERE id = ?`

	_, err = ExecSQL(query,
		board.Section, board.ListOrder,
		board.Title, board.Subtitle, board.Description, board.MaxFilesize, board.DefaultStyle,
		board.Locked, board.Anonymous, board.ForcedAnon, board.AutosageAfter, board.NoImagesAfter,
		board.MaxMessageLength, board.EmbedsAllowed, board.RedirectToThread, board.RequireFile,
		board.EnableCatalog, id)
	return err
}

// PopulateData gets the board data from the database, according to its id, and sets the respective properties.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func (board *Board) PopulateData(id int) error {
	const sql = "SELECT id, section_id, dir, navbar_position, title, subtitle, description, max_file_size, default_style, locked, created_at, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog FROM DBPREFIXboards WHERE id = ?"
	return QueryRowSQL(sql, interfaceSlice(id), interfaceSlice(&board.ID, &board.Section, &board.Dir, &board.ListOrder, &board.Title, &board.Subtitle, &board.Description, &board.MaxFilesize, &board.DefaultStyle, &board.Locked, &board.CreatedOn, &board.Anonymous, &board.ForcedAnon, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog))
}

// Delete deletes the board from the database (if a row with the struct's ID exists) and
// returns any errors. It does not remove the board directory or its files
func (board *Board) Delete() error {
	exists := DoesBoardExistByID(board.ID)
	if !exists {
		return ErrBoardDoesNotExist
	}
	if board.ID == 0 {
		return ErrNilBoard
	}
	const delSql = `DELETE FROM DBPREFIXboards WHERE id = ?`
	_, err := ExecSQL(delSql, board.ID)
	return err
}

// WordFilters gets an array of wordfilters that should be applied to new posts on
// this board
func (board *Board) WordFilters() ([]WordFilter, error) {
	wfs, err := GetWordFilters()
	if err != nil {
		return wfs, err
	}
	var applicable []WordFilter
	for _, filter := range wfs {
		if filter.OnBoard(board.Dir) {
			applicable = append(applicable, filter)
		}
	}
	return applicable, nil
}

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
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBoards() ([]Board, error) {
	const sql = `SELECT id, section_id, dir, navbar_position, title, subtitle, description, max_file_size, default_style, locked, created_at, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog FROM DBPREFIXboards
	ORDER BY navbar_position ASC, dir ASC`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var boards []Board
	for rows.Next() {
		var board Board
		err = rows.Scan(&board.ID, &board.Section, &board.Dir, &board.ListOrder, &board.Title, &board.Subtitle, &board.Description, &board.MaxFilesize, &board.DefaultStyle, &board.Locked, &board.CreatedOn, &board.Anonymous, &board.ForcedAnon, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, nil
}

// GetBoardFromID returns the board corresponding to a given id
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetBoardFromID(boardID int) (Board, error) {
	var board Board
	err := board.PopulateData(boardID)
	return board, err
}

// GetBoardFromPostID gets the boardURI that a given postid exists on
func GetBoardFromPostID(postID int) (boardURI string, wasFound bool, err error) {
	const query = `SELECT board.uri FROM DBPREFIXboards as board
	JOIN (
		SELECT threads.board_id FROM DBPREFIXthreads as threads
		JOIN DBPREFIXposts as posts ON posts.thread_id = threads.id
		WHERE posts.id = ?
	) as threads ON threads.board_id = board.id`
	err = QueryRowSQL(query, interfaceSlice(postID), interfaceSlice(&boardURI))
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	return boardURI, true, err
}

func getBoardIDFromURI(URI string) (id int, err error) {
	const sql = `SELECT id FROM DBPREFIXboards WHERE uri = ?`
	err = QueryRowSQL(sql, interfaceSlice(URI), interfaceSlice(&id))
	return id, err
}

// CreateDefaultBoardIfNoneExist creates a default board if no boards exist yet
func CreateDefaultBoardIfNoneExist() error {
	const sqlStr = `SELECT COUNT(id) FROM DBPREFIXboards`
	var count int
	QueryRowSQL(sqlStr, interfaceSlice(), interfaceSlice(&count))
	if count > 0 {
		return nil
	}
	defaultSectionID, err := GetOrCreateDefaultSectionID()
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	board := Board{}
	board.SetDefaults("", "", "")
	board.Section = defaultSectionID
	if err = CreateBoard(&board); err != nil {
		return err
	}
	return nil
}

// CreateBoard creates this board in the database if it doesnt exist already, also sets ID to correct value
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CreateBoard(values *Board) error {
	exists := DoesBoardExistByDir(values.Dir)
	if exists {
		return ErrBoardExists
	}
	const maxThreads = 300
	const sqlINSERT = `INSERT INTO DBPREFIXboards (
		navbar_position, dir, uri, title, subtitle, description, max_file_size, max_threads, default_style, locked, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog, section_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	const sqlSELECT = "SELECT id FROM DBPREFIXboards WHERE dir = ?"
	//Excecuted in two steps this way because last row id functions arent thread safe, dir and uri is unique

	if values == nil {
		return ErrNilBoard
	}
	_, err := ExecSQL(sqlINSERT,
		values.ListOrder, values.Dir, values.Dir, values.Title, values.Subtitle,
		values.Description, values.MaxFilesize, maxThreads, values.DefaultStyle,
		values.Locked, values.Anonymous, values.ForcedAnon, values.AutosageAfter,
		values.NoImagesAfter, values.MaxMessageLength, 1, values.EmbedsAllowed,
		values.RedirectToThread, values.RequireFile, values.EnableCatalog, values.Section)
	if err != nil {
		return err
	}
	return QueryRowSQL(sqlSELECT, interfaceSlice(values.Dir), interfaceSlice(&values.ID))
}

// GetBoardUris gets a list of all existing board URIs
func GetBoardUris() (URIS []string, err error) {
	const sql = `SELECT uri FROM DBPREFIXboards`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
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
