package gcsql

import (
	"errors"
	"net"
	"strings"
	"time"
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
	updateCount := len(messages)
	sqlToRun := strings.Repeat(sql, updateCount)

	interfaceSlice := make([]interface{}, 2*updateCount) //put all ids + message in one array, in pairs
	for _, message := range messages {
		interfaceSlice = append(interfaceSlice, message.Message, message.ID)
	}

	_, err := ExecSQL(sqlToRun, interfaceSlice...) //TODO disable cache on this execution
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
func GetReplyFileCount(postID int) (fileCount int, err error) {
	return 420, errors.New("Not implemented")
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (name string, err error) {
	//after refactor, check if still used
	return "DUMMY", errors.New("Not implemented")
}

// GetStaffBySession gets the staff that is logged in in the given session
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffBySession(session string) (*Staff, error) { //TODO not upt to date with old db yet
	// staff := new(Staff)
	// err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
	// 	[]interface{}{name},
	// 	[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	// )
	// return staff, err

	return nil, errors.New("Not implemented")
}

// GetStaffByName gets the staff with a given name
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffByName(name string) (*Staff, error) { //TODO not upt to date with old db yet
	// staff := new(Staff)
	// err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
	// 	[]interface{}{name},
	// 	[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	// )
	// return staff, err

	return nil, errors.New("Not implemented")
}

// NewStaff creates a new staff account from a given username, password and rank
func NewStaff(username string, password string, rank int) error { //TODO not up to date with old db yet
	// _, err := execSQL("INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES(?,?,?)",
	// 	&username, bcryptSum(password), &rank)
	// return err
	return errors.New("Not implemented")
}

// DeleteStaff deletes the staff with a given name.
func DeleteStaff(username string) error { //TODO not up to date with old db yet
	// _, err := execSQL("DELETE FROM DBPREFIXstaff WHERE username = ?", username)
	// return err
	return errors.New("Not implemented")
}

// CreateSession inserts a session for a given key and username into the database
func CreateSession(key string, username string) error { //TODO not up to date with old db yet
	//TODO move amount of time to config file
	//TODO also set last login
	// return execSQL("INSERT INTO DBPREFIXsessions (name,sessiondata,expires) VALUES(?,?,?)",
	// 	key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
	// )
	return errors.New("Not implemented")
}

// PermanentlyRemoveDeletedPosts removes all posts and files marked as deleted from the database
func PermanentlyRemoveDeletedPosts() error {
	//Remove all deleted posts
	//Remove orphaned threads
	//Make sure cascades are set up properly
	return errors.New("Not implemented")
}

// OptimizeDatabase peforms a database optimisation
func OptimizeDatabase() error { //TODO FIX, try to do it entirely within one SQL transaction

	// html += "Optimizing all tables in database.<hr />"
	// tableRows, tablesErr := querySQL("SHOW TABLES")
	// defer closeHandle(tableRows)

	// if tablesErr != nil && tablesErr != sql.ErrNoRows {
	// 	return html + "<tr><td>" +
	// 		gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
	// 		"</td></tr></table>"
	// }
	// for tableRows.Next() {
	// 	var table string
	// 	tableRows.Scan(&table)
	// 	if _, err := execSQL("OPTIMIZE TABLE " + table); err != nil {
	// 		return html + "<tr><td>" +
	// 			gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
	// 			"</td></tr></table>"
	// 	}
	// }
	return errors.New("Not implemented")
}

// FileBan creates a new ban on a file. If boards = nil, the ban is global.
func FileBan(fileChecksum string, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return errors.New("Not implemented")
}

// FileNameBan creates a new ban on a filename. If boards = nil, the ban is global.
func FileNameBan(fileName string, isRegex bool, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return errors.New("Not implemented")
}

// UserNameBan creates a new ban on a username. If boards = nil, the ban is global.
func UserNameBan(userName string, isRegex bool, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return errors.New("Not implemented")
}

// UserBan creates either a full ip ban, or an ip ban for threads only, for a given IP.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UserBan(IP net.IP, threadBan bool, staffName string, boardURI string, expires time.Time, permaban bool,
	staffNote string, message string, canAppeal bool, appealAt time.Time) error {
	return errors.New("Not implemented")
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
