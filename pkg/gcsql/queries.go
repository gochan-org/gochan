package gcsql

import (
	"database/sql"
	"errors"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// GetAllNondeletedMessageRaw gets all the raw message texts from the database, saved per id
func GetAllNondeletedMessageRaw() ([]MessagePostContainer, error) {
	const sql = `select posts.id, posts.message, posts.message_raw from DBPREFIXposts as posts
	WHERE posts.is_deleted = FALSE`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var messages []MessagePostContainer
	for rows.Next() {
		var message MessagePostContainer
		err = rows.Scan(message.ID, message.Message, message.MessageRaw)
		if err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

// SetFormattedInDatabase sets all the non-raw text for a given array of items.
func SetFormattedInDatabase(messages []MessagePostContainer) error {
	const sql = `UPDATE DBPREFIXposts
	SET message = ?
	WHERE id = ? ;
	`
	stmt, err := PrepareSQL(sql)
	defer stmt.Close()
	if err != nil {
		return err
	}
	for _, message := range messages {
		_, err = stmt.Exec(message.Message, message.ID)
		if err != nil {
			return err
		}
	}
	return err
}

// GetReplyCount gets the total amount non-deleted of replies in a thread
func GetReplyCount(postID int) (int, error) {
	const sql = `SELECT COUNT(posts.id) FROM DBPREFIXposts as posts
	JOIN (
		SELECT threads.id FROM DBPREFIXthreads as threads
		JOIN DBPREFIXposts as posts
		ON posts.thread_id = threads.id
		WHERE posts.id = ?
	) as thread
	ON posts.thread_id = thread.id
	WHERE posts.is_deleted = FALSE`
	var count int
	err := QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&count))
	return count, err
}

// GetReplyFileCount gets the amount of files non-deleted posted in total in a thread
func GetReplyFileCount(postID int) (int, error) {
	const sql = `SELECT COUNT(files.id) from DBPREFIXfiles as files
	JOIN (SELECT posts.id FROM DBPREFIXposts as posts
		JOIN (
			SELECT threads.id FROM DBPREFIXthreads as threads
			JOIN DBPREFIXposts as posts
			ON posts.thread_id = threads.id
			WHERE posts.id = ?
		) as thread
		ON posts.thread_id = thread.id
		WHERE posts.is_deleted = FALSE) as posts
	ON posts.id = files.post_id`
	var count int
	err := QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&count))
	return count, err
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (string, error) {
	const sql = `SELECT staff.username from DBPREFIXstaff as staff
	JOIN DBPREFIXsessions as sessions
	ON sessions.staff_id = staff.id
	WHERE sessions.data = ?`
	var username string
	err := QueryRowSQL(sql, InterfaceSlice(session), InterfaceSlice(&username))
	return username, err
}

// GetStaffBySession gets the staff that is logged in in the given session
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffBySession(session string) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	JOIN DBPREFIXsessions as sessions
	ON sessions.staff_id = staff.id
	WHERE sessions.data = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, InterfaceSlice(session), InterfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

// GetStaffByName gets the staff with a given name
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffByName(name string) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	WHERE staff.username = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, InterfaceSlice(name), InterfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

func getStaffByID(id int) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	WHERE staff.id = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, InterfaceSlice(id), InterfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

// NewStaff creates a new staff account from a given username, password and rank
func NewStaff(username string, password string, rank int) error {
	const sql = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank)
	VALUES (?, ?, ?);`
	_, err := ExecSQL(sql, username, gcutil.BcryptSum(password), rank)
	return err
}

// DeleteStaff deletes the staff with a given name.
// Implemented to change the account name to a random string and set it to inactive
func DeleteStaff(username string) error {
	const sql = `UPDATE DBPREFIXstaff SET username = ?, is_active = FALSE WHERE username = ?`
	_, err := ExecSQL(sql, gcutil.RandomString(45), username)
	return err
}

func getStaffID(username string) (int, error) {
	staff, err := GetStaffByName(username)
	if err != nil {
		return -1, err
	}
	return staff.ID, nil
}

// CreateSession inserts a session for a given key and username into the database
func CreateSession(key string, username string) error {
	const sql = `INSERT INTO DBPREFIXsessions (staff_id,data,expires) VALUES(?,?,?); 
	UPDATE DBPREFIXstaff SET last_login = CURRENT_TIMESTAMP WHERE id = ?;`
	staffID, err := getStaffID(username)
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql, staffID, key, time.Now().Add(time.Duration(time.Hour*730)), staffID) //TODO move amount of time to config file
	return err
}

// PermanentlyRemoveDeletedPosts removes all posts and files marked as deleted from the database
func PermanentlyRemoveDeletedPosts() error {
	const sql = `DELETE FROM DBPREFIXposts WHERE is_deleted;
	DELETE FROM DBPREFIXthreads WHERE is_deleted;`
	_, err := ExecSQL(sql)
	return err
}

// OptimizeDatabase peforms a database optimisation
func OptimizeDatabase() error {
	tableRows, tablesErr := QuerySQL("SHOW TABLES")
	if tablesErr != nil {
		return tablesErr
	}
	for tableRows.Next() {
		var table string
		tableRows.Scan(&table)
		if _, err := ExecSQL("OPTIMIZE TABLE " + table); err != nil {
			return err
		}
	}
	return nil
}

func getBoardIDFromURIOrNil(URI string) *int {
	ID, err := getBoardIDFromURI(URI)
	if err != nil {
		return nil
	}
	return &ID
}

// CreateFileBan creates a new ban on a file. If boards = nil, the ban is global.
func CreateFileBan(fileChecksum string, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfile_ban (board_id, staff_id, staff_note, checksum) VALUES board_id = ?, staff_id = ?, staff_note = ?, checksum = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileChecksum)
	return err
}

// CreateFileNameBan creates a new ban on a filename. If boards = nil, the ban is global.
func CreateFileNameBan(fileName string, isRegex bool, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfilename_ban (board_id, staff_id, staff_note, filename, is_regex) VALUES board_id = ?, staff_id = ?, staff_note = ?, filename = ?, is_regex = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileName, isRegex)
	return err
}

// CreateUserNameBan creates a new ban on a username. If boards = nil, the ban is global.
func CreateUserNameBan(userName string, isRegex bool, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXusername_ban (board_id, staff_id, staff_note, username, is_regex) VALUES board_id = ?, staff_id = ?, staff_note = ?, username = ?, is_regex = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, userName, isRegex)
	return err
}

// CreateUserBan creates either a full ip ban, or an ip ban for threads only, for a given IP.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CreateUserBan(IP string, threadBan bool, staffName string, boardURI string, expires time.Time, permaban bool,
	staffNote string, message string, canAppeal bool, appealAt time.Time) error {
	const sql = `INSERT INTO DBPREFIXip_ban (board_id, staff_id, staff_note, is_thread_ban, ip, appeal_at, expires_at, permanent, message, can_appeal, issued_at, copy_posted_text, is_active)
	VALUES (?,?,?,?,?,?,?,?,?,?,CURRENT_TIMESTAMP,'OLD SYSTEM BAN, NO TEXT AVAILABLE',TRUE)`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, threadBan, IP, appealAt, expires, permaban, message, canAppeal)
	return err
}

//GetAllAccouncements gets all announcements, newest first
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllAccouncements() ([]Announcement, error) {
	const sql = `SELECT s.username, a.timestamp, a.subject, a.message FROM DBPREFIXannouncements AS a
	JOIN DBPREFIXstaff AS s
	ON a.staff_id = s.id
	ORDER BY a.id DESC`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var announcements []Announcement
	for rows.Next() {
		var announcement Announcement
		err = rows.Scan(&announcement.Poster, &announcement.Timestamp, &announcement.Subject, &announcement.Message)
		if err != nil {
			return nil, err
		}
		announcements = append(announcements, announcement)
	}
	return announcements, nil
}

//CreateBoard creates this board in the database if it doesnt exist already, also sets ID to correct value
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CreateBoard(values *Board) error {
	const maxThreads = 300
	const sql = `INSERT INTO DBPREFIXboards (navbar_position, dir, uri, title, subtitle, description, max_file_size, max_threads, default_style, locked, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog, section_id)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING id`
	if values == nil {
		return errors.New("Board is nil")
	}
	return QueryRowSQL(
		sql,
		InterfaceSlice(values.ListOrder, values.Dir, values.Dir, values.Title, values.Subtitle, values.Description, values.MaxFilesize, maxThreads, values.DefaultStyle, values.Locked, values.Anonymous, values.ForcedAnon, values.AutosageAfter, values.NoImagesAfter, values.MaxMessageLength, 1, values.EmbedsAllowed, values.RedirectToThread, values.RequireFile, values.EnableCatalog, values.Section),
		InterfaceSlice(&values.ID))
}

//GetBoardUris gets a list of all existing board URIs
func GetBoardUris() (URIS []string, err error) {
	const sql = `SELECT uri FROM DBPREFIXboards`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var uris []string
	for rows.Next() {
		var uri string
		err = rows.Scan(&uri)
		if err != nil {
			return nil, err
		}
		uris = append(uris, uri)
	}
	return uris, nil
}

//GetAllSections gets a list of all existing sections
func GetAllSections() ([]BoardSection, error) {
	const sql = `SELECT id, name, abbreviation, position, hidden FROM DBPREFIXsections`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var sections []BoardSection
	for rows.Next() {
		var section BoardSection
		err = rows.Scan(&section.ID, &section.Name, &section.Abbreviation, &section.ListOrder, &section.Hidden)
		if err != nil {
			return nil, err
		}
		sections = append(sections, section)
	}
	return sections, nil
}

// GetAllSectionsOrCreateDefault gets all sections in the database, creates default if none exist
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllSectionsOrCreateDefault() ([]BoardSection, error) {
	_, err := GetOrCreateDefaultSectionID()
	if err != nil {
		return nil, err
	}
	return GetAllSections()
}

func getNextSectionListOrder() (int, error) {
	const sql = `SELECT COALESCE(MAX(position) + 1, 0) FROM DBPREFIXsections`
	var ID int
	err := QueryRowSQL(sql, InterfaceSlice(), InterfaceSlice(&ID))
	return ID, err
}

//GetOrCreateDefaultSectionID creates the default section if it does not exist yet, returns default section ID if it exists
func GetOrCreateDefaultSectionID() (sectionID int, err error) {
	const SQL = `SELECT id FROM DBPREFIXsections WHERE name = 'Main'`
	var ID int
	err = QueryRowSQL(SQL, InterfaceSlice(), InterfaceSlice(&ID))
	if err == sql.ErrNoRows {
		//create it
		ID, err := getNextSectionListOrder()
		if err != nil {
			return 0, err
		}
		board := BoardSection{Name: "Main", Abbreviation: "Main", Hidden: false, ListOrder: ID}
		err = CreateSection(&board)
		return board.ID, err
	}
	if err != nil {
		return 0, err //other error
	}
	return ID, nil
}

//CreateSection creates a section, setting the newly created id in the given struct
func CreateSection(section *BoardSection) error {
	const sql = `INSERT INTO DBPREFIXsections (name, abbreviation, hidden, position) VALUES (?,?,?,?)
	RETURNING id`
	return QueryRowSQL(
		sql,
		InterfaceSlice(section.Name, section.Abbreviation, section.Hidden, section.ListOrder),
		InterfaceSlice(&section.ID))
}

//GetAllStaffNopass gets all staff accounts without their password
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllStaffNopass() ([]Staff, error) {
	const sql = `SELECT id, username, global_rank, added_on, last_login FROM DBPREFIXstaff`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var staffs []Staff
	for rows.Next() {
		var staff Staff
		err = rows.Scan(&staff.ID, &staff.Username, &staff.Rank, &staff.AddedOn, &staff.LastActive)
		if err != nil {
			return nil, err
		}
		staffs = append(staffs, staff)
	}
	return staffs, nil
}

//GetAllBans gets a list of all bans
//Warning, currently only gets ip bans, not other types of bans, as the ban functionality needs a major revamp anyway
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBans() ([]BanInfo, error) {
	const sql = `SELECT 
	ban.id, 
	ban.ip, 
	COALESCE(board.title, '') as boardname,
	staff.username as staff,
	ban.issued_at,
	ban.expires_at,
	ban.permanent,
	ban.message,
	ban.staff_note,
	ban.appeal_at,
	ban.can_appeal
FROM DBPREFIXip_ban as ban
JOIN DBPREFIXstaff as staff
ON ban.staff_id = staff.id
JOIN DBPREFIXboards as board
ON ban.board_id = board.id`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var bans []BanInfo
	for rows.Next() {
		var ban BanInfo
		err = rows.Scan(&ban.ID, &ban.IP, &ban.Boards, &ban.Staff, &ban.Timestamp, &ban.Expires, &ban.Permaban, &ban.Reason, &ban.StaffNote, &ban.AppealAt, &ban.CanAppeal)
		if err != nil {
			return nil, err
		}
		bans = append(bans, ban)
	}
	return bans, nil
}

//CheckBan returns banentry if a ban was found or a sql.ErrNoRows if not banned
// name, filename and checksum may be empty strings and will be treated as not requested if done so
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CheckBan(ip string, name string, filename string, checksum string) (*BanInfo, error) {
	ban := new(BanInfo)
	ipban, err1 := checkIPBan(ip)
	_, err2 := checkFileBan(checksum)
	_, err3 := checkFilenameBan(filename)
	_, err4 := checkUsernameBan(name)

	if err1 == sql.ErrNoRows && err2 == sql.ErrNoRows && err3 == sql.ErrNoRows && err4 == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err1 != nil && err1 != sql.ErrNoRows {
		return nil, err1
	}
	if err2 != nil && err2 != sql.ErrNoRows {
		return nil, err2
	}
	if err3 != nil && err3 != sql.ErrNoRows {
		return nil, err3
	}
	if err4 != nil && err4 != sql.ErrNoRows {
		return nil, err4
	}

	if ipban != nil {
		ban.ID = 0
		ban.IP = string(ipban.IP)
		staff, _ := getStaffByID(ipban.StaffID)
		ban.Staff = staff.Username
		ban.Timestamp = ipban.IssuedAt
		ban.Expires = ipban.ExpiresAt
		ban.Permaban = ipban.Permanent
		ban.Reason = ipban.Message
		ban.StaffNote = ipban.StaffNote
		ban.AppealAt = ipban.AppealAt
		ban.CanAppeal = ipban.CanAppeal
		return ban, nil
	}

	//TODO implement other types of bans or refactor banning code
	return nil, errors.New("Not implemented")
}

func checkIPBan(ip string) (*IPBan, error) {
	const sql = `SELECT id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban, is_active, ip, issued_at, appeal_at, expires_at, permanent, staff_note, message, can_appeal
	FROM DBPREFIXusername_ban WHERE username = ?`
	var ban = new(IPBan)
	err := QueryRowSQL(sql, InterfaceSlice(ip), InterfaceSlice(&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &ban.CopyPostText, &ban.IsThreadBan, &ban.IsActive, &ban.IP, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message, &ban.CanAppeal))
	return ban, err
}

func checkUsernameBan(name string) (*UsernameBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, username, is_regex 
	FROM DBPREFIXusername_ban WHERE username = ?`
	var ban = new(UsernameBan)
	err := QueryRowSQL(sql, InterfaceSlice(name), InterfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Username, &ban.IsRegex))
	return ban, err
}

func checkFilenameBan(filename string) (*FilenameBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, filename, is_regex 
	FROM DBPREFIXfilename_ban WHERE filename = ?`
	var ban = new(FilenameBan)
	err := QueryRowSQL(sql, InterfaceSlice(filename), InterfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Filename, &ban.IsRegex))
	return ban, err
}

func checkFileBan(checksum string) (*FileBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, checksum 
	FROM DBPREFIXfile_ban WHERE checksum = ?`
	var ban = new(FileBan)
	err := QueryRowSQL(sql, InterfaceSlice(checksum), InterfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum))
	return ban, err
}

//SinceLastPost returns the seconds since the last post by the ip address that made this post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func SinceLastPost(postID int) (int, error) {
	const sql = `SELECT MAX(created_on) FROM DBPREFIXposts as posts
	JOIN (SELECT ip FROM DBPREFIXposts as sp
		 WHERE sp.id = ?) as ip
	ON posts.ip = ip.ip`
	var when time.Time
	err := QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&when))
	if err != nil {
		return -1, err
	}
	return int(time.Now().Sub(when).Seconds()), nil
}

// InsertPost insersts prepared post object into the SQL table so that it can be rendered
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func InsertPost(post *Post, bump bool) error {
	isNewThread := post.ParentID == 0
	var threadID int
	var err error
	if isNewThread {
		threadID, err = createThread(post.BoardID, post.Locked, post.Stickied, post.Autosage, false)
	} else {
		threadID, err = getThreadID(post.ParentID)
	}
	if err != nil {
		return err
	}

	//threadid, istoppost, ip, message, message_raw, password, banned_message, trip, rolesig, name, email, subject
	const sql = `INSERT INTO DBPREFIXposts (thread_id, name, tripcode, is_role_signature, email, subject, ip, is_top_post, message, message_raw, banned_message, password)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	RETURNING id`

	err = QueryRowSQL(sql,
		InterfaceSlice(threadID, post.Name, post.Tripcode, false, post.Email, post.Subject, post.IP, isNewThread, post.MessageHTML, post.MessageText, "", post.Password),
		InterfaceSlice(&post.ID))
	if err != nil {
		return err
	}

	if post.Filename != "" {
		err = appendFile(post.ID, post.FilenameOriginal, post.Filename, post.FileChecksum, post.Filesize, false, post.ImageW, post.ImageH, post.ThumbW, post.ThumbH)
	}
	if err != nil {
		return err
	}
	if bump {
		return bumpThread(threadID)
	}
	return nil
}

func createThread(boardID int, locked bool, stickied bool, anchored bool, cyclical bool) (threadID int, err error) {
	const sql = `INSERT INTO DBPREFIXthreads (board_id, locked, stickied, anchored, cyclical) VALUES (?,?,?,?,?)
	RETURNING id`
	err = QueryRowSQL(sql, InterfaceSlice(boardID, locked, stickied, anchored, cyclical), InterfaceSlice(&threadID))
	return threadID, err
}

func bumpThreadOfPost(postID int) error {
	id, err := getThreadID(postID)
	if err != nil {
		return err
	}
	return bumpThread(id)
}

func bumpThread(threadID int) error {
	const sql = "UPDATE DBPREFIXthreads SET last_bump = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := ExecSQL(sql, threadID)
	return err
}

func appendFile(postID int, originalFilename string, filename string, checksum string, fileSize int, isSpoilered bool, width int, height int, thumbnailWidth int, thumbnailHeight int) error {
	const nextIDSQL = `SELECT COALESCE(MAX(file_order) + 1, 0) FROM DBPREFIXfiles WHERE post_id = ?`
	var nextID int
	err := QueryRowSQL(nextIDSQL, InterfaceSlice(postID), InterfaceSlice(&nextID))
	if err != nil {
		return err
	}
	const insertSQL = `INSERT INTO DBPREFIXfiles (file_order, post_id, original_filename, filename, checksum, file_size, is_spoilered, width, height, thumbnail_width, thumbnail_height)
	VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	_, err = ExecSQL(insertSQL, nextID, postID, originalFilename, filename, checksum, fileSize, isSpoilered, width, height, thumbnailWidth, thumbnailHeight)
	return err
}

//GetMaxMessageLength returns the max message length on a board
func GetMaxMessageLength(boardID int) (length int, err error) {
	const sql = `SELECT max_message_length FROM DBPREFIXboards
	WHERE id = ?`
	err = QueryRowSQL(sql, InterfaceSlice(boardID), InterfaceSlice(&length))
	return length, err
}

//GetEmbedsAllowed returns if embeds are allowed on a given board
func GetEmbedsAllowed(boardID int) (allowed bool, err error) {
	const sql = `SELECT allow_embeds FROM DBPREFIXboards
	WHERE id = ?`
	err = QueryRowSQL(sql, InterfaceSlice(boardID), InterfaceSlice(&allowed))
	return allowed, err
}

//GetBoardFromPostID gets the boardURI that a given postid exists on
func GetBoardFromPostID(postID int) (boardURI string, err error) {
	const sql = `SELECT board.uri FROM DBPREFIXboards as board
	JOIN (
		SELECT threads.board_id FROM DBPREFIXthreads as threads
		JOIN DBPREFIXposts as posts ON posts.thread_id = threads.id
		WHERE posts.id = ?
	) as threads ON threads.board_id = board.id`
	err = QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&boardURI))
	return boardURI, err
}

//GetThreadIDZeroIfTopPost gets the post id of the top post of the thread a post belongs to, zero if the post itself is the top post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Posts do not directly reference their post post anymore.
func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	const sql = `SELECT t1.id FROM DBPREFIXposts as t1
	JOIN (SELECT thread_id FROM DBPREFIXposts where id = ?) as t2 ON t1.thread_id = t2.thread_id
	WHERE t1.is_top_post`
	err = QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&ID))
	if err != nil {
		return 0, err
	}
	if ID == postID {
		return 0, nil
	}
	return ID, nil
}

func getThreadID(postID int) (ID int, err error) {
	const sql = `SELECT thread_id FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&ID))
	return ID, err
}

//AddBanAppeal adds a given appeal to a given ban
func AddBanAppeal(banID uint, message string) error {
	const sql = `
	/*copy old to audit*/
	INSERT INTO DBPREFIXip_ban_appeals_audit (appeal_id, staff_id, appeal_text, staff_response, is_denied)
	SELECT id, staff_id, appeal_text, staff_response, is_denied
	FROM DBPREFIXip_ban_appeals
	WHERE DBPREFIXip_ban_appeals.ip_ban_id = ?;

	/*update old values to new values*/
	UPDATE DBPREFIXip_ban_appeals SET appeal_text = ? WHERE ip_ban_id = ?;
	`
	_, err := ExecSQL(sql, banID, message, banID)
	return err
}

//GetPostPassword gets the password associated with a given post
func GetPostPassword(postID int) (password string, err error) {
	const sql = `SELECT password_checksum FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&password))
	return password, err
}

//UpdatePost updates a post with new information
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UpdatePost(postID int, email string, subject string, message string, messageRaw string) error {
	const sql = `UPDATE DBPREFIXposts SET email = ?, subject = ?, message = ?, message_raw = ? WHERE id = ?`
	_, err := ExecSQL(sql, email, subject, message, messageRaw)
	return err
}

//DeleteFilesFromPost deletes all files belonging to a given post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Should be implemented to delete files individually
func DeleteFilesFromPost(postID int) error {
	board, err := GetBoardFromPostID(postID)
	if err != nil {
		return err
	}

	//Get all filenames
	const filenameSQL = `SELECT filename FROM DBPREFIXfiles WHERE post_id = ?`
	rows, err := QuerySQL(filenameSQL)
	if err != nil {
		return err
	}
	var filenames []string
	for rows.Next() {
		var filename string
		err = rows.Scan(&filename)
		if err != nil {
			return err
		}
		filenames = append(filenames, filename)
	}

	//Remove files from disk
	for _, fileName := range filenames {
		fileName = fileName[:strings.Index(fileName, ".")]
		fileType := fileName[strings.Index(fileName, ".")+1:]
		var thumbType string
		if fileType == "gif" || fileType == "webm" {
			thumbType = "jpg"
		}

		os.Remove(path.Join(config.Config.DocumentRoot, board, "/src/"+fileName+"."+fileType))
		os.Remove(path.Join(config.Config.DocumentRoot, board, "/thumb/"+fileName+"t."+thumbType))
		os.Remove(path.Join(config.Config.DocumentRoot, board, "/thumb/"+fileName+"c."+thumbType))
	}

	const removeFilesSQL = `DELETE FROM DBPREFIXfiles WHERE post_id = ?`
	_, err = ExecSQL(removeFilesSQL, postID)
	return err
}

//DeletePost deletes a post with a given ID
func DeletePost(postID int, checkIfTopPost bool) error {
	if checkIfTopPost {
		isTopPost, err := isTopPost(postID)
		if err != nil {
			return err
		}
		if isTopPost {
			threadID, err := getThreadID(postID)
			if err != nil {
				return err
			}
			return deleteThread(threadID)
		}
	}

	DeleteFilesFromPost(postID)
	const sql = `UPDATE DBPREFIXposts SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := ExecSQL(sql, postID)
	return err
}

func isTopPost(postID int) (val bool, err error) {
	const sql = `SELECT is_top_post FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, InterfaceSlice(postID), InterfaceSlice(&val))
	return val, err
}

func deleteThread(threadID int) error {
	const sql = `UPDATE DBPREFIXthreads SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?;
	
	SELECT id FROM DBPREFIXposts WHERE thread_id = ?;`

	rows, err := QuerySQL(sql, threadID, threadID)
	if err != nil {
		return err
	}
	var ids []int
	for rows.Next() {
		var id int
		err = rows.Scan(&id)
		if err != nil {
			return err
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		err = DeletePost(id, false)
		if err != nil {
			return err
		}
	}
	return nil
}

//CreateDefaultBoardIfNoneExist creates a default board if no boards exist yet
func CreateDefaultBoardIfNoneExist() error {
	const sql = `SELECT COUNT(id) FROM DBPREFIXboards`
	var count int
	QueryRowSQL(sql, InterfaceSlice(), InterfaceSlice(&count))
	if count > 0 {
		return nil
	}
	defaultSectionID, err := GetOrCreateDefaultSectionID()
	if err != nil {
		return err
	}
	return CreateBoard(
		&Board{
			Dir:         "test",
			Title:       "Testing board",
			Subtitle:    "Board for testing",
			Description: "Board for testing",
			Section:     defaultSectionID})
}

//CreateDefaultAdminIfNoStaff creates a new default admin account if no accounts exist
func CreateDefaultAdminIfNoStaff() error {
	const sql = `SELECT COUNT(id) FROM DBPREFIXstaff`
	var count int
	QueryRowSQL(sql, InterfaceSlice(), InterfaceSlice(&count))
	if count > 0 {
		return nil
	}
	_, err := createUser("admin", gcutil.BcryptSum("password"), 3)
	return err
}

func createUser(username string, passwordEncrypted string, globalRank int) (userID int, err error) {
	const sql = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank) VALUES (?,?,?)
	RETURNING id`
	err = QueryRowSQL(sql, InterfaceSlice(username, passwordEncrypted, globalRank), InterfaceSlice(&userID))
	return userID, err
}

//UpdateID takes a board struct and sets the database id according to the dir that is already set
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. (Just bad design in general, try to avoid directly mutating state like this)
func (board *Board) UpdateID() error {
	const sql = `SELECT id FROM DBPREFIXboards WHERE dir = ?`
	return QueryRowSQL(sql, InterfaceSlice(board.Dir), InterfaceSlice(&board.ID))
}

// PopulateData gets the board data from the database, according to its id, and sets the respective properties.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func (board *Board) PopulateData(id int) error {
	const sql = "SELECT id, section_id, dir, navbar_position, title, subtitle, description, max_file_size, default_style, locked, created_at, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog FROM DBPREFIXboards WHERE id = ?"
	return QueryRowSQL(sql, InterfaceSlice(id), InterfaceSlice(&board.ID, &board.Section, &board.Dir, &board.ListOrder, &board.Title, &board.Subtitle, &board.Description, &board.MaxFilesize, &board.DefaultStyle, &board.Locked, &board.CreatedOn, &board.Anonymous, &board.ForcedAnon, &board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed, &board.RedirectToThread, &board.RequireFile, &board.EnableCatalog))
}

//DoesBoardExistByID returns a bool indicating whether a board with a given id exists
func DoesBoardExistByID(ID int) (bool, error) {
	const sql = `SELECT COUNT(id) FROM DBPREFIXboards WHERE id = ?`
	var count int
	err := QueryRowSQL(sql, InterfaceSlice(ID), InterfaceSlice(&count))
	return count > 0, err
}

//GetAllBoards gets a list of all existing boards
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBoards() ([]Board, error) {
	const sql = "SELECT id, section_id, dir, navbar_position, title, subtitle, description, max_file_size, default_style, locked, created_at, anonymous_name, force_anonymous, autosage_after, no_images_after, max_message_length, allow_embeds, redirect_to_thread, require_file, enable_catalog FROM DBPREFIXboards"
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

//GetBoardFromID returns the board corresponding to a given id
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetBoardFromID(boardID int) (Board, error) {
	var board Board
	err := board.PopulateData(boardID)
	return board, err
}

func getBoardIDFromURI(URI string) (id int, err error) {
	const sql = `SELECT id FROM DBPREFIXboards WHERE uri = ?`
	err = QueryRowSQL(sql, InterfaceSlice(URI), InterfaceSlice(&id))
	return id, err
}
