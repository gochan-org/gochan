package gcsql

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// GochanVersionKeyConstant is the key value used in the version table of the database to store and receive the (database) version of base gochan
const GochanVersionKeyConstant = "gochan"

var (
	ErrNilBoard          = errors.New("board is nil")
	ErrBoardExists       = errors.New("board already exists")
	ErrBoardDoesNotExist = errors.New("board does not exist")
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
		var formattedHTML template.HTML
		if err = rows.Scan(&message.ID, &formattedHTML, &message.MessageRaw); err != nil {
			return nil, err
		}
		message.Message = template.HTML(formattedHTML)
		messages = append(messages, message)
	}
	return messages, nil
}

// SetFormattedInDatabase sets all the non-raw text for a given array of items.
func SetFormattedInDatabase(messages []MessagePostContainer) error {
	const sql = `UPDATE DBPREFIXposts
	SET message = ?
	WHERE id = ?`
	stmt, err := PrepareSQL(sql)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, message := range messages {
		if _, err = stmt.Exec(string(message.Message), message.ID); err != nil {
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
	err := QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&count))
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
	err := QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&count))
	return count, err
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (string, error) {
	const sql = `SELECT staff.username from DBPREFIXstaff as staff
	JOIN DBPREFIXsessions as sessions
	ON sessions.staff_id = staff.id
	WHERE sessions.data = ?`
	var username string
	err := QueryRowSQL(sql, interfaceSlice(session), interfaceSlice(&username))
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
	err := QueryRowSQL(sql, interfaceSlice(session), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
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
	err := QueryRowSQL(sql, interfaceSlice(name), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
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
	err := QueryRowSQL(sql, interfaceSlice(id), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

// NewStaff creates a new staff account from a given username, password and rank
func NewStaff(username, password string, rank int) error {
	const sql = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank)
	VALUES (?, ?, ?)`
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
func CreateSession(key, username string) error {
	const sql1 = `INSERT INTO DBPREFIXsessions (staff_id,data,expires) VALUES(?,?,?)`
	const sql2 = `UPDATE DBPREFIXstaff SET last_login = CURRENT_TIMESTAMP WHERE id = ?`
	staffID, err := getStaffID(username)
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql1, staffID, key, time.Now().Add(time.Duration(time.Hour*730))) //TODO move amount of time to config file
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql2, staffID)
	return err
}

// PermanentlyRemoveDeletedPosts removes all posts and files marked as deleted from the database
func PermanentlyRemoveDeletedPosts() error {
	const sql1 = `DELETE FROM DBPREFIXposts WHERE is_deleted`
	const sql2 = `DELETE FROM DBPREFIXthreads WHERE is_deleted`
	_, err := ExecSQL(sql1)
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql2)
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
func CreateFileBan(fileChecksum, staffName string, permaban bool, staffNote, boardURI string) error {
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
func CreateFileNameBan(fileName string, isRegex bool, staffName string, permaban bool, staffNote, boardURI string) error {
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
func CreateUserNameBan(userName string, isRegex bool, staffName string, permaban bool, staffNote, boardURI string) error {
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
func CreateUserBan(IP string, threadBan bool, staffName, boardURI string, expires time.Time, permaban bool,
	staffNote, message string, canAppeal bool, appealAt time.Time) error {
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
	announcements := []Announcement{}
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
		if err = rows.Scan(&uri); err != nil {
			return nil, err
		}
		uris = append(uris, uri)
	}
	return uris, nil
}

//GetAllSections gets a list of all existing sections
func GetAllSections() ([]BoardSection, error) {
	const sql = `SELECT id, name, abbreviation, position, hidden FROM DBPREFIXsections ORDER BY position ASC, name ASC`
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
	err := QueryRowSQL(sql, interfaceSlice(), interfaceSlice(&ID))
	return ID, err
}

//GetOrCreateDefaultSectionID creates the default section if it does not exist yet, returns default section ID if it exists
func GetOrCreateDefaultSectionID() (sectionID int, err error) {
	const SQL = `SELECT id FROM DBPREFIXsections WHERE name = 'Main'`
	var ID int
	err = QueryRowSQL(SQL, interfaceSlice(), interfaceSlice(&ID))
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
	const sqlINSERT = `INSERT INTO DBPREFIXsections (name, abbreviation, hidden, position) VALUES (?,?,?,?)`
	const sqlSELECT = `SELECT id FROM DBPREFIXsections WHERE position = ?`
	//Excecuted in two steps this way because last row id functions arent thread safe, position is unique
	_, err := ExecSQL(sqlINSERT, section.Name, section.Abbreviation, section.Hidden, section.ListOrder)
	if err != nil {
		return err
	}
	return QueryRowSQL(
		sqlSELECT,
		interfaceSlice(section.ListOrder),
		interfaceSlice(&section.ID))
}

//GetAllStaffNopass gets all staff accounts without their password
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllStaffNopass(onlyactive bool) ([]Staff, error) {
	sql := `SELECT id, username, global_rank, added_on, last_login FROM DBPREFIXstaff`
	if onlyactive {
		sql += " where is_active = 1"
	}
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
func CheckBan(ip, name, filename, checksum string) (*BanInfo, error) {
	ban := new(BanInfo)
	ipban, err1 := checkIPBan(ip)
	err1NoRows := (err1 == sql.ErrNoRows)
	_, err2 := checkFileBan(checksum)
	err2NoRows := (err2 == sql.ErrNoRows)
	_, err3 := checkFilenameBan(filename)
	err3NoRows := (err3 == sql.ErrNoRows)
	_, err4 := checkUsernameBan(name)
	err4NoRows := (err4 == sql.ErrNoRows)

	if err1NoRows && err2NoRows && err3NoRows && err4NoRows {
		return nil, sql.ErrNoRows
	}

	if err1NoRows {
		return nil, err1
	}
	if err2NoRows {
		return nil, err2
	}
	if err3NoRows {
		return nil, err3
	}
	if err4NoRows {
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
	return nil, gcutil.ErrNotImplemented
}

func checkIPBan(ip string) (*IPBan, error) {
	const sql = `SELECT id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban, is_active, ip, issued_at, appeal_at, expires_at, permanent, staff_note, message, can_appeal
	FROM DBPREFIXip_ban WHERE ip = ?`
	var ban = new(IPBan)
	var formattedHTMLcopyposttest template.HTML
	err := QueryRowSQL(sql, interfaceSlice(ip), interfaceSlice(&ban.ID, &ban.StaffID, &ban.BoardID, &ban.BannedForPostID, &formattedHTMLcopyposttest, &ban.IsThreadBan, &ban.IsActive, &ban.IP, &ban.IssuedAt, &ban.AppealAt, &ban.ExpiresAt, &ban.Permanent, &ban.StaffNote, &ban.Message, &ban.CanAppeal))
	ban.CopyPostText = formattedHTMLcopyposttest
	return ban, err
}

func checkUsernameBan(name string) (*UsernameBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, username, is_regex 
	FROM DBPREFIXusername_ban WHERE username = ?`
	var ban = new(UsernameBan)
	err := QueryRowSQL(sql, interfaceSlice(name), interfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Username, &ban.IsRegex))
	return ban, err
}

func checkFilenameBan(filename string) (*FilenameBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, filename, is_regex 
	FROM DBPREFIXfilename_ban WHERE filename = ?`
	var ban = new(FilenameBan)
	err := QueryRowSQL(sql, interfaceSlice(filename), interfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Filename, &ban.IsRegex))
	return ban, err
}

func checkFileBan(checksum string) (*FileBan, error) {
	const sql = `SELECT id, board_id, staff_id, staff_note, issued_at, checksum 
	FROM DBPREFIXfile_ban WHERE checksum = ?`
	var ban = new(FileBan)
	err := QueryRowSQL(sql, interfaceSlice(checksum), interfaceSlice(&ban.ID, &ban.BoardID, &ban.StaffID, &ban.StaffNote, &ban.IssuedAt, &ban.Checksum))
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
	err := QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&when))
	if err != nil {
		return -1, err
	}

	return int(time.Since(when).Seconds()), nil
}

// InsertPost insersts prepared post object into the SQL table so that it can be rendered
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func InsertPost(post *Post, bump bool) error {
	const sql = `INSERT INTO DBPREFIXposts (id, thread_id, name, tripcode, is_role_signature, email, subject, ip, is_top_post, message, message_raw, banned_message, password)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
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

	//Retrieves next free ID, explicitly inserts it, keeps retrying until succesfull insert or until a non-pk error is encountered.
	//This is done because mysql doesnt support RETURNING and both LAST_INSERT_ID() and last_row_id() are not thread-safe
	isPrimaryKeyError := true
	for isPrimaryKeyError {
		nextFreeID, err := getNextFreeID("DBPREFIXposts")
		if err != nil {
			return err
		}
		_, err = ExecSQL(sql, nextFreeID, threadID, post.Name, post.Tripcode, false, post.Email, post.Subject, post.IP, isNewThread, string(post.MessageHTML), post.MessageText, "", post.Password)

		isPrimaryKeyError, err = errFilterDuplicatePrimaryKey(err)
		if err != nil {
			return err
		}
		if !isPrimaryKeyError {
			post.ID = nextFreeID
		}
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

func createThread(boardID int, locked, stickied, anchored, cyclical bool) (threadID int, err error) {
	const sql = `INSERT INTO DBPREFIXthreads (board_id, locked, stickied, anchored, cyclical) VALUES (?,?,?,?,?)`
	//Retrieves next free ID, explicitly inserts it, keeps retrying until succesfull insert or until a non-pk error is encountered.
	//This is done because mysql doesnt support RETURNING and both LAST_INSERT_ID() and last_row_id() are not thread-safe
	isPrimaryKeyError := true
	for isPrimaryKeyError {
		threadID, err = getNextFreeID("DBPREFIXthreads")
		if err != nil {
			return 0, err
		}
		_, err = ExecSQL(sql, boardID, locked, stickied, anchored, cyclical)

		isPrimaryKeyError, err = errFilterDuplicatePrimaryKey(err)
		if err != nil {
			return 0, err
		}
	}
	return threadID, nil
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

func appendFile(postID int, originalFilename, filename, checksum string, fileSize int, isSpoilered bool, width, height, thumbnailWidth, thumbnailHeight int) error {
	const nextIDSQL = `SELECT COALESCE(MAX(file_order) + 1, 0) FROM DBPREFIXfiles WHERE post_id = ?`
	var nextID int
	err := QueryRowSQL(nextIDSQL, interfaceSlice(postID), interfaceSlice(&nextID))
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
	err = QueryRowSQL(sql, interfaceSlice(boardID), interfaceSlice(&length))
	return length, err
}

//GetEmbedsAllowed returns if embeds are allowed on a given board
func GetEmbedsAllowed(boardID int) (allowed bool, err error) {
	const sql = `SELECT allow_embeds FROM DBPREFIXboards
	WHERE id = ?`
	err = QueryRowSQL(sql, interfaceSlice(boardID), interfaceSlice(&allowed))
	return allowed, err
}

//GetBoardFromPostID gets the boardURI that a given postid exists on
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

//GetThreadIDZeroIfTopPost gets the post id of the top post of the thread a post belongs to, zero if the post itself is the top post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Posts do not directly reference their post post anymore.
func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	const sql = `SELECT t1.id FROM DBPREFIXposts as t1
	JOIN (SELECT thread_id FROM DBPREFIXposts where id = ?) as t2 ON t1.thread_id = t2.thread_id
	WHERE t1.is_top_post`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&ID))
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
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&ID))
	return ID, err
}

//AddBanAppeal adds a given appeal to a given ban
func AddBanAppeal(banID uint, message string) error {
	const sql1 = `
	/*copy old to audit*/
	INSERT INTO DBPREFIXip_ban_appeals_audit (appeal_id, staff_id, appeal_text, staff_response, is_denied)
	SELECT id, staff_id, appeal_text, staff_response, is_denied
	FROM DBPREFIXip_ban_appeals
	WHERE DBPREFIXip_ban_appeals.ip_ban_id = ?`
	const sql2 = `
	/*update old values to new values*/
	UPDATE DBPREFIXip_ban_appeals SET appeal_text = ? WHERE ip_ban_id = ?
	`
	_, err := ExecSQL(sql1, banID)
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql2, message, banID)
	return err
}

//GetPostPassword gets the password associated with a given post
func GetPostPassword(postID int) (password string, err error) {
	const sql = `SELECT password_checksum FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&password))
	return password, err
}

//UpdatePost updates a post with new information
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UpdatePost(postID int, email, subject string, message template.HTML, messageRaw string) error {
	const sql = `UPDATE DBPREFIXposts SET email = ?, subject = ?, message = ?, message_raw = ? WHERE id = ?`
	_, err := ExecSQL(sql, email, subject, string(message), messageRaw)
	return err
}

//DeleteFilesFromPost deletes all files belonging to a given post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Should be implemented to delete files individually
func DeleteFilesFromPost(postID int) error {
	board, boardWasFound, err := GetBoardFromPostID(postID)
	if err != nil {
		return err
	}
	if !boardWasFound {
		return fmt.Errorf("could not find board for post %v", postID)
	}

	//Get all filenames
	const filenameSQL = `SELECT filename FROM DBPREFIXfiles WHERE post_id = ?`
	rows, err := QuerySQL(filenameSQL, postID)
	if err != nil {
		return err
	}
	var filenames []string
	for rows.Next() {
		var filename string
		if err = rows.Scan(&filename); err != nil {
			return err
		}
		filenames = append(filenames, filename)
	}

	systemCriticalCfg := config.GetSystemCriticalConfig()

	//Remove files from disk
	for _, fileName := range filenames {
		_, filenameBase, fileExt := gcutil.GetFileParts(fileName)

		thumbExt := fileExt
		if thumbExt == "gif" || thumbExt == "webm" || thumbExt == "mp4" {
			thumbExt = "jpg"
		}

		uploadPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/src/", filenameBase+"."+fileExt)
		thumbPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/thumb/", filenameBase+"t."+thumbExt)
		catalogThumbPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/thumb/", filenameBase+"c."+thumbExt)

		os.Remove(uploadPath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
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
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&val))
	return val, err
}

func deleteThread(threadID int) error {
	const sql1 = `UPDATE DBPREFIXthreads SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	const sql2 = `SELECT id FROM DBPREFIXposts WHERE thread_id = ?`

	_, err := QuerySQL(sql1, threadID)
	if err != nil {
		return err
	}
	rows, err := QuerySQL(sql2, threadID)
	if err != nil {
		return err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		if err = DeletePost(id, false); err != nil {
			return err
		}
	}
	return nil
}

//CreateDefaultBoardIfNoneExist creates a default board if no boards exist yet
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
		gclog.Println(gclog.LFatal|gclog.LStdLog, err.Error())
		return err
	}
	return nil
}

//CreateDefaultAdminIfNoStaff creates a new default admin account if no accounts exist
func CreateDefaultAdminIfNoStaff() error {
	const sql = `SELECT COUNT(id) FROM DBPREFIXstaff`
	var count int
	QueryRowSQL(sql, interfaceSlice(), interfaceSlice(&count))
	if count > 0 {
		return nil
	}
	_, err := createUser("admin", gcutil.BcryptSum("password"), 3)
	return err
}

func createUser(username, passwordEncrypted string, globalRank int) (userID int, err error) {
	const sqlInsert = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank) VALUES (?,?,?)`
	const sqlSelect = `SELECT id FROM DBPREFIXstaff WHERE username = ?`
	//Excecuted in two steps this way because last row id functions arent thread safe, username is unique
	_, err = ExecSQL(sqlInsert, username, passwordEncrypted, globalRank)
	if err != nil {
		return 0, err
	}
	err = QueryRowSQL(sqlSelect, interfaceSlice(username), interfaceSlice(&userID))
	return userID, err
}

//UpdateID takes a board struct and sets the database id according to the dir that is already set
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
	if err != nil {
		return err
	}
	absPath := board.AbsolutePath()
	gclog.Printf(gclog.LStaffLog,
		"Deleting board /%s/, absolute path: %s\n", board.Dir, absPath)
	err = os.RemoveAll(absPath)
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

//GetAllBoards gets a list of all existing boards
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
	err = QueryRowSQL(sql, interfaceSlice(URI), interfaceSlice(&id))
	return id, err
}

// GetWordFilters gets a list of wordfilters from the database and returns an array of them and any errors
// encountered
func GetWordFilters() ([]WordFilter, error) {
	var wfs []WordFilter
	query := `SELECT id,board_dirs,staff_id,staff_note,issued_at,search,is_regex,change_to FROM DBPREFIXwordfilters`
	rows, err := QuerySQL(query)
	if err != nil {
		return wfs, err
	}
	defer rows.Close()
	for rows.Next() {
		var dirsStr string
		var wf WordFilter
		if err = rows.Scan(
			&wf.ID,
			&dirsStr,
			&wf.StaffID,
			&wf.StaffNote,
			&wf.IssuedAt,
			&wf.Search,
			&wf.IsRegex,
			&wf.ChangeTo,
		); err != nil {
			return wfs, err
		}
		if dirsStr == "*" {
			// wordfilter applies to all boards
			continue
		}
		wf.BoardDirs = strings.Split(dirsStr, ",")
		wfs = append(wfs, wf)
	}
	return wfs, err
}

// BoardString returns a string representing the boards that this wordfilter applies to,
// or "*" if the filter should be applied to posts on all boards
func (wf *WordFilter) BoardsString() string {
	if wf.BoardDirs == nil {
		return "*"
	}
	return strings.Join(wf.BoardDirs, ",")
}

func (wf *WordFilter) OnBoard(dir string) bool {
	if dir == "*" {
		return true
	}
	for _, board := range wf.BoardDirs {
		if dir == board {
			return true
		}
	}
	return false
}

//getDatabaseVersion gets the version of the database, or an error if none or multiple exist
func getDatabaseVersion(componentKey string) (int, error) {
	const sql = `SELECT version FROM DBPREFIXdatabase_version WHERE component = ?`
	var version int
	err := QueryRowSQL(sql, []interface{}{componentKey}, []interface{}{&version})
	if err != nil {
		return 0, err
	}
	return version, err
}

func getNextFreeID(tableName string) (ID int, err error) {
	var sql = `SELECT COALESCE(MAX(id), 0) + 1 FROM ` + tableName
	err = QueryRowSQL(sql, interfaceSlice(), interfaceSlice(&ID))
	return ID, err
}

func doesTableExist(tableName string) (bool, error) {
	const existQuery = `SELECT COUNT(*)
	FROM INFORMATION_SCHEMA.TABLES
	WHERE TABLE_NAME = ?`

	var count int
	err := QueryRowSQL(existQuery, []interface{}{config.GetSystemCriticalConfig().DBprefix + tableName}, []interface{}{&count})
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

//doesGochanPrefixTableExist returns true if any table with a gochan prefix was found.
//Returns false if the prefix is an empty string
func doesGochanPrefixTableExist() (bool, error) {
	if config.GetSystemCriticalConfig().DBprefix == "" {
		return false, nil
	}
	var prefixTableExist = `SELECT count(*) 
	FROM INFORMATION_SCHEMA.TABLES
	WHERE TABLE_NAME LIKE 'DBPREFIX%'`

	var count int
	err := QueryRowSQL(prefixTableExist, []interface{}{}, []interface{}{&count})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
