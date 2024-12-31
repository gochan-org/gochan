package pre2021

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
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

	newBoardID int
	foundBoard bool
	// oldParentID int
}

func (m *Pre2021Migrator) MigratePosts() error {
	var err error
	if err = m.migrateThreads(); err != nil {
		return err
	}
	return m.migratePostsUtil()
}

func (m *Pre2021Migrator) migrateThreads() error {
	tx, err := m.db.Begin()
	if err != nil {
		return err
	}

	stmt, err := m.db.PrepareSQL(postsQuery, tx)
	if err != nil {
		tx.Rollback()
		return err
	}
	rows, err := stmt.Query()
	if err != nil && err != sql.ErrNoRows {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	defer rows.Close()
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
			tx.Rollback()
			return err
		}
		var postBoardDir string
		for _, oldBoard := range m.oldBoards {
			if oldBoard.id == post.boardid {
				postBoardDir = oldBoard.dir
			}
		}
		for _, newBoard := range gcsql.AllBoards {
			if newBoard.Dir == postBoardDir {
				post.newBoardID = newBoard.ID
				post.foundBoard = true
			}
		}
		if !post.foundBoard {
			common.LogWarning().Int("boardID", post.boardid).
				Msg("Pre-migrated post has an invalid boardid (board doesn't exist), skipping")
			continue
		}

		// var stmt *sql.Stmt
		// var err error
		preparedStr, _ := gcsql.SetupSQLString(`SELECT id FROM DBPREFIXboards WHERE ui = ?`, m.db)
		stmt, err := tx.Prepare(preparedStr)
		if err != nil {
			tx.Rollback()
			return err
		}
		stmt.QueryRow(post.boardid).Scan(&post.newBoardID)

		// gcsql.QueryRowSQL(`SELECT id FROM DBPREFIXboards WHERE uri = ?`, []interface{}{})
		if post.parentid == 0 {
			// post is a thread, save it to the DBPREFIXthreads table
			// []interfaceP{{post.newParentID}

			if err = gcsql.QueryRowSQL(
				`SELECT board_id FROM DBPREFIXthreads ORDER BY board_id LIMIT 1`,
				nil,
				[]interface{}{&post.newBoardID},
			); err != nil {
				tx.Rollback()
				return err
			}
			fmt.Println("Current board ID:", post.newBoardID)
			prepareStr, _ := gcsql.SetupSQLString(
				`INSERT INTO DBPREFIXthreads
				(board_id, locked, stickied)
				VALUES(?, ?, ?)`, m.db)
			stmt, err = tx.Prepare(prepareStr)
			if err != nil {
				tx.Rollback()
				return err
			}
			stmt.Exec(post.newBoardID, post.locked, post.stickied)
			// 			// stmt, err := db.Prepare("INSERT table SET unique_id=? ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id)")
			// 			gcsql.ExecSQL(`INSERT INTO DBPREFIXthreads (board_id) VALUES(?)`, post.newBoardID)

			// 			/*
			// id
			// board_id
			// locked
			// stickied
			// anchored
			// cyclical
			// last_bump
			// deleted_at
			// is_deleted

			// 			*/

		}
		m.posts = append(m.posts, post)
	}
	return tx.Commit()
}

func (*Pre2021Migrator) migratePostsUtil() error {
	return nil
}
