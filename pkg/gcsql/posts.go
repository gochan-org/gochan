package gcsql

import "time"

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
