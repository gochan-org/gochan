package pre2021

import (
	"time"

	"github.com/gochan-org/gochan/pkg/gclog"
)

type postTable struct {
	id                int
	boardid           int
	parentid          int
	name              string
	tripcode          string
	email             string
	subject           string
	message           string
	message_raw       string
	password          string
	filename          string
	filename_original string
	file_checksum     string
	filesize          int
	image_w           int
	image_h           int
	thumb_w           int
	thumb_h           int
	ip                string
	tag               string
	timestamp         time.Time
	autosage          bool
	deleted_timestamp time.Time
	bumped            time.Time
	stickied          bool
	locked            bool
	reviewed          bool

	newBoardID  int
	newParentID int
}

func (m *Pre2021Migrator) MigratePosts() error {
	var err error
	if err = m.migrateThreads(); err != nil {
		return err
	}
	return m.migratePostsUtil()
}

func (m *Pre2021Migrator) migrateThreads() error {
	rows, err := m.db.QuerySQL(`SELECT
	id,
	boardid,
	parentid,
	name,
	tripcode,
	email,
	subject,
	message,
	message_raw,
	password,
	filename,
	filename_original,
	file_checksum,
	filesize,
	image_w,
	image_h,
	thumb_w,
	thumb_h,
	ip,
	tag,
	timestamp,
	autosage,
	deleted_timestamp,
	bumped,
	stickied,
	locked,
	reviewed from DBPREFIXposts WHERE is_deleted = 0`)
	if err != nil {
		return err
	}

	for rows.Next() {
		var post postTable
		if err = rows.Scan(
			&post.id,
			&post.boardid,
			&post.parentid,
			&post.name,
			&post.tripcode,
			&post.email,
			&post.subject,
			&post.message,
			&post.message_raw,
			&post.password,
			&post.filename,
			&post.filename_original,
			&post.file_checksum,
			&post.filesize,
			&post.image_w,
			&post.image_h,
			&post.thumb_w,
			&post.thumb_h,
			&post.ip,
			&post.tag,
			&post.timestamp,
			&post.autosage,
			&post.deleted_timestamp,
			&post.bumped,
			&post.stickied,
			&post.locked,
			&post.reviewed,
		); err != nil {
			return err
		}
		_, ok := m.oldBoards[post.boardid]
		if !ok {
			// board doesn't exist
			gclog.Printf(gclog.LStdLog|gclog.LErrorLog, "Pre-migrated post #%d has an invalid boardid %d (board doesn't exist), skipping", post.id, post.boardid)
			continue
		}

		// gcsql.QueryRowSQL(`SELECT id FROM DBPREFIXboards WHERE uri = ?`, []interface{}{})
		m.posts = append(m.posts, post)
		if post.parentid == 0 {
			// post is a thread, save it to the DBPREFIXthreads table

		}
	}
	return nil
}

func (m *Pre2021Migrator) migratePostsUtil() error {
	return nil
}
