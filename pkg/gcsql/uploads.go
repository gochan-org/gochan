package gcsql

import (
	"database/sql"
	"errors"

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
	rows, err := QuerySQL(query, post.ID)
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

func (p *Post) nextFileOrder() (int, error) {
	const query = `SELECT COALESCE(MAX(file_order) + 1, 0) FROM DBPREFIXfiles WHERE post_id = ?`
	var next int
	err := QueryRowSQL(query, []any{p.ID}, []any{&next})
	return next, err
}

func (p *Post) AttachFileTx(tx *sql.Tx, upload *Upload) error {
	if upload == nil {
		return nil // no upload to attach, so no error
	}

	_, err, recovered := events.TriggerEvent("incoming-upload", upload)
	if recovered {
		return errors.New("recovered from a panic in an event handler (incoming-upload)")
	}
	if err != nil {
		return errors.New("unable to attach upload to post: " + err.Error())
	}

	const insertSQL = `INSERT INTO DBPREFIXfiles (
		post_id, file_order, original_filename, filename, checksum, file_size,
		is_spoilered, thumbnail_width, thumbnail_height, width, height)
	VALUES(?,?,?,?,?,?,?,?,?,?,?)`
	if upload.ID > 0 {
		return ErrAlreadyAttached
	}

	if _, err = ExecTxSQL(tx, insertSQL,
		&upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum, &upload.FileSize,
		&upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
	); err != nil {
		return err
	}

	upload.ID, err = getLatestID("DBPREFIXfiles", tx)
	return err
}

func (p *Post) AttachFile(upload *Upload) error {
	tx, err := BeginTx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = p.AttachFileTx(tx, upload); err != nil {
		return err
	}
	return tx.Commit()
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
	err := QueryRowSQL(query, []any{postID}, []any{&filename, &dir})
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil
	} else if err != nil {
		return "", "", err
	}
	return filename, dir, nil
}
