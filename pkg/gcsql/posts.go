package gcsql

import (
	"database/sql"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// SinceLastPost returns the number of seconds since the given IP address created a post
// (used for checking against the new thread/new reply cooldown)
func SinceLastPost(postIP string) (int, error) {
	const sql = `SELECT MAX(created_on) FROM DBPREFIXposts WHERE ip = ?`
	var when time.Time
	err := QueryRowSQL(sql, interfaceSlice(postIP), interfaceSlice(&when))
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

const selectPostsQuery = `SELECT
	DBPREFIXposts.id,
	thread_id AS threadid,
	(SELECT id FROM DBPREFIXposts WHERE is_top_post = TRUE AND thread_id = threadid LIMIT 1),
	(SELECT board_id FROM DBPREFIXthreads WHERE id = DBPREFIXposts.thread_id) as board_id,
	ip,created_on,name,tripcode,email,subject,message,message_raw,password,

	COALESCE(files.filename,''),
	COALESCE(files.original_filename,''),
	COALESCE(files.file_size, 0),
	COALESCE(files.checksum, ''),
	COALESCE(files.thumbnail_width, 0),
	COALESCE(files.thumbnail_height, 0),
	COALESCE(files.width, 0),
	COALESCE(files.height, 0)
		FROM DBPREFIXposts LEFT OUTER JOIN DBPREFIXfiles AS files ON files.post_id = DBPREFIXposts.id`

// GetPostFromID gets the post from the database with a matching ID,
// optionally requiring it to not be deleted
func GetPostFromID(id int, onlyNotDeleted bool) (*Post, error) {
	sql := selectPostsQuery + ` WHERE DBPREFIXposts.id = ?`
	if onlyNotDeleted {
		sql += " AND is_deleted = 0"
	}
	var drop int
	post := new(Post)
	err := QueryRowSQL(sql, []interface{}{id}, []interface{}{
		&post.ID, &drop, &post.ParentID,
		&post.BoardID,
		&post.IP, &post.Timestamp, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password,
		&post.Filename, &post.FilenameOriginal, &post.Filesize, &post.FileChecksum, &post.ThumbW, &post.ThumbH, &post.ImageW, &post.ImageH,
	})
	if err != nil {
		return nil, err
	}
	return post, err
}

// GetPostsFromIP gets the posts from the database with a matching IP address, specifying
// optionally requiring them to not be deleted
func GetPostsFromIP(ip string, limit int, onlyNotDeleted bool) ([]Post, error) {
	sql := selectPostsQuery + ` WHERE DBPREFIXposts.ip = ?`
	if onlyNotDeleted {
		sql += " AND is_deleted = 0"
	}

	sql += " ORDER BY id DESC LIMIT ?"
	rows, err := QuerySQL(sql, ip, limit)
	if err != nil {
		return nil, err
	}
	var posts []Post
	for rows.Next() {
		var post Post
		var drop int
		if err = rows.Scan(
			&post.ID, &drop, &post.ParentID,
			&post.BoardID,
			&post.IP, &post.Timestamp, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password,
			&post.Filename, &post.FilenameOriginal, &post.Filesize, &post.FileChecksum, &post.ThumbW, &post.ThumbH, &post.ImageW, &post.ImageH,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// GetFilePaths returns an array of absolute paths to uploaded files and thumbnails associated
// with this post, and any errors that occurred
func (p *Post) GetFilePaths() ([]string, error) {
	boardDir, err := p.GetBoardDir()
	if err != nil {
		return nil, err
	}
	const filenameSQL = `SELECT filename FROM DBPREFIXfiles WHERE post_id = ?`
	rows, err := QuerySQL(filenameSQL, p.ID)
	var paths []string
	if err == sql.ErrNoRows {
		return paths, nil
	} else if err != nil {
		return nil, err
	}
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	for rows.Next() {
		var filename string
		if err = rows.Scan(&filename); err != nil {
			return paths, err
		}
		_, filenameBase, fileExt := gcutil.GetFileParts(filename)
		thumbExt := fileExt
		if thumbExt == "gif" || thumbExt == "webm" || thumbExt == "mp4" {
			thumbExt = "jpg"
		}
		paths = append(paths,
			path.Join(documentRoot, boardDir, "/src/", filenameBase+"."+fileExt),
			path.Join(documentRoot, boardDir, "/thumb/", filenameBase+"t."+thumbExt), // thumbnail path
		)
		if p.ParentID == 0 {
			paths = append(paths, path.Join(documentRoot, boardDir, "/thumb/", filenameBase+"c."+thumbExt)) // catalog thumbnail path
		}
	}
	return paths, nil
}

// UnlinkUploads disassociates the post with any uploads in DBPREFIXfiles
// that may have been uploaded with it, optionally leaving behind a "File Deleted"
// frame where the thumbnail appeared
func (p *Post) UnlinkUploads(leaveDeletedBox bool) error {
	var sqlStr string
	if leaveDeletedBox {
		// leave a "File Deleted" box
		sqlStr = `UPDATE DBPREFIXfiles SET filename = 'deleted', original_filename = 'deleted' WHERE post_id = ?`
	} else {
		sqlStr = `DELETE FROM DBPREFIXfiles WHERE post_id = ?`
	}
	_, err := ExecSQL(sqlStr, p.ID)
	return err
}
