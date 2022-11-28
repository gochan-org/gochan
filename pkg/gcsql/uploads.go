package gcsql

import (
	"errors"

	"github.com/gochan-org/gochan/pkg/gcutil"
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

func (p *Post) AttachFile(upload *Upload) error {
	if upload == nil {
		return nil //
	}
	const query = `INSERT INTO DBPREFIXfiles (
		post_id, file_order, original_filename, filename, checksum, file_size,
		is_spoilered, thumbnail_width, thumbnail_height, width, height)
	VALUES(?,?,?,?,?,?,?,?,?,?,?)`
	if upload.ID > 0 {
		return ErrAlreadyAttached
	}
	uploadID, err := getNextFreeID("DBPREFIXfiles")
	if err != nil {
		return err
	}
	if _, err = ExecSQL(query,
		&upload.PostID, &upload.FileOrder, &upload.OriginalFilename, &upload.Filename, &upload.Checksum, &upload.FileSize,
		&upload.IsSpoilered, &upload.ThumbnailWidth, &upload.ThumbnailHeight, &upload.Width, &upload.Height,
	); err != nil {
		return err
	}
	upload.ID = uploadID
	return nil
}

// ThumbnailPath returns the thumbnail path of the upload, given an thumbnail type ("thumbnail" or "catalog")
func (u *Upload) ThumbnailPath(thumbType string) string {
	return gcutil.GetThumbnailPath(thumbType, u.Filename)
}
