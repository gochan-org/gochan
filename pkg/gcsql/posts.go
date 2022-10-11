package gcsql

import (
	"database/sql"
	"errors"
	"fmt"
	"html/template"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	// should be appended when selecting info from DBPREFIXboards, requires a post ID
	boardFromPostIdSuffixSQL = ` WHERE id = (
		SELECT board_id FROM DBPREFIXthreads WHERE id = (
			SELECT thread_id FROM DBPREFIXposts WHERE id = ?))`
	selectPostsBaseSQL = `SELECT 
	id, thread_id, is_top_post, ip, created_on, name, tripcode, is_role_signature,
	email, subject, message, message_raw, password, deleted_at, is_deleted, banned_message
	FROM DBPREFIXposts `
)

var (
	ErrNotTopPost       = errors.New("not the top post in the thread")
	ErrPostDoesNotExist = errors.New("post does not exist")
	ErrPostDeleted      = errors.New("post is deleted")
)

func GetPostFromID(id int, onlyNotDeleted bool) (*Post, error) {
	query := selectPostsBaseSQL + "WHERE id = ?"
	if onlyNotDeleted {
		query += " AND is_deleted = 0"
	}
	post := new(Post)
	post.ID = id
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(
		&post.ID, &post.ThreadID, &post.IsTopPost, &post.IP, &post.CreatedOn, &post.Name, &post.Tripcode, &post.IsRoleSignature,
		&post.Email, &post.Subject, &post.Message, &post.MessageRaw, &post.Password, &post.DeletedAt, &post.IsDeleted, &post.BannedMessage,
	))
	if err == sql.ErrNoRows {
		return nil, ErrPostDoesNotExist

	}
	return post, err
}

// GetPostPassword returns the password checksum of the post with the given ID
func GetPostPassword(id int) (string, error) {
	const query = `SELECT password FROM DBPREFIXposts WHERE id = ?`
	var passwordChecksum string
	err := QueryRowSQL(query, interfaceSlice(id), interfaceSlice(&passwordChecksum))
	return passwordChecksum, err
}

// UpdateContents updates the email, subject, and message text of the post
func (p *Post) UpdateContents(email string, subject string, message template.HTML, messageRaw string) error {
	const sqlUpdate = `UPDATE DBPREFIXposts SET email = ?, subject = ?, message = ?, message_raw = ? WHERE ID = ?`
	_, err := ExecSQL(sqlUpdate, email, subject, message, messageRaw)
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
	err := QueryRowSQL(query, interfaceSlice(), interfaceSlice(
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

func (p *Post) WebPath() string {
	webRoot := config.GetSystemCriticalConfig().WebRoot
	var threadID, opID, boardID int
	var boardDir string
	const query = `SELECT thread_id as threadid,
		(SELECT id from DBPREFIXposts WHERE thread_id = threadid AND is_top_post = TRUE LIMIT 1) as op,
		(SELECT board_id from DBPREFIXthreads WHERE id = threadid) AS boardid,
		(SELECT dir FROM DBPREFIXboards WHERE id = boardid) AS dir
	FROM DBPREFIXposts WHERE id = ?`
	err := QueryRowSQL(query, interfaceSlice(p.ID), interfaceSlice(&threadID, &opID, &boardID, &boardDir))
	if err != nil {
		return webRoot
	}
	return webRoot + boardDir + fmt.Sprintf("/res/%d.html#%d", opID, p.ID)
}
