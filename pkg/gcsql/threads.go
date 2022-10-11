package gcsql

import "errors"

var (
	ErrThreadExists       = errors.New("thread already exists")
	ErrThreadDoesNotExist = errors.New("thread does not exist")
)

// GetThreadsWithBoardID queries the database for the threads with the given board ID from the database.
// If onlyNotDeleted is true, it omits deleted threads and threads that were removed because the max
// thread limit was reached
func GetThreadsWithBoardID(boardID int, onlyNotDeleted bool) ([]Thread, error) {
	query := `SELECT
	id, board_id, locked, stickied, anchored, cyclical, last_bump, deleted_at, is_deleted
	FROM DBPREFIXthreads WHERE board_id = ?`
	if onlyNotDeleted {
		query += " AND  is_deleted = FALSE"
	}
	rows, err := QuerySQL(query, boardID)
	if err != nil {
		return nil, err
	}
	var threads []Thread
	for rows.Next() {
		var thread Thread
		if err = rows.Scan(
			&thread.ID, &thread.BoardID, &thread.Locked, &thread.Stickied, &thread.Anchored,
			&thread.Cyclical, &thread.LastBump,
		); err != nil {
			return threads, err
		}
	}
	return threads, nil
}

func GetThreadReplyCountFromOP(opID int) (int, error) {
	const query = `SELECT COUNT(*) FROM DBPREFIXposts WHERE thread_id = (
		SELECT thread_id FROM DBPREFIXposts WHERE id = ?) AND is_deleted = 0`
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
