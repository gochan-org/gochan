package gcsql

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/gochan-org/gochan/pkg/events"
)

const (
	selectFilesBaseSQL = `SELECT
	id, post_id, file_order, original_filename, filename, checksum,
	file_size, is_spoilered, thumbnail_width, thumbnail_height, width, height
	FROM DBPREFIXfiles `
)

var (
	ErrAlreadyAttached = errors.New("upload already processed")
)

// GetThreadFiles gets a list of the files owned by posts in the thread, including thumbnails for convenience.
func GetThreadFiles(post *Post) ([]Upload, error) {
	query := selectFilesBaseSQL + `WHERE post_id IN (
		SELECT id FROM DBPREFIXposts WHERE thread_id = (
			SELECT thread_id FROM DBPREFIXposts WHERE id = ?)) AND filename != 'deleted'`
	rows, err := Query(nil, query, post.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var uploads []Upload
	for rows.Next() {
		var upload Upload
		if err = rows.Scan(
			&upload.ID, &upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum,
			&upload.FileSize, &upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
		); err != nil {
			return uploads, err
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}

// NextFileOrder gets what would be the next file_order value (not particularly useful until multi-file posting is implemented)
func (p *Post) NextFileOrder(requestOpts ...*RequestOptions) (int, error) {
	opts := setupOptions(requestOpts...)
	const query = `SELECT COALESCE(MAX(file_order) + 1, 0) FROM DBPREFIXfiles WHERE post_id = ?`
	var next int
	err := QueryRow(opts, query, []any{p.ID}, []any{&next})
	return next, err
}

func (p *Post) AttachFile(upload *Upload, requestOpts ...*RequestOptions) error {
	if upload == nil {
		return nil // no upload to attach, so no error
	}
	opts := setupOptions(requestOpts...)
	shouldCommit := opts.Tx == nil
	var err error
	if shouldCommit {
		opts.Tx, err = BeginTx()
		if err != nil {
			return err
		}
		defer opts.Tx.Rollback()
	}

	_, err, recovered := events.TriggerEvent("incoming-upload", upload)
	if recovered {
		return errors.New("recovered from a panic in an event handler (incoming-upload)")
	}
	if err != nil {
		return fmt.Errorf("unable to attach upload to post: %w", err)
	}

	const insertSQL = `INSERT INTO DBPREFIXfiles (
		post_id, file_order, original_filename, filename, checksum, file_size,
		is_spoilered, thumbnail_width, thumbnail_height, width, height)
	VALUES(?,?,?,?,?,?,?,?,?,?,?)`
	if upload.ID > 0 {
		return ErrAlreadyAttached
	}
	if upload.FileOrder < 1 {
		upload.FileOrder, err = p.NextFileOrder(opts)
		if err != nil {
			return err
		}
	}
	upload.PostID = p.ID
	if _, err = Exec(opts, insertSQL,
		&upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum, &upload.FileSize,
		&upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
	); err != nil {
		return err
	}

	upload.ID, err = getLatestID(opts, "DBPREFIXfiles")
	if err != nil {
		return err
	}
	if shouldCommit {
		if err = opts.Tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// GetUploadFilenameAndBoard returns the filename (or an empty string) and
// the board of the given post ID
func GetUploadFilenameAndBoard(postID int) (string, string, error) {
	const query = `SELECT filename, dir FROM DBPREFIXfiles
		JOIN DBPREFIXposts ON post_id = DBPREFIXposts.id
		JOIN DBPREFIXthreads ON thread_id = DBPREFIXthreads.id
		JOIN DBPREFIXboards ON DBPREFIXboards.id = board_id
		WHERE DBPREFIXposts.id = ?`
	var filename, dir string
	err := QueryRow(nil, query, []any{postID}, []any{&filename, &dir})
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	} else if err != nil {
		return "", "", err
	}
	return filename, dir, nil
}
