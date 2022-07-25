package gcsql

import (
	"database/sql"
	"errors"
	"html/template"
	"regexp"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
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
	const sql = `SELECT posts.id, posts.message, posts.message_raw, DBPREFIXboards.dir as dir
	FROM DBPREFIXposts as posts, DBPREFIXboards
	where posts.is_deleted = FALSE`
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var messages []MessagePostContainer
	for rows.Next() {
		var message MessagePostContainer
		var formattedHTML template.HTML
		if err = rows.Scan(&message.ID, &formattedHTML, &message.MessageRaw, &message.Board); err != nil {
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
	stmt, err := PrepareSQL(sql, nil)
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

// CreateWordFilter inserts the given wordfilter data into the database and returns a pointer to a new WordFilter struct
func CreateWordFilter(from string, to string, isRegex bool, boards []string, staffID int, staffNote string) (*WordFilter, error) {
	var err error
	if isRegex {
		_, err = regexp.Compile(from)
		if err != nil {
			return nil, err
		}
	}

	_, err = ExecSQL(`INSERT INTO DBPREFIXwordfilters
		(board_dirs,staff_id,staff_note,search,is_regex,change_to)
		VALUES(?,?,?,?,?,?)`, strings.Join(boards, ","), staffID, staffNote, from, isRegex, to)
	if err != nil {
		return nil, err
	}
	return &WordFilter{
		BoardDirs: boards,
		StaffID:   staffID,
		StaffNote: staffNote,
		IssuedAt:  time.Now(),
		Search:    from,
		IsRegex:   isRegex,
		ChangeTo:  to,
	}, err
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
		if dirsStr != "*" {
			wf.BoardDirs = strings.Split(dirsStr, ",")
		}
		wfs = append(wfs, wf)
	}
	return wfs, err
}

func GetBoardWordFilters(board string) ([]WordFilter, error) {
	wfs, err := GetWordFilters()
	if err != nil {
		return wfs, err
	}
	var boardFilters []WordFilter
	for _, wf := range wfs {
		if wf.OnBoard(board) {
			boardFilters = append(boardFilters, wf)
		}
	}
	return boardFilters, nil
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

func (wf *WordFilter) StaffName() string {
	staff, err := getStaffByID(wf.StaffID)
	if err != nil {
		return "?"
	}
	return staff.Username
}

// Apply runs the current wordfilter on the given string, without checking the board or (re)building the post
// It returns an error if it is a regular expression and regexp.Compile failed to parse it
func (wf *WordFilter) Apply(message string) (string, error) {
	if wf.IsRegex {
		re, err := regexp.Compile(wf.Search)
		if err != nil {
			return message, err
		}
		message = re.ReplaceAllString(message, wf.ChangeTo)
	} else {
		message = strings.Replace(message, wf.Search, wf.ChangeTo, -1)
	}
	return message, nil
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
