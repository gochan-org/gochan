package gcsql

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// should be appended when selecting info from DBPREFIXboards, requires a post ID
	boardFromPostIdSuffixSQL = ` WHERE DBPREFIXboards.id = (
		SELECT board_id FROM DBPREFIXthreads WHERE id = (
			SELECT thread_id FROM DBPREFIXposts WHERE id = ?))`
	selectPostsBaseSQL = `SELECT 
	id, thread_id, is_top_post, ip, created_on, name, tripcode, is_role_signature,
	email, subject, message, message_raw, password, deleted_at, is_deleted, COALESCE(banned_message,'') AS banned_message
	FROM DBPREFIXposts `
)

var (
	ErrNotTopPost        = errors.New("not the top post in the thread")
	ErrPostDoesNotExist  = errors.New("post does not exist")
	ErrPostDeleted       = errors.New("post is deleted")
	ErrorPostAlreadySent = errors.New("post already submitted")
	// TempPosts is a cached list of all of the posts in the temporary posts table, used for temporarily storing CAPTCHA
	TempPosts []Post
)

func GetPostFromID(id int, onlyNotDeleted bool) (*Post, error) {
	query := selectPostsBaseSQL + "WHERE id = ?"
	if onlyNotDeleted {
		query += " AND is_deleted = FALSE"
	}
	post := new(Post)
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(
		&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
		&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
		&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &post.BannedMessage,
	))
	if err == sql.ErrNoRows {
		return nil, ErrPostDoesNotExist

	}
	return post, err
}

// GetPostsFromIP gets the posts from the database with a matching IP address, specifying
// optionally requiring them to not be deleted
func GetPostsFromIP(ip string, limit int, onlyNotDeleted bool) ([]Post, error) {
	sql := selectPostsBaseSQL + ` WHERE DBPREFIXposts.ip = ?`
	if onlyNotDeleted {
		sql += " AND is_deleted = FALSE"
	}

	sql += " ORDER BY id DESC LIMIT ?"
	rows, err := QuerySQL(sql, ip, limit)
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
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

func GetTopPostInThread(postID int) (int, error) {
	const query = `SELECT id FROM DBPREFIXposts WHERE thread_id = (
		SELECT thread_id FROM DBPREFIXposts WHERE id = ?
	) AND is_top_post = TRUE ORDER BY id ASC LIMIT 1`
	var id int
	err := QueryRowSQL(query, interfaceSlice(postID), interfaceSlice(&id))
	return id, err
}

func GetThreadTopPost(threadID int) (*Post, error) {
	const query = selectPostsBaseSQL + "WHERE thread_id = ? AND is_top_post = TRUE LIMIT 1"
	post := new(Post)
	err := QueryRowSQL(query, interfaceSlice(threadID), interfaceSlice(
		&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
		&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
		&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &post.BannedMessage,
	))
	return post, err
}

func GetBoardTopPosts(boardID int) ([]Post, error) {
	query := `SELECT DBPREFIXposts.id, thread_id, is_top_post, ip, created_on, name,
		tripcode, is_role_signature, email, subject, message, message_raw,
		password, deleted_at, is_deleted, banned_message
		FROM DBPREFIXposts
		LEFT JOIN (
		SELECT id, board_id from DBPREFIXthreads
		) t on t.id = DBPREFIXposts.thread_id
		WHERE is_deleted = FALSE AND is_top_post AND t.board_id = ?`

	rows, err := QuerySQL(query, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []Post
	for rows.Next() {
		var post Post
		// var tmp int // only needed for WHERE clause in query

		bannedMessage := new(string)
		err = rows.Scan(
			&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
			&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
			&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &bannedMessage,
		)
		if err != nil {
			return posts, err
		}
		if bannedMessage != nil {
			post.BannedMessage = *bannedMessage
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// GetPostPassword returns the password checksum of the post with the given ID
func GetPostPassword(id int) (string, error) {
	const query = `SELECT password FROM DBPREFIXposts WHERE id = ?`
	var passwordChecksum string
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(&passwordChecksum))
	return passwordChecksum, err
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

// SinceLastPost returns the number of seconds since the given IP address created a post
// (used for checking against the new reply cooldown)
func SinceLastPost(postIP string) (int, error) {
	const query = `SELECT COALESCE(MAX(created_on), '1970-01-01 00:00:00') FROM DBPREFIXposts WHERE ip = ?`
	var whenStr string
	err := QueryRowSQL(query, interfaceSlice(postIP), interfaceSlice(&whenStr))
	if err != nil {
		return -1, err
	}

	when, err := ParseSQLTimeString(whenStr)
	if err != nil {
		return -1, err
	}
	return int(time.Since(when).Seconds()), nil
}

// SinceLastThread returns the number of seconds since the given IP address created a new thread/top post
// (used for checking against the new thread cooldown)
func SinceLastThread(postIP string) (int, error) {
	const query = `SELECT COALESCE(MAX(created_on), '1970-01-01 00:00:00') FROM DBPREFIXposts WHERE ip = ? AND is_top_post`
	var whenStr string

	err := QueryRowSQL(query, interfaceSlice(postIP), interfaceSlice(&whenStr))
	if err != nil {
		return -1, err
	}
	when, err := ParseSQLTimeString(whenStr)
	if err != nil {
		return -1, err
	}
	return int(time.Since(when).Seconds()), nil
}

// UpdateContents updates the email, subject, and message text of the post
func (p *Post) UpdateContents(email string, subject string, message template.HTML, messageRaw string) error {
	const sqlUpdate = `UPDATE DBPREFIXposts SET email = ?, subject = ?, message = ?, message_raw = ? WHERE ID = ?`
	_, err := ExecSQL(sqlUpdate, email, subject, message, messageRaw, p.ID)
	if err != nil {
		return err
	}
	p.Email = email
	p.Subject = subject
	p.Message = message
	p.MessageRaw = messageRaw
	return nil
}

func (p *Post) GetBoardID() (int, error) {
	const query = `SELECT board_id FROM DBPREFIXthreads where id = ?`
	var boardID int
	err := QueryRowSQL(query, interfaceSlice(p.ThreadID), interfaceSlice(&boardID))
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrBoardDoesNotExist
	}
	return boardID, err
}

func (p *Post) GetBoardDir() (string, error) {
	const query = "SELECT dir FROM DBPREFIXboards" + boardFromPostIdSuffixSQL
	var dir string
	err := QueryRowSQL(query, interfaceSlice(p.ID), interfaceSlice(&dir))
	return dir, err
}

func (p *Post) GetBoard() (*Board, error) {
	const query = selectBoardsBaseSQL + boardFromPostIdSuffixSQL
	board := new(Board)
	err := QueryRowSQL(query, interfaceSlice(p.ID), interfaceSlice(
		&board.ID, &board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
		&board.MaxMessageLength, &board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile,
		&board.EnableCatalog,
	))
	return board, err
}

// ChangeBoardID updates the post with the new board ID if it is a top post. It returns an error if it is not
// an OP or if ChangeThreadBoardID returned any errors
func (p *Post) ChangeBoardID(newBoardID int) error {
	if !p.IsTopPost {
		return ErrNotTopPost
	}
	return ChangeThreadBoardID(p.ThreadID, newBoardID)
}

// TopPostID returns the OP post ID of the thread that p is in
func (p *Post) TopPostID() (int, error) {
	if p.IsTopPost {
		return p.ID, nil
	}
	const query = `SELECT id FROM DBPREFIXposts WHERE thread_id = ? and is_top_post = TRUE ORDER BY id ASC LIMIT 1`
	var topPostID int
	err := QueryRowSQL(query, interfaceSlice(p.ThreadID), interfaceSlice(&topPostID))
	return topPostID, err
}

// GetTopPost returns the OP of the thread that p is in
func (p *Post) GetTopPost() (*Post, error) {
	opID, err := p.TopPostID()
	if err != nil {
		return nil, err
	}
	return GetPostFromID(opID, true)
}

// GetPostUpload returns the upload info associated with the file as well as any errors encountered.
// If the file has no uploads, then *Upload is nil. If the file was removed from the post, then Filename
// and OriginalFilename = "deleted"
func (p *Post) GetUpload() (*Upload, error) {
	const query = `SELECT
	id, post_id, file_order, original_filename, filename, checksum,
	file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height
	FROM DBPREFIXfiles WHERE post_id = ?`
	upload := new(Upload)
	err := QueryRowSQL(query, interfaceSlice(p.ID), interfaceSlice(
		&upload.ID, &upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum,
		&upload.FileSize, &upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return upload, err
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

// Delete sets the post as deleted and sets the deleted_at timestamp to the current time
func (p *Post) Delete() error {
	if p.IsTopPost {
		return deleteThread(p.ThreadID)
	}
	const deleteSQL = `UPDATE DBPREFIXposts SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := ExecSQL(deleteSQL, p.ID)
	return err
}

func (p *Post) Insert(bumpThread bool, boardID int, locked bool, stickied bool, anchored bool, cyclical bool) error {
	if p.ID > 0 {
		// already inserted
		return ErrorPostAlreadySent
	}
	insertSQL := `INSERT INTO DBPREFIXposts
	(thread_id, is_top_post, ip, created_on, name, tripcode, is_role_signature, email, subject,
		message, message_raw, password) 
	VALUES(?,?,?,CURRENT_TIMESTAMP,?,?,?,?,?,?,?,?)`
	bumpSQL := `UPDATE DBPREFIXthreads SET last_bump = CURRENT_TIMESTAMP WHERE id = ?`

	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if p.ThreadID == 0 {
		// thread doesn't exist yet, this is a new post
		p.IsTopPost = true
		var threadID int
		threadID, err = createThread(tx, boardID, locked, stickied, anchored, cyclical)
		if err != nil {
			return err
		}
		p.ThreadID = threadID
	}

	stmt, err := PrepareSQL(insertSQL, tx)
	if err != nil {
		return err
	}
	if _, err = stmt.Exec(
		p.ThreadID, p.IsTopPost, p.IP, p.Name, p.Tripcode, p.IsRoleSignature, p.Email, p.Subject,
		p.Message, p.MessageRaw, p.Password,
	); err != nil {
		return err
	}
	if p.ID, err = getLatestID("DBPREFIXposts", tx); err != nil {
		return err
	}
	if bumpThread {
		stmt2, err := PrepareSQL(bumpSQL, tx)
		if err != nil {
			return err
		}
		if _, err = stmt2.Exec(p.ThreadID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *Post) WebPath() string {
	webRoot := config.GetSystemCriticalConfig().WebRoot
	var opID int
	var boardDir string
	const query = `SELECT
		op.id,
		(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id) AS dir
	FROM DBPREFIXposts
	LEFT JOIN (
		SELECT id, board_id FROM DBPREFIXthreads
	) t ON t.id = DBPREFIXposts.thread_id
	INNER JOIN (
		SELECT id, thread_id FROM DBPREFIXposts WHERE is_top_post
	) op on op.thread_id = DBPREFIXposts.thread_id
	WHERE DBPREFIXposts.id = ?`
	err := QueryRowSQL(query, interfaceSlice(p.ID), interfaceSlice(&opID, &boardDir))
	if err != nil {
		return webRoot
	}
	return webRoot + boardDir + fmt.Sprintf("/res/%d.html#%d", opID, p.ID)
}
