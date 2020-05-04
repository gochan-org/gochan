package gcsql

import (
	"errors"
	"net"
	"time"

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

// FileBan creates a new ban on a file. If boards = nil, the ban is global.
func FileBan(fileChecksum string, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfile_ban (board_id, staff_id, staff_note, checksum) VALUES board_id = ?, staff_id = ?, staff_note = ?, checksum = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileChecksum)
	return err
}

// FileNameBan creates a new ban on a filename. If boards = nil, the ban is global.
func FileNameBan(fileName string, isRegex bool, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXfilename_ban (board_id, staff_id, staff_note, filename, is_regex) VALUES board_id = ?, staff_id = ?, staff_note = ?, filename = ?, is_regex = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, fileName, isRegex)
	return err
}

// UserNameBan creates a new ban on a username. If boards = nil, the ban is global.
func UserNameBan(userName string, isRegex bool, staffName string, permaban bool, staffNote string, boardURI string) error {
	const sql = `INSERT INTO DBPREFIXusername_ban (board_id, staff_id, staff_note, username, is_regex) VALUES board_id = ?, staff_id = ?, staff_note = ?, username = ?, is_regex = ?`
	staffID, err := getStaffID(staffName)
	if err != nil {
		return err
	}
	boardID := getBoardIDFromURIOrNil(boardURI)
	_, err = ExecSQL(sql, boardID, staffID, staffNote, userName, isRegex)
	return err
}

// UserBan creates either a full ip ban, or an ip ban for threads only, for a given IP.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UserBan(IP net.IP, threadBan bool, staffName string, boardURI string, expires time.Time, permaban bool,
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
	//("SELECT subject,message,poster,timestamp FROM DBPREFIXannouncements ORDER BY id DESC")
	//rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)

	return nil, errors.New("Not implemented")
}

//CreateBoard creates this board in the database if it doesnt exist already, also sets ID to correct value
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CreateBoard(values *Board) error {
	/*
		"INSERT INTO DBPREFIXboards (list_order,dir,type,upload_type,title,subtitle,"+
			"description,section,max_file_size,max_pages,default_style,locked,created_on,"+
			"anonymous,forced_anon,max_age,autosage_after,no_images_after,max_message_length,embeds_allowed,"+
			"redirect_to_thread,require_file,enable_catalog) "+
			"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		board.ListOrder, board.Dir, board.Type, board.UploadType,
		board.Title, board.Subtitle, board.Description, board.Section,
		board.MaxFilesize, board.MaxPages, board.DefaultStyle,
		board.Locked, getSpecificSQLDateTime(board.CreatedOn), board.Anonymous,
		board.ForcedAnon, board.MaxAge, board.AutosageAfter,
		board.NoImagesAfter, board.MaxMessageLength, board.EmbedsAllowed,
		board.RedirectToThread, board.RequireFile, board.EnableCatalog,*/
	return errors.New("Not implemented")
	//set id to created id
	//errors.New("board already exists in database")
}

//GetBoardUris gets a list of all existing board URIs
func GetBoardUris() (URIS []string, err error) {
	/*
		rows, err = querySQL("SELECT dir FROM DBPREFIXboards")
					defer closeHandle(rows)
					if err != nil {
						return html + gclog.Print(lErrorLog, "Error getting board list: ", err.Error())
					}

					for rows.Next() {
						var boardDir string
						rows.Scan(&boardDir)
						html += "<option>" + boardDir + "</option>"
					}
	*/
	return nil, errors.New("Not implemented")
}

//GetAllSections gets a list of all existing sections
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllSections() (sections []BoardSection, err error) {
	return nil, errors.New("Not implemented")
}

// GetAllSectionsOrCreateDefault gets all sections in the database, creates default if none exist
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllSectionsOrCreateDefault() (sections []BoardSection, err error) {
	// allSections, _ = getSectionArr("")
	// if len(allSections) == 0 {
	// 	if _, err = execSQL(
	// 		"INSERT INTO DBPREFIXsections (hidden,name,abbreviation) VALUES(0,'Main','main')",
	// 	); err != nil {
	// 		gclog.Print(lErrorLog, "Error creating new board section: ", err.Error())
	// 	}
	// }
	// allSections, _ = getSectionArr("")
	// return allSections
	return nil, errors.New("Not implemented")
}

//CreateDefaultSectionIfNotExist creates the default section if it does not exist yet
func CreateDefaultSectionIfNotExist() error {
	return errors.New("Not implemented")
}

//GetAllStaffNopass gets all staff accounts without their password
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllStaffNopass() ([]Staff, error) {

	return nil, errors.New("Not implemented")
}

//GetAllBans gets a list of all bans
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBans() ([]BanInfo, error) {
	// rows, err := querySQL("SELECT ip,name,reason,boards,staff,timestamp,expires,permaban,can_appeal FROM DBPREFIXbanlist")
	// defer closeHandle(rows)
	// if err != nil {
	// 	return pageHTML + gclog.Print(lErrorLog, "Error getting ban list: ", err.Error())
	// }

	// var banlist []BanInfo
	// for rows.Next() {
	// 	var ban BanInfo
	// 	rows.Scan(&ban.IP, &ban.Name, &ban.Reason, &ban.Boards, &ban.Staff, &ban.Timestamp, &ban.Expires, &ban.Permaban, &ban.CanAppeal)
	// 	banlist = append(banlist, ban)
	// }
	return nil, errors.New("Not implemented")
}

//CheckBan returns banentry if a ban was found or a sql.ErrNoRows if not banned
// name, filename and checksum may be empty strings and will be treated as not requested is done so
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func CheckBan(ip string, name string, filename string, checksum string) (*BanInfo, error) {
	//Note to coder, extremely shoddy code uses this function, tread carefully

	// in := []interface{}{ip}
	// query := "SELECT id,ip,name,boards,timestamp,expires,permaban,reason,type,appeal_at,can_appeal FROM DBPREFIXbanlist WHERE ip = ? "

	// if tripcode != "" {
	// 	in = append(in, tripcode)
	// 	query += "OR name = ? "
	// }
	// if filename != "" {
	// 	in = append(in, filename)
	// 	query += "OR filename = ? "
	// }
	// if checksum != "" {
	// 	in = append(in, checksum)
	// 	query += "OR file_checksum = ? "
	// }
	// query += " ORDER BY id DESC LIMIT 1"

	// err = queryRowSQL(query, in, []interface{}{
	// 	&banEntry.ID, &banEntry.IP, &banEntry.Name, &banEntry.Boards, &banEntry.Timestamp,
	// 	&banEntry.Expires, &banEntry.Permaban, &banEntry.Reason, &banEntry.Type,
	// 	&banEntry.AppealAt, &banEntry.CanAppeal},
	// )
	// return &banEntry, err
	return nil, errors.New("Not implemented")
}

//SinceLastPost returns the seconds since the last post by the ip address that made this post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func SinceLastPost(post *Post) int {
	// var lastPostTime time.Time
	// if err := queryRowSQL("SELECT timestamp FROM DBPREFIXposts WHERE ip = ? ORDER BY timestamp DESC LIMIT 1",
	// 	[]interface{}{post.IP},
	// 	[]interface{}{&lastPostTime},
	// ); err == sql.ErrNoRows {
	// 	// no posts by that IP.
	// 	return -1
	// }
	return -1 //int(time.Since(lastPostTime).Seconds())
}

// InsertPost insersts prepared post object into the SQL table so that it can be rendered
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func InsertPost(post *Post, bump bool) error {
	// queryStr := "INSERT INTO DBPREFIXposts " +
	// 	"(boardid,parentid,name,tripcode,email,subject,message,message_raw,password,filename,filename_original,file_checksum,filesize,image_w,image_h,thumb_w,thumb_h,ip,tag,timestamp,autosage,deleted_timestamp,bumped,stickied,locked,reviewed)" +
	// 	"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"

	// result, err := execSQL(queryStr,
	// 	post.BoardID, post.ParentID, post.Name, post.Tripcode, post.Email,
	// 	post.Subject, post.MessageHTML, post.MessageText, post.Password,
	// 	post.Filename, post.FilenameOriginal, post.FileChecksum, post.Filesize,
	// 	post.ImageW, post.ImageH, post.ThumbW, post.ThumbH, post.IP, post.Capcode,
	// 	post.Timestamp, post.Autosage, post.DeletedTimestamp, post.Bumped,
	// 	post.Stickied, post.Locked, post.Reviewed)
	// if err != nil {
	// 	return err
	// }

	// switch config.DBtype {
	// case "mysql":
	// 	var postID int64
	// 	postID, err = result.LastInsertId()
	// 	post.ID = int(postID)
	// case "postgres":
	// 	err = queryRowSQL("SELECT currval(pg_get_serial_sequence('DBPREFIXposts','id'))", nil, []interface{}{&post.ID})
	// case "sqlite3":
	// 	err = queryRowSQL("SELECT LAST_INSERT_ROWID()", nil, []interface{}{&post.ID})
	// }

	// // Bump parent post if requested.
	// if err != nil && post.ParentID != 0 && bump {
	// 	err = BumpThread(post.ParentID, post.BoardID)
	// }
	// return err
	return errors.New("Not implemented")
}

//GetMaxMessageLength returns the max message length on a board
func GetMaxMessageLength(boardID int) (int, error) {

	return 0, errors.New("Not implemented")
}

//GetEmbedsAllowed returns if embeds are allowed on a given board
func GetEmbedsAllowed(boardID int) (bool, error) {

	return false, errors.New("Not implemented")
}

//GetBoardFromPostID gets the boardURI that a given postid exists on
func GetBoardFromPostID(postID int) (boardURI string, err error) {
	return "", errors.New("Not implemented")
}

//GetThreadIDZeroIfTopPost gets the post id of the top post of the thread a post belongs to, zero if the post itself is the top post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Posts do not directly reference their post post anymore.
func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	return 0, errors.New("Not implemented")
}

//AddBanAppeal adds a given appeal to a given ban
func AddBanAppeal(banID uint, message string) error {
	return errors.New("Not implemented")
}

//GetPostPassword gets the password associated with a given post
func GetPostPassword(postID int) (password string, err error) {
	return "", errors.New("Not implemented")
}

//UpdatePost updates a post with new information
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UpdatePost(postID int, email string, subject string, message string, messageRaw string) error {
	return errors.New("Not implemented")
}

//DeleteFilesFromPost deletes all files belonging to a given post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Should be implemented to delete files individually
func DeleteFilesFromPost(postID int) error {

	// fileName = fileName[:strings.Index(fileName, ".")]
	// fileType = fileName[strings.Index(fileName, ".")+1:]
	// if fileType == "gif" || fileType == "webm" {
	// 	thumbType = "jpg"
	// }

	// os.Remove(path.Join(config.DocumentRoot, board, "/src/"+fileName+"."+fileType))
	// os.Remove(path.Join(config.DocumentRoot, board, "/thumb/"+fileName+"t."+thumbType))
	// os.Remove(path.Join(config.DocumentRoot, board, "/thumb/"+fileName+"c."+thumbType))
	return errors.New("Not implemented")
}

//DeletePost deletes a post with a given ID
func DeletePost(postID int) error {
	DeleteFilesFromPost(postID)
	//Also delete child posts if its a top post
	return errors.New("Not implemented")
}

//CreateDefaultBoardIfNoneExist creates a default board if no boards exist yet
func CreateDefaultBoardIfNoneExist() error {
	return errors.New("Not implemented")
	// firstBoard := Board{
	// 	Dir:         "test",
	// 	Title:       "Testing board",
	// 	Subtitle:    "Board for testing",
	// 	Description: "Board for testing",
	// 	Section:     1}
	// firstBoard.SetDefaults()
	// firstBoard.Build(true, true)
}

//CreateDefaultAdminIfNoStaff creates a new default admin account if no accounts exist
func CreateDefaultAdminIfNoStaff() error {
	return errors.New("Not implemented")
	// if _, err = execSQL(
	// 	"INSERT INTO DBPREFIXstaff (username,password_checksum,rank) VALUES(?,?,?)",
	// 	"admin", bcryptSum("password"), 3,
	// ); err != nil {
	// 	gclog.Print(lErrorLog|lStdLog|lFatal, "Failed creating admin user with error: ", err.Error())
	// }
}

//UpdateID takes a board struct and sets the database id according to the dir that is already set
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. (Just bad design in general, try to avoid directly mutating state like this)
func (board *Board) UpdateID() error {
	return errors.New("Not implemented")
	// return queryRowSQL("SELECT id FROM DBPREFIXboards WHERE dir = ?",
	// 	[]interface{}{board.Dir},
	// 	[]interface{}{&board.ID})
}

// PopulateData gets the board data from the database, according to its id, and sets the respective properties.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func (board *Board) PopulateData(id int) error {
	// queryStr := "SELECT * FROM DBPREFIXboards WHERE id = ?"
	// var values []interface{}
	// values = append(values, id)

	// return queryRowSQL(queryStr, values, []interface{}{
	// 	&board.ID, &board.ListOrder, &board.Dir, &board.Type, &board.UploadType,
	// 	&board.Title, &board.Subtitle, &board.Description, &board.Section,
	// 	&board.MaxFilesize, &board.MaxPages, &board.DefaultStyle, &board.Locked,
	// 	&board.CreatedOn, &board.Anonymous, &board.ForcedAnon, &board.MaxAge,
	// 	&board.AutosageAfter, &board.NoImagesAfter, &board.MaxMessageLength,
	// 	&board.EmbedsAllowed, &board.RedirectToThread, &board.RequireFile,
	// 	&board.EnableCatalog})
	return errors.New("Not implemented")
}

//DoesBoardExistByID returns a bool indicating whether a board with a given id exists
func DoesBoardExistByID(ID int) (bool, error) {
	return false, errors.New("Not implemented")
}

//GetAllBoards gets a list of all existing boards
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBoards() ([]Board, error) {
	return nil, errors.New("Not implemented")
}

//GetBoardFromID returns the board corresponding to a given id
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetBoardFromID(boardID int) (Board, error) {
	return Board{}, errors.New("Not implemented")
}

func getBoardIDFromURI(URI string) (int, error) {
	return -1, errors.New("Not implemented")
}
