package gcsql

import (
	"database/sql"
	"errors"
	"strconv"
)

var abstractSelectPosts = `
/*
Left join singlefiles on recentposts where recentposts.selfid = singlefiles.post_id
Coalesce filenames to "" (if filename = null -> "" else filename)


*/

Select 
	recentposts.selfid AS id,
	recentposts.toppostid AS parentid,
	recentposts.boardid,
	recentposts.name,
	recentposts.tripcode,
	recentposts.email,
	recentposts.subject,
	recentposts.message,
	recentposts.message_raw,
	recentposts.password,
	COALESCE(singlefiles.filename, '') as filename,
	singlefiles.original_filename,
	singlefiles.checksum,
	singlefiles.file_size,
	singlefiles.width as image_w,
	singlefiles.height as image_h,
	singlefiles.thumbnail_width as thumb_w,
	singlefiles.thumbnail_height as thumb_h,
	recentposts.ip,
	recentposts.created_on,
	recentposts.anchored,
	recentposts.last_bump,
	recentposts.stickied,
	recentposts.locked
FROM
	(SELECT 
		posts.id AS selfid,
		COALESCE(NULLIF(topposts.id, posts.id), 0) AS toppostid,
		boards.id AS boardid,
		posts.name,
		posts.ip,
		posts.tripcode,
		posts.message,
		posts.email,
		posts.subject,
		posts.message_raw,
		posts.password,
		posts.created_on,
		posts.is_top_post,
		threads.anchored,
		threads.last_bump,
		threads.stickied,
		threads.locked
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
	(SELECT files.*
	FROM DBPREFIXfiles as files
	JOIN 
		(SELECT post_id, min(file_order) as file_order
		FROM DBPREFIXfiles
		GROUP BY post_id) as topfiles 
		ON files.post_id = topfiles.post_id AND files.file_order = topfiles.file_order
	) AS singlefiles 
	
	ON recentposts.selfid = singlefiles.post_id`

// getPostsExcecution excecutes a given variation on abstractSelectPosts with parameters and loads the result into an array of posts
func getPostsExcecution(sql string, arguments ...interface{}) ([]Post, error) {
	rows, err := QuerySQL(sql, arguments...)
	if err != nil {
		return nil, err
	}
	var posts []Post
	for rows.Next() {
		post := new(Post)
		err = rows.Scan(&post.ID, &post.ParentID, &post.BoardID, &post.Name, &post.Tripcode, &post.Email,
			&post.Subject, &post.MessageHTML, &post.MessageText, &post.Password, &post.Filename,
			&post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW, &post.ImageH,
			&post.ThumbW, &post.ThumbH, &post.IP, &post.Timestamp, &post.Autosage, &post.Bumped, &post.Stickied, &post.Locked)
		if err != nil {
			return nil, err
		}
		post.FileExt = "placeholder"
		posts = append(posts, *post)
	}
	return posts, nil
}

var onlyTopPosts = abstractSelectPosts + "\nWHERE recentposts.is_top_post"
var sortedTopPosts = onlyTopPosts + "\nORDER BY recentposts.last_bump DESC"

// GetTopPostsNoSort gets the thread ops for a given board.
// Results are unsorted
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetTopPostsNoSort(boardID int) (posts []Post, err error) {
	return getPostsExcecution(onlyTopPosts)
}

// GetTopPosts gets the thread ops for a given board.
// newestFirst sorts the ops by the newest first if true, by newest last if false
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetTopPosts(boardID int, newestFirst bool) (posts []Post, err error) {
	return getPostsExcecution(sortedTopPosts)
}

var repliesToX = abstractSelectPosts + "\nWHERE recentposts.toppostid = ?"
var oldestRepliesFirst = repliesToX + "\nORDER BY recentposts.created_on DESC"
var newestFirstLimited = repliesToX + "\nORDER BY recentposts.created_on DESC\nLIMIT ?"

// GetExistingReplies gets all the reply posts to a given thread, ordered by oldest first.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetExistingReplies(topPost int) (posts []Post, err error) {
	return getPostsExcecution(oldestRepliesFirst, topPost)
}

// GetExistingRepliesLimitedRev gets N amount of reply posts to a given thread, ordered by newest first.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetExistingRepliesLimitedRev(topPost int, limit int) (posts []Post, err error) {
	return getPostsExcecution(newestFirstLimited, topPost, limit)
}

//Toppost: where a post with a given id has this as their top post

// GetSpecificTopPost gets the information for the top post for a given id.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificTopPost(ID int) (Post, error) {
	const topPostIDQuery = `SELECT posts.id from DBPREFIXposts as posts
	JOIN (
		SELECT threads.id from DBPREFIXthreads as threads
		JOIN DBPREFIXposts as posts
		ON posts.thread_id = threads.id
		WHERE posts.id = ?
	) as thread
	ON posts.thread_id = thread.id
	WHERE posts.is_top_post`
	//get top post of item with given id
	var FoundID int
	err := QueryRowSQL(topPostIDQuery, interfaceSlice(ID), interfaceSlice(&FoundID))
	if err != nil {
		return Post{}, err
	}
	return GetSpecificPost(FoundID, false)
}

// GetSpecificPostByString gets a specific post for a given string id.
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificPostByString(ID string) (post Post, err error) {
	return getSpecificPostStringDecorated(ID, false)
}

// GetSpecificPost gets a specific post for a given id.
// returns SQL.ErrNoRows if no post could be found
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetSpecificPost(ID int, onlyNotDeleted bool) (post Post, err error) {
	return getSpecificPostStringDecorated(strconv.Itoa(ID), onlyNotDeleted)
}

var specificPostSQL = abstractSelectPosts + "\nWHERE recentposts.selfid = ?"
var specificPostSQLNotDeleted = specificPostSQL + "\nWHERE recentposts.is_deleted = FALSE"

func getSpecificPostStringDecorated(ID string, onlyNotDeleted bool) (Post, error) {
	var sql string
	if onlyNotDeleted {
		sql = specificPostSQL
	} else {
		sql = specificPostSQLNotDeleted
	}
	posts, err := getPostsExcecution(sql, ID)
	if err != nil {
		return Post{}, err
	}
	if len(posts) == 0 {
		return Post{}, errors.New("Could not find a post with the ID: " + ID)
	}
	return posts[0], nil
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
		recentQueryStr += "\n" + `WHERE singlefiles.filename IS NOT NULL AND recentposts.boardid = ?
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, boardID, amount)
	}
	if onlyWithFile && !onSpecificBoard {
		recentQueryStr += "\n" + `WHERE singlefiles.filename IS NOT NULL
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, amount)
	}
	if !onlyWithFile && onSpecificBoard {
		recentQueryStr += "\n" + `WHERE recentposts.boardid = ?
		ORDER BY recentposts.created_on DESC LIMIT ?`
		rows, err = QuerySQL(recentQueryStr, boardID, amount)
	}
	if !onlyWithFile && !onSpecificBoard {
		recentQueryStr += "\nORDER BY recentposts.created_on DESC LIMIT ?"
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
