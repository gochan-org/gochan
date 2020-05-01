package gcsql

import (
	"database/sql"
	"errors"
	"net"
	"time"
)

//ErrNotImplemented is a not implemented exception
var ErrNotImplemented = errors.New("Not implemented")

// GetTopPostsNoSort gets the thread ops for a given board.
// Results are unsorted
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetTopPostsNoSort(boardID int) (posts []Post, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// GetTopPosts gets the thread ops for a given board.
// newestFirst sorts the ops by the newest first if true, by newest last if false
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetTopPosts(boardID int, newestFirst bool) (posts []Post, err error) {
	//TODO sort by bump
	return nil, ErrNotImplemented
}

// GetExistingReplies gets all the reply posts to a given thread, ordered by oldest first.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetExistingReplies(topPost int) (posts []Post, err error) {
	//TODO sort by number/date
	return nil, ErrNotImplemented
}

// GetExistingRepliesLimitedRev gets N amount of reply posts to a given thread, ordered by newest first.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetExistingRepliesLimitedRev(topPost int, limit int) (posts []Post, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// GetSpecificTopPost gets the information for the top post for a given id.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificTopPost(ID int) (posts Post, err error) {
	//Currently implemented as GetSpecificPost because getSpecificPost can also be a top post.
	return GetSpecificPost(ID, false)
}

// GetSpecificPostByString gets a specific post for a given string id.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificPostByString(ID string) (post Post, err error) {
	//TODO
	return Post{}, ErrNotImplemented
}

// GetSpecificPost gets a specific post for a given id.
// returns SQL.ErrNoRows if no post could be found
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificPost(ID int, onlyNotDeleted bool) (post Post, err error) {
	//TODO
	return Post{}, ErrNotImplemented
}

// GetAllNondeletedMessageRaw gets all the raw message texts from the database, saved per id
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllNondeletedMessageRaw() (messages []MessagePostContainer, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// SetMessages sets all the non-raw text for a given array of items.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func SetMessages(messages []MessagePostContainer) (err error) {
	//TODO
	return ErrNotImplemented
}

// getRecentPostsInternal returns the most recent N posts, on a specific board if specified, only with files if specified
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func getRecentPostsInternal(amount int, onlyWithFile bool, boardID int, onSpecificBoard bool) ([]RecentPost, error) {
	//TODO: rework so it uses all features/better sql
	//get recent posts
	recentQueryStr := `
	/*
	recentposts = join all non-deleted posts with the post id of their thread and the board it belongs on, sort by date and grab top x posts
	singlefiles = the top file per post id
	
	Left join singlefiles on recentposts where recentposts.selfid = singlefiles.post_id
	Coalesce filenames to "" (if filename = null -> "" else filename)
	
	Query might benefit from [filter on posts with at least one file -> ] filter N most recent -> manually loop N results for file/board/parentthreadid
	*/
	
	Select 
		recentposts.selfid AS id,
		recentposts.toppostid AS parentid,
		recentposts.boardname,
		recentposts.boardid,
		recentposts.name,
		recentposts.tripcode,
		recentposts.message,
		COALESCE(singlefiles.filename, '') as filename,
		singlefiles.thumbnail_width as thumb_w,
		singlefiles.thumbnail_height as thumb_h
	FROM
		(SELECT 
			posts.id AS selfid,
			topposts.id AS toppostid,
			boards.dir AS boardname,
			boards.id AS boardid,
			posts.name,
			posts.tripcode,
			posts.message,
			posts.email,
			 posts.created_on
		FROM
			DBPREFIXposts AS posts
		JOIN DBPREFIXthreads AS threads 
			ON threads.id = posts.thread_id
		JOIN DBPREFIXposts AS topposts 
			ON threads.id = topposts.thread_id
		JOIN DBPREFIXboards AS boards
			ON threads.board_id = boards.id
		WHERE 
			topposts.is_top_post = TRUE AND posts.is_deleted = FALSE
		
		) as recentposts
	LEFT JOIN 
		(SELECT files.post_id, filename, files.thumbnail_width, files.thumbnail_height
		FROM DBPREFIXfiles as files
		JOIN 
			(SELECT post_id, min(file_order) as file_order
			FROM DBPREFIXfiles
			GROUP BY post_id) as topfiles 
			ON files.post_id = topfiles.post_id AND files.file_order = topfiles.file_order
		) AS singlefiles 
		
		ON recentposts.selfid = singlefiles.post_id`
	var rows *sql.Rows
	var err error

	if onlyWithFile && onSpecificBoard {
		recentQueryStr += `\nWHERE singlefiles.filename IS NOT NULL AND recentposts.boardid = ?
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, boardID, amount)
	}
	if onlyWithFile && !onSpecificBoard {
		recentQueryStr += `\nWHERE singlefiles.filename IS NOT NULL
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, amount)
	}
	if !onlyWithFile && onSpecificBoard {
		recentQueryStr += `\nWHERE recentposts.boardid = ?
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, boardID, amount)
	}
	if !onlyWithFile && !onSpecificBoard {
		recentQueryStr += `\nORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, amount)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var recentPostsArr []RecentPost

	for rows.Next() {
		recentPost := new(RecentPost)
		if err = rows.Scan(
			&recentPost.PostID, &recentPost.ParentID, &recentPost.BoardName, &recentPost.BoardID,
			&recentPost.Name, &recentPost.Tripcode, &recentPost.Message, &recentPost.Filename, &recentPost.ThumbW, &recentPost.ThumbH,
		); err != nil {
			return nil, err
		}
		recentPostsArr = append(recentPostsArr, *recentPost)
	}

	return recentPostsArr, nil
}

// GetRecentPostsGlobal returns the global N most recent posts from the database.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetRecentPostsGlobal(amount int, onlyWithFile bool) ([]RecentPost, error) {
	return getRecentPostsInternal(amount, onlyWithFile, 0, false)
}

// GetReplyCount gets the total amount non-deleted of replies in a thread
func GetReplyCount(postID int) (replyCount int, err error) {
	return 420, ErrNotImplemented
}

// GetReplyFileCount gets the amount of files non-deleted posted in total in a thread
func GetReplyFileCount(postID int) (fileCount int, err error) {
	return 420, ErrNotImplemented
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (name string, err error) {
	//after refactor, check if still used
	return "DUMMY", ErrNotImplemented
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

	return nil, ErrNotImplemented
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

	return nil, ErrNotImplemented
}

// NewStaff creates a new staff account from a given username, password and rank
func NewStaff(username string, password string, rank int) error { //TODO not up to date with old db yet
	// _, err := execSQL("INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES(?,?,?)",
	// 	&username, bcryptSum(password), &rank)
	// return err
	return ErrNotImplemented
}

// DeleteStaff deletes the staff with a given name.
func DeleteStaff(username string) error { //TODO not up to date with old db yet
	// _, err := execSQL("DELETE FROM DBPREFIXstaff WHERE username = ?", username)
	// return err
	return ErrNotImplemented
}

// CreateSession inserts a session for a given key and username into the database
func CreateSession(key string, username string) error { //TODO not up to date with old db yet
	//TODO move amount of time to config file
	//TODO also set last login
	// return execSQL("INSERT INTO DBPREFIXsessions (name,sessiondata,expires) VALUES(?,?,?)",
	// 	key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
	// )
	return ErrNotImplemented
}

// PermanentlyRemoveDeletedPosts removes all posts and files marked as deleted from the database
func PermanentlyRemoveDeletedPosts() error {
	//Remove all deleted posts
	//Remove orphaned threads
	//Make sure cascades are set up properly
	return ErrNotImplemented
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
	return ErrNotImplemented
}

// FileBan creates a new ban on a file. If boards = nil, the ban is global.
func FileBan(fileChecksum string, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return ErrNotImplemented
}

// FileNameBan creates a new ban on a filename. If boards = nil, the ban is global.
func FileNameBan(fileName string, isRegex bool, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return ErrNotImplemented
}

// UserNameBan creates a new ban on a username. If boards = nil, the ban is global.
func UserNameBan(userName string, isRegex bool, staffName string, expires time.Time, permaban bool, staffNote string, boardURI string) error {
	return ErrNotImplemented
}

// UserBan creates either a full ip ban, or an ip ban for threads only, for a given IP.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UserBan(IP net.IP, threadBan bool, staffName string, boardURI string, expires time.Time, permaban bool,
	staffNote string, message string, canAppeal bool, appealAt time.Time) error {
	return ErrNotImplemented
}

//GetAllAccouncements gets all announcements, newest first
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllAccouncements() ([]Announcement, error) {
	//("SELECT subject,message,poster,timestamp FROM DBPREFIXannouncements ORDER BY id DESC")
	//rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)

	return nil, ErrNotImplemented
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
	return ErrNotImplemented
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
	return nil, ErrNotImplemented
}

//GetAllSections gets a list of all existing sections
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllSections() (sections []BoardSection, err error) {
	return nil, ErrNotImplemented
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
	return nil, ErrNotImplemented
}

//CreateDefaultSectionIfNotExist creates the default section if it does not exist yet
func CreateDefaultSectionIfNotExist() error {
	return ErrNotImplemented
}

//GetAllStaffNopass gets all staff accounts without their password
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllStaffNopass() ([]Staff, error) {

	return nil, ErrNotImplemented
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
	return nil, ErrNotImplemented
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
	return nil, ErrNotImplemented
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
	return ErrNotImplemented
}

//GetMaxMessageLength returns the max message length on a board
func GetMaxMessageLength(boardID int) (int, error) {

	return 0, ErrNotImplemented
}

//GetEmbedsAllowed returns if embeds are allowed on a given board
func GetEmbedsAllowed(boardID int) (bool, error) {

	return false, ErrNotImplemented
}

//GetBoardFromPostID gets the boardURI that a given postid exists on
func GetBoardFromPostID(postID int) (boardURI string, err error) {
	return "", ErrNotImplemented
}

//GetThreadIDZeroIfTopPost gets the post id of the top post of the thread a post belongs to, zero if the post itself is the top post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Posts do not directly reference their post post anymore.
func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	return 0, ErrNotImplemented
}

//AddBanAppeal adds a given appeal to a given ban
func AddBanAppeal(banID uint, message string) error {
	return ErrNotImplemented
}

//GetPostPassword gets the password associated with a given post
func GetPostPassword(postID int) (password string, err error) {
	return "", ErrNotImplemented
}

//UpdatePost updates a post with new information
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UpdatePost(postID int, email string, subject string, message string, messageRaw string) error {
	return ErrNotImplemented
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
	return ErrNotImplemented
}

//DeletePost deletes a post with a given ID
func DeletePost(postID int) error {
	DeleteFilesFromPost(postID)
	//Also delete child posts if its a top post
	return ErrNotImplemented
}

//CreateDefaultBoardIfNoneExist creates a default board if no boards exist yet
func CreateDefaultBoardIfNoneExist() error {
	return ErrNotImplemented
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
	return ErrNotImplemented
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
	return ErrNotImplemented
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
	return ErrNotImplemented
}

//DoesBoardExistByID returns a bool indicating whether a board with a given id exists
func DoesBoardExistByID(ID int) (bool, error) {
	return false, ErrNotImplemented
}

//GetAllBoards gets a list of all existing boards
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllBoards() ([]Board, error) {
	return nil, ErrNotImplemented
}

//GetBoardFromID returns the board corresponding to a given id
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetBoardFromID(boardID int) (Board, error) {
	return Board{}, ErrNotImplemented
}
