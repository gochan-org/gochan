package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// should be appended when selecting info from DBPREFIXboards, requires a post ID
	boardFromPostIdSuffixSQL = ` WHERE DBPREFIXboards.id = (
		SELECT board_id FROM DBPREFIXthreads WHERE id = (
			SELECT thread_id FROM DBPREFIXposts WHERE id = ?))`

	selectPostsBaseSQL = `SELECT 
	id, thread_id, is_top_post, IP_NTOA, created_on, name, tripcode, is_role_signature,
	email, subject, message, message_raw, password, deleted_at, is_deleted,
	COALESCE(banned_message,'') AS banned_message, flag, country
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
	err := QueryRowSQL(query, []any{id}, []any{
		&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
		&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
		&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted,
		&post.BannedMessage, &post.Flag, &post.Country,
	})
	if err == sql.ErrNoRows {
		return nil, ErrPostDoesNotExist

	}
	return post, err
}

func GetPostIP(postID int) (string, error) {
	sql := "SELECT IP_NTOA FROM DBPREFIXposts WHERE id = ?"
	var ip string
	err := QueryRowSQL(sql, []any{postID}, []any{&ip})
	return ip, err
}

// GetPostsFromIP gets the posts from the database with a matching IP address, specifying
// optionally requiring them to not be deleted
func GetPostsFromIP(ip string, limit int, onlyNotDeleted bool) ([]Post, error) {
	sql := selectPostsBaseSQL + ` WHERE DBPREFIXposts.ip = PARAM_ATON`
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
			&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted,
			&post.BannedMessage, &post.Flag, &post.Country,
		); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	return posts, nil
}

// GetTopPostAndBoardDirFromPostID returns the ID of the top post and the board dir in postID's thread
func GetTopPostAndBoardDirFromPostID(postID int) (int, string, error) {
	const query = "SELECT op_id, dir FROM DBPREFIXv_top_post_board_dir WHERE id = ?"
	var opID int
	var dir string
	err := QueryRowTimeoutSQL(nil, query, []any{postID}, []any{&opID, &dir})
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return opID, dir, err
}

// GetTopPostIDsInThreadIDs takes a variable number of threads and returns a map[threadID]topPostID
func GetTopPostIDsInThreadIDs(threads ...any) (map[any]int, error) {
	ids := make(map[any]int)
	if threads == nil {
		return ids, nil
	}
	params := createArrayPlaceholder(threads)
	query := `SELECT id FROM DBPREFIXposts WHERE thread_id in ` + params + " AND is_top_post"
	rows, err := QuerySQL(query, threads...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var i int
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}

		ids[threads[i]] = id
		i++
	}
	return ids, nil
}

func GetThreadTopPost(threadID int) (*Post, error) {
	const query = selectPostsBaseSQL + "WHERE thread_id = ? AND is_top_post = TRUE LIMIT 1"
	post := new(Post)
	err := QueryRowSQL(query, []any{threadID}, []any{
		&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
		&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
		&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted,
		&post.BannedMessage, &post.Flag, &post.Country,
	})
	return post, err
}

// GetBoardTopPosts gets the top posts of the given
func GetBoardTopPosts[B intOrStringConstraint](board B) ([]*Post, error) {
	query := `SELECT id, thread_id, is_top_post, ip, created_on, name, tripcode, is_role_signature,
		email, subject, message, message_raw, password, deleted_at, is_deleted, coalesce(banned_message,''),
		flag, country
		FROM DBPREFIXv_post_with_board WHERE is_top_post AND is_deleted = FALSE`
	switch any(board).(type) {
	case int:
		query += " AND id = ?"
	case string:
		query += " AND dir = ?"
	}

	rows, cancel, err := QueryTimeoutSQL(nil, query, board)
	if err != nil {
		return nil, err
	}
	defer func() {
		rows.Close()
		cancel()
	}()
	var posts []*Post
	for rows.Next() {
		var post Post
		if err = rows.Scan(
			&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name,
			&post.Tripcode, &post.IsRoleSignature, &post.Email, &post.Subject, &post.Message,
			&post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &post.BannedMessage,
			&post.Flag, &post.Country,
		); err != nil {
			return posts, err
		}
		posts = append(posts, &post)
	}
	return posts, nil
}

// GetPostPassword returns the password checksum of the post with the given ID
func GetPostPassword(id int) (string, error) {
	const query = `SELECT password FROM DBPREFIXposts WHERE id = ?`
	var passwordChecksum string
	err := QueryRowSQL(query, []any{id}, []any{&passwordChecksum})
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
	err := QueryRowSQL(query, []any{postIP}, []any{&whenStr})
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

	err := QueryRowSQL(query, []any{postIP}, []any{&whenStr})
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
	err := QueryRowSQL(query, []any{p.ThreadID}, []any{&boardID})
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrBoardDoesNotExist
	}
	return boardID, err
}

func (p *Post) GetBoardDir() (string, error) {
	const query = "SELECT dir FROM DBPREFIXboards" + boardFromPostIdSuffixSQL
	var dir string
	err := QueryRowSQL(query, []any{p.ID}, []any{&dir})
	return dir, err
}

func (p *Post) GetBoard() (*Board, error) {
	const query = selectBoardsBaseSQL + boardFromPostIdSuffixSQL
	board := new(Board)
	err := QueryRowSQL(query, []any{p.ID}, []any{
		&board.ID, &board.SectionID, &board.URI, &board.Dir, &board.NavbarPosition, &board.Title, &board.Subtitle,
		&board.Description, &board.MaxFilesize, &board.MaxThreads, &board.DefaultStyle, &board.Locked,
		&board.CreatedAt, &board.AnonymousName, &board.ForceAnonymous, &board.AutosageAfter, &board.NoImagesAfter,
		&board.MaxMessageLength, &board.MinMessageLength, &board.AllowEmbeds, &board.RedirectToThread, &board.RequireFile,
		&board.EnableCatalog,
	})
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
	err := QueryRowSQL(query, []any{p.ThreadID}, []any{&topPostID})
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
	err := QueryRowSQL(query, []any{p.ID}, []any{
		&upload.ID, &upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum,
		&upload.FileSize, &upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
	})
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

// InCyclicThread returns true if the post is in a cyclic thread
func (p *Post) InCyclicThread() (bool, error) {
	var cyclic bool
	err := QueryRowTimeoutSQL(nil, "SELECT cyclical FROM DBPREFIXthreads WHERE id = ?", []any{p.ThreadID}, []any{&cyclic})
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrThreadDoesNotExist
	}
	return cyclic, err
}

// Delete sets the post as deleted and sets the deleted_at timestamp to the current time
func (p *Post) Delete() error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()
	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var rowCount int
	err = QueryRowContextSQL(ctx, tx, "SELECT COUNT(*) FROM DBPREFIXposts WHERE id = ?", []any{p.ID}, []any{&rowCount})
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrPostDoesNotExist
	}
	if err != nil {
		return err
	}

	if p.IsTopPost {
		return deleteThread(ctx, tx, p.ThreadID)
	}
	if _, err = ExecContextSQL(ctx, tx, "UPDATE DBPREFIXposts SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?", p.ID); err != nil {
		return err
	}
	return tx.Commit()
}

// InsertWithContext inserts the post into the database with the given context and transaction
func (p *Post) InsertWithContext(ctx context.Context, tx *sql.Tx, bumpThread bool, boardID int, locked bool, stickied bool, anchored bool, cyclical bool) error {
	if p.ID > 0 {
		// already inserted
		return ErrorPostAlreadySent
	}
	insertSQL := `INSERT INTO DBPREFIXposts
	(thread_id, is_top_post, ip, created_on, name, tripcode, is_role_signature, email, subject,
		message, message_raw, password, flag, country) 
	VALUES(?,?,PARAM_ATON,CURRENT_TIMESTAMP,?,?,?,?,?,?,?,?,?,?)`
	bumpSQL := `UPDATE DBPREFIXthreads SET last_bump = CURRENT_TIMESTAMP WHERE id = ?`

	var err error
	if p.ThreadID == 0 {
		// thread doesn't exist yet, this is a new post
		p.IsTopPost = true
		var threadID int
		threadID, err = CreateThread(tx, boardID, locked, stickied, anchored, cyclical)
		if err != nil {
			return err
		}
		p.ThreadID = threadID
	} else {
		var threadIsLocked bool
		if err = QueryRowTxSQL(tx, "SELECT locked FROM DBPREFIXthreads WHERE id = ?",
			[]any{p.ThreadID}, []any{&threadIsLocked}); err != nil {
			return err
		}
		if threadIsLocked {
			return ErrThreadLocked
		}
	}

	if _, err = ExecContextSQL(ctx, tx, insertSQL,
		p.ThreadID, p.IsTopPost, p.IP, p.Name, p.Tripcode, p.IsRoleSignature, p.Email, p.Subject,
		p.Message, p.MessageRaw, p.Password, p.Flag, p.Country,
	); err != nil {
		return err
	}
	if p.ID, err = getLatestID("DBPREFIXposts", tx); err != nil {
		return err
	}
	if bumpThread {
		if _, err = ExecContextSQL(ctx, tx, bumpSQL, p.ThreadID); err != nil {
			return err
		}
	}
	return nil
}

func (p *Post) Insert(bumpThread bool, boardID int, locked bool, stickied bool, anchored bool, cyclical bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	tx, err := BeginContextTx(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = p.InsertWithContext(ctx, tx, bumpThread, boardID, locked, stickied, anchored, cyclical); err != nil {
		return err
	}

	return tx.Commit()
}

// CyclicThreadPost represents a post that should be deleted in a cyclic thread
type CyclicThreadPost struct {
	PostID    int    // sql: post_id
	ThreadID  int    // sql: thread_id
	OPID      int    // sql: op_id
	IsTopPost bool   // sql: is_top_post
	Filename  string // sql: filename
	Dir       string // sql: dir
}

// CyclicPostsToBePruned returns posts that should be deleted in a cyclic thread that has reached its board's post limit
func (p *Post) CyclicPostsToBePruned() ([]CyclicThreadPost, error) {
	if p.IsTopPost {
		// don't prune if this is the OP
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), gcdb.defaultTimeout)
	defer cancel()

	var cyclic bool
	err := QueryRowContextSQL(ctx, nil, "SELECT cyclical FROM DBPREFIXthreads WHERE id = ?", []any{p.ThreadID}, []any{&cyclic})
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrThreadDoesNotExist
	}
	if err != nil {
		return nil, err
	}

	if !cyclic {
		return nil, nil
	}

	rows, err := QueryContextSQL(ctx, nil, `SELECT post_id, thread_id, op_id, filename, dir
		FROM DBPREFIXv_posts_cyclical_check WHERE thread_id = ? AND post_id <> op_id ORDER BY post_id ASC`, p.ThreadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var posts []CyclicThreadPost
	for rows.Next() {
		var post CyclicThreadPost
		if err = rows.Scan(&post.PostID, &post.ThreadID, &post.OPID, &post.Filename, &post.Dir); err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	boardCfg := config.GetBoardConfig(posts[0].Dir)
	if !boardCfg.EnableCyclicThreads {
		return nil, nil
	}
	cyclicThreadMaxPosts := boardCfg.CyclicThreadNumPosts
	if cyclicThreadMaxPosts < 1 {
		// no limit set
		return nil, nil
	}

	if len(posts) == 0 || len(posts) < cyclicThreadMaxPosts {
		return nil, nil
	}

	return posts[:len(posts)-cyclicThreadMaxPosts], nil
}

func (p *Post) WebPath() string {
	if p.opID > 0 && p.boardDir != "" {
		return config.WebPath(p.boardDir, "res/", strconv.Itoa(p.opID)+".html#"+strconv.Itoa(p.ID))
	}
	webRoot := config.GetSystemCriticalConfig().WebRoot

	const query = "SELECT op_id, dir FROM DBPREFIXv_top_post_board_dir WHERE id = ?"
	err := QueryRowSQL(query, []any{p.ID}, []any{&p.opID, &p.boardDir})
	if err != nil {
		return webRoot
	}
	return config.WebPath(p.boardDir, "res/", strconv.Itoa(p.opID)+".html#"+strconv.Itoa(p.ID))
}
