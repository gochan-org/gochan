package main

import (
	"database/sql"
	"errors"
	"net"
	"strconv"
	"time"
)

//ErrNotImplemented is a not implemented exception
var ErrNotImplemented = errors.New("Not implemented")

// GetTopPostsNoSort gets the thread ops for a given board.
// Results are unsorted
func GetTopPostsNoSort(boardID int) (posts []Post, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// GetTopPosts gets the thread ops for a given board.
// newestFirst sorts the ops by the newest first if true, by newest last if false
func GetTopPosts(boardID int, newestFirst bool) (posts []Post, err error) {
	//TODO sort by bump
	return nil, ErrNotImplemented
}

// GetExistingReplies gets all the reply posts to a given thread, ordered by oldest first.
func GetExistingReplies(topPost int) (posts []Post, err error) {
	//TODO sort by number/date
	return nil, ErrNotImplemented
}

// GetExistingRepliesLimitedRev gets N amount of reply posts to a given thread, ordered by newest first.
func GetExistingRepliesLimitedRev(topPost int, limit int) (posts []Post, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// GetSpecificTopPost gets the information for the top post for a given id.
func GetSpecificTopPost(ID int) (posts Post, err error) {
	//Currently implemented as GetSpecificPost because getSpecificPost can also be a top post.
	return GetSpecificPost(ID, false)
}

// GetSpecificPostByString gets a specific post for a given string id.
func GetSpecificPostByString(ID string) (post Post, err error) {
	//TODO
	return Post{}, ErrNotImplemented
}

// GetSpecificPost gets a specific post for a given id.
// returns SQL.ErrNoRows if no post could be found
func GetSpecificPost(ID int, onlyNotDeleted bool) (post Post, err error) {
	//TODO
	return Post{}, ErrNotImplemented
}

// GetAllNondeletedMessageRaw gets all the raw message texts from the database, saved per id
func GetAllNondeletedMessageRaw() (messages []MessagePostContainer, err error) {
	//TODO
	return nil, ErrNotImplemented
}

// SetMessages sets all the non-raw text for a given array of items.
func SetMessages(messages []MessagePostContainer) (err error) {
	//TODO
	return ErrNotImplemented
}

// GetRecentPostsOnBoard returns the most recent N posts on a specific board
func GetRecentPostsOnBoard(amount int, boardID int) ([]RecentPost, error) {
	return getRecentPostsInternal(amount, false, boardID, true)
}

// getRecentPostsInternal returns the most recent N posts, on a specific board if specified, only with files if specified
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
		rows, err = querySQL(recentQueryStr, boardID, amount)
	}
	if onlyWithFile && !onSpecificBoard {
		recentQueryStr += `\nWHERE singlefiles.filename IS NOT NULL
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = querySQL(recentQueryStr, amount)
	}
	if !onlyWithFile && onSpecificBoard {
		recentQueryStr += `\nWHERE recentposts.boardid = ?
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = querySQL(recentQueryStr, boardID, amount)
	}
	if !onlyWithFile && !onSpecificBoard {
		recentQueryStr += `\nORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = querySQL(recentQueryStr, amount)
	}

	defer closeHandle(rows)
	if err != nil {
		return nil, err
	}

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

// GetStaffData returns the data associated with a given username
func GetStaffData(staffName string) (data string, err error) {
	//("SELECT sessiondata FROM DBPREFIXsessions WHERE name = ?",
	return "DUMMY", ErrNotImplemented
}

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (name string, err error) {
	//after refactor, check if still used
	return "DUMMY", ErrNotImplemented
}

func GetStaffBySession(session string) (*Staff, error) { //TODO not upt to date with old db yet
	// staff := new(Staff)
	// err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
	// 	[]interface{}{name},
	// 	[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	// )
	// return staff, err

	return nil, ErrNotImplemented
}

func GetStaffByName(name string) (*Staff, error) { //TODO not upt to date with old db yet
	// staff := new(Staff)
	// err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
	// 	[]interface{}{name},
	// 	[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	// )
	// return staff, err

	return nil, ErrNotImplemented
}

func newStaff(username string, password string, rank int) error { //TODO not up to date with old db yet
	// _, err := execSQL("INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES(?,?,?)",
	// 	&username, bcryptSum(password), &rank)
	// return err
	return ErrNotImplemented
}

func deleteStaff(username string) error { //TODO not up to date with old db yet
	// _, err := execSQL("DELETE FROM DBPREFIXstaff WHERE username = ?", username)
	// return err
	return ErrNotImplemented
}

func CreateSession(key string, username string) error { //TODO not up to date with old db yet
	//TODO move amount of time to config file
	//TODO also set last login
	// return execSQL("INSERT INTO DBPREFIXsessions (name,sessiondata,expires) VALUES(?,?,?)",
	// 	key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
	// )
	return ErrNotImplemented
}

func PermanentlyRemoveDeletedPosts() error {
	//Remove all deleted posts
	//Remove orphaned threads
	//Make sure cascades are set up properly
	return ErrNotImplemented
}

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
func UserBan(IP net.IP, threadBan bool, staffName string, boardURI string, expires time.Time, permaban bool,
	staffNote string, message string, canAppeal bool, appealAt time.Time) error {
	return ErrNotImplemented
}

func GetStaffRankAndBoards(username string) (rank int, boardUris []string, err error) {

	return 420, nil, ErrNotImplemented
}

//GetAllAccouncements gets all announcements, newest first
func GetAllAccouncements() ([]Announcement, error) {
	//("SELECT subject,message,poster,timestamp FROM DBPREFIXannouncements ORDER BY id DESC")
	//rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)

	return nil, ErrNotImplemented
}

func CreateBoard(values Board) error {
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
}

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

// GetAllSectionsOrCreateDefault gets all sections in the database, creates default if none exist
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

func GetAllStaffNopass() ([]Staff, error) {

	return nil, ErrNotImplemented
}

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

//HackyStringToInt parses a string to an int, or 0 if error
func HackyStringToInt(text string) int {
	value, _ := strconv.Atoi(text)
	return value
}

// BumpThread the given thread on the given board and returns true if there were no errors
func BumpThread(postID, boardID int) error { //NOT UP TO DATE
	// _, err := execSQL("UPDATE DBPREFIXposts SET bumped = ? WHERE id = ? AND boardid = ?",
	// 	time.Now(), postID, boardID,
	// )

	return ErrNotImplemented
}

//CheckBan returns banentry if a ban was found or a sql.ErrNoRows if not banned
// name, filename and checksum may be empty strings and will be treated as not requested is done so
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

func GetEmbedsAllowed(boardID int) (bool, error) {

	return false, ErrNotImplemented
}

func GetBoardFromPostID(postID int) (boardURI string, err error) {
	return "", ErrNotImplemented
}

func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	return 0, ErrNotImplemented
}

func AddBanAppeal(banID uint, message string) error {
	return ErrNotImplemented
}

func GetPostPassword(postID int) (password string, err error) {
	return "", ErrNotImplemented
}

func UpdatePost(postID int, email string, subject string, message string, message_raw string) error {
	return ErrNotImplemented
}

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

func DeletePost(postID int) error {
	DeleteFilesFromPost(postID)
	//Also delete child posts if its a top post
	return ErrNotImplemented
}
