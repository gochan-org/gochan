package gcsql

import (
	"database/sql"
	"errors"
	"strconv"
)

const (
	selectThreadsBaseSQL = `SELECT
	id, board_id, locked, stickied, anchored, cyclical, last_bump, deleted_at, is_deleted
	FROM DBPREFIXthreads `
)

var (
	ErrThreadExists       = errors.New("thread already exists")
	ErrThreadDoesNotExist = errors.New("thread does not exist")
)

func createThread(tx *sql.Tx, boardID int, locked bool, stickied bool, anchored bool, cyclical bool) (threadID int, err error) {
	const insertQuery = `INSERT INTO DBPREFIXthreads (board_id, locked, stickied, anchored, cyclical) VALUES (?,?,?,?,?)`
	stmt, err := PrepareSQL(insertQuery, tx)
	if err != nil {
		return 0, err
	}
	if tx == nil {
		defer stmt.Close()
	}
	if _, err = stmt.Exec(boardID, locked, stickied, anchored, cyclical); err != nil {
		return 0, err
	}
	stmt2, err := PrepareSQL(`SELECT MAX(id) FROM DBPREFIXthreads`, tx)
	if err != nil {
		return 0, err
	}
	if tx == nil {
		defer stmt2.Close()
	}

	err = stmt2.QueryRow().Scan(&threadID)
	return threadID, err
}

// GetThread returns a a thread object from the database
func GetThread(threadID int) (*Thread, error) {
	const query = selectThreadsBaseSQL + `WHERE id = ?`
	thread := new(Thread)
	err := QueryRowSQL(query, interfaceSlice(threadID), interfaceSlice(
		&thread.ID, &thread.BoardID, &thread.Locked, &thread.Stickied, &thread.Anchored, &thread.Cyclical,
		&thread.LastBump, &thread.DeletedAt, &thread.IsDeleted,
	))
	return thread, err
}

// GetTopPostThreadID gets the thread ID from the database, given the post ID of a top post
func GetTopPostThreadID(opID int) (int, error) {
	const query = `SELECT thread_id FROM DBPREFIXposts WHERE id = ? and is_top_post`
	var threadID int
	err := QueryRowSQL(query, interfaceSlice(opID), interfaceSlice(&threadID))
	if err == sql.ErrNoRows {
		err = ErrThreadDoesNotExist
	}
	return threadID, err
}

// GetThreadsWithBoardID queries the database for the threads with the given board ID from the database.
// If onlyNotDeleted is true, it omits deleted threads and threads that were removed because the max
// thread limit was reached
func GetThreadsWithBoardID(boardID int, onlyNotDeleted bool) ([]Thread, error) {
	query := selectThreadsBaseSQL + `WHERE board_id = ?`
	if onlyNotDeleted {
		query += " AND  is_deleted = FALSE"
	}
	rows, err := QuerySQL(query, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var threads []Thread
	for rows.Next() {
		var thread Thread
		if err = rows.Scan(
			&thread.ID, &thread.BoardID, &thread.Locked, &thread.Stickied, &thread.Anchored,
			&thread.Cyclical, &thread.LastBump, &thread.DeletedAt, &thread.IsDeleted,
		); err != nil {
			return threads, err
		}
		threads = append(threads, thread)
	}
	return threads, nil
}

func GetThreadReplyCountFromOP(opID int) (int, error) {
	const query = `SELECT COUNT(*) FROM DBPREFIXposts WHERE thread_id = (
		SELECT thread_id FROM DBPREFIXposts WHERE id = ?) AND is_deleted = FALSE AND is_top_post = FALSE`
	var num int
	err := QueryRowSQL(query, interfaceSlice(opID), interfaceSlice(&num))
	return num, err
}

// ChangeThreadBoardID updates the given thread's post ID and the destination board ID
func ChangeThreadBoardID(threadID int, newBoardID int) error {
	if !DoesBoardExistByID(newBoardID) {
		return ErrBoardDoesNotExist
	}
	_, err := ExecSQL(`UPDATE DBPREFIXthreads SET board_id = ? WHERE id = ?`, newBoardID, threadID)
	return err
}

// ChangeThreadBoardByURI updates a thread's board ID, given the thread's post ID and
// the destination board's uri
func ChangeThreadBoardByURI(postID int, uri string) error {
	boardID, err := getBoardIDFromURI(uri)
	if err != nil {
		return err
	}
	return ChangeThreadBoardID(postID, boardID)
}

func (t *Thread) GetBoard() (*Board, error) {
	return GetBoardFromID(t.BoardID)
}

func (t *Thread) GetReplyFileCount() (int, error) {
	const query = `SELECT COUNT(filename) FROM DBPREFIXfiles WHERE post_id IN (
		SELECT id FROM DBPREFIXposts WHERE thread_id = ? AND is_deleted = FALSE)`
	var fileCount int
	err := QueryRowSQL(query, interfaceSlice(t.ID), interfaceSlice(&fileCount))
	return fileCount, err
}

// GetReplyCount returns the number of posts in the thread, not including the top post or any deleted posts
func (t *Thread) GetReplyCount() (int, error) {
	const query = `SELECT COUNT(*) FROM DBPREFIXposts WHERE thread_id = ? AND is_top_post = FALSE AND is_deleted = FALSE`
	var numReplies int
	err := QueryRowSQL(query, interfaceSlice(t.ID), interfaceSlice(&numReplies))
	return numReplies, err
}

// GetPosts returns the posts in the thread, optionally excluding the top post. If limit >= 0, a limit is set.
// If reversed is true, it is returned in descending order
func (t *Thread) GetPosts(repliesOnly bool, boardPage bool, limit int) ([]Post, error) {
	query := selectPostsBaseSQL + "WHERE thread_id = ?"
	if boardPage {
		query = "SELECT * FROM (" + query + " AND is_deleted = FALSE ORDER BY id DESC LIMIT " +
			strconv.Itoa(limit+1) + ") AS posts ORDER BY id"
	} else if repliesOnly {
		query += " AND is_top_post = FALSE"
	}
	if !boardPage && limit > 0 {
		query += " LIMIT " + strconv.Itoa(limit)
	}

	rows, err := QuerySQL(query, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		if err = rows.Scan(
			&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
			&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
			&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &post.BannedMessage,
		); err != nil {
			return posts, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func (t *Thread) GetUploads() ([]Upload, error) {
	const query = selectFilesBaseSQL + ` WHERE post_id IN (
		SELECT id FROM DBPREFIXposts WHERE thread_id = ? and is_deleted = FALSE) AND filename != 'deleted'`
	rows, err := QuerySQL(query, t.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uploads []Upload
	for rows.Next() {
		var upload Upload
		err = rows.Scan(
			&upload.ID, &upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename,
			&upload.Checksum, &upload.FileSize, &upload.IsSpoilered, &upload.ThumbnailWidth,
			&upload.ThumbnailHeight, &upload.Width, &upload.Height,
		)
		if err != nil {
			return uploads, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

// deleteThread updates the thread and sets it as deleted, as well as the posts where thread_id = threadID
func deleteThread(threadID int) error {
	const deletePostsSQL = `UPDATE DBPREFIXposts SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE thread_id = ?`
	const deleteThreadSQL = `UPDATE DBPREFIXthreads SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := ExecSQL(deletePostsSQL, threadID)
	if err != nil {
		return err
	}
	_, err = ExecSQL(deleteThreadSQL, threadID)
	return err
}
