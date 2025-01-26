package pre2021

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

type migrationPost struct {
	gcsql.Post
	autosage bool
	bumped   time.Time
	stickied bool
	locked   bool

	filename         string
	filenameOriginal string
	fileChecksum     string
	filesize         int
	imageW           int
	imageH           int
	thumbW           int
	thumbH           int

	oldID       int
	boardID     int
	oldBoardID  int
	oldParentID int
}

func (m *Pre2021Migrator) MigratePosts() error {
	if m.IsMigratingInPlace() {
		return m.migratePostsInPlace()
	}
	return m.migratePostsToNewDB()
}

func (m *Pre2021Migrator) migratePost(tx *sql.Tx, post *migrationPost, errEv *zerolog.Event) error {
	var err error

	if post.oldParentID == 0 {
		// migrating post was a thread OP, create the row in the threads table
		if post.ThreadID, err = gcsql.CreateThread(tx, post.boardID, false, post.stickied, post.autosage, false); err != nil {
			errEv.Err(err).Caller().
				Int("boardID", post.boardID).
				Msg("Failed to create thread")
		}
	}

	// insert thread top post
	if err = post.InsertWithContext(context.Background(), tx, true, post.boardID, false, post.stickied, post.autosage, false); err != nil {
		errEv.Err(err).Caller().
			Int("boardID", post.boardID).
			Int("threadID", post.ThreadID).
			Msg("Failed to insert thread OP")
	}

	if post.filename != "" {
		if err = post.AttachFileTx(tx, &gcsql.Upload{
			PostID:           post.ID,
			OriginalFilename: post.filenameOriginal,
			Filename:         post.filename,
			Checksum:         post.fileChecksum,
			FileSize:         post.filesize,
			ThumbnailWidth:   post.thumbW,
			ThumbnailHeight:  post.thumbH,
			Width:            post.imageW,
			Height:           post.imageH,
		}); err != nil {
			errEv.Err(err).Caller().
				Int("oldPostID", post.oldID).
				Msg("Failed to attach upload to migrated post")
			return err
		}
	}
	return nil
}

func (m *Pre2021Migrator) migratePostsToNewDB() error {
	errEv := common.LogError()
	defer errEv.Discard()

	tx, err := gcsql.BeginTx()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to start transaction")
		return err
	}
	defer tx.Rollback()

	rows, err := m.db.QuerySQL(threadsQuery)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to get threads")
		return err
	}
	defer rows.Close()

	var threadIDsWithInvalidBoards []int
	var missingBoardIDs []int
	var migratedThreads int
	for rows.Next() {
		var thread migrationPost
		if err = rows.Scan(
			&thread.oldID, &thread.oldBoardID, &thread.oldParentID, &thread.Name, &thread.Tripcode, &thread.Email,
			&thread.Subject, &thread.Message, &thread.MessageRaw, &thread.Password, &thread.filename,
			&thread.filenameOriginal, &thread.fileChecksum, &thread.filesize, &thread.imageW, &thread.imageH,
			&thread.thumbW, &thread.thumbH, &thread.IP, &thread.CreatedOn, &thread.autosage,
			&thread.bumped, &thread.stickied, &thread.locked,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan thread")
			return err
		}
		var foundBoard bool
		for _, board := range m.boards {
			if board.oldID == thread.oldBoardID {
				thread.boardID = board.ID
				foundBoard = true
				break
			}
		}
		if !foundBoard {
			threadIDsWithInvalidBoards = append(threadIDsWithInvalidBoards, thread.oldID)
			missingBoardIDs = append(missingBoardIDs, thread.oldBoardID)
			continue
		}

		if err = m.migratePost(tx, &thread, errEv); err != nil {
			return err
		}

		// get and insert replies
		replyRows, err := m.db.QuerySQL(postsQuery+" AND parentid = ?", thread.oldID)
		if err != nil {
			errEv.Err(err).Caller().
				Int("parentID", thread.oldID).
				Msg("Failed to get reply rows")
			return err
		}
		defer replyRows.Close()

		for replyRows.Next() {
			var reply migrationPost
			if err = replyRows.Scan(
				&reply.oldID, &reply.oldBoardID, &reply.oldParentID, &reply.Name, &reply.Tripcode, &reply.Email,
				&reply.Subject, &reply.Message, &reply.MessageRaw, &reply.Password, &reply.filename,
				&reply.filenameOriginal, &reply.fileChecksum, &reply.filesize, &reply.imageW, &reply.imageH,
				&reply.thumbW, &reply.thumbH, &reply.IP, &reply.CreatedOn, &reply.autosage,
				&reply.bumped, &reply.stickied, &reply.locked,
			); err != nil {
				errEv.Err(err).Caller().
					Int("parentID", thread.oldID).
					Msg("Failed to scan reply")
				return err
			}
			reply.ThreadID = thread.ThreadID
			if err = m.migratePost(tx, &reply, errEv); err != nil {
				return err
			}
		}

		if thread.locked {
			if _, err = gcsql.ExecTxSQL(tx, "UPDATE DBPREFIXthreads SET locked = TRUE WHERE id = ?", thread.ThreadID); err != nil {
				errEv.Err(err).Caller().
					Int("threadID", thread.ThreadID).
					Msg("Unable to re-lock migrated thread")
			}
		}
		migratedThreads++
	}
	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to close posts rows")
		return err
	}

	if len(threadIDsWithInvalidBoards) > 0 {
		errEv.Caller().
			Ints("threadIDs", threadIDsWithInvalidBoards).
			Ints("boardIDs", missingBoardIDs).
			Msg("Failed to find boards for threads")
		return common.NewMigrationError("pre2021", "Found threads with missing boards")
	}

	if err = tx.Commit(); err != nil {
		errEv.Err(err).Caller().Msg("Failed to commit transaction")
		return err
	}
	gcutil.LogInfo().
		Int("migratedThreads", migratedThreads).
		Msg("Migrated threads successfully")
	return nil
}

func (m *Pre2021Migrator) migratePostsInPlace() error {
	errEv := common.LogError()
	defer errEv.Discard()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(m.config.DBTimeoutSeconds))
	defer cancel()

	ba, err := os.ReadFile(gcutil.FindResource("sql/initdb_" + m.db.SQLDriver() + ".sql"))
	if err != nil {
		errEv.Err(err).Caller().
			Msg("Failed to read initdb SQL file")
		return err
	}
	statements := strings.Split(string(ba), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if strings.HasPrefix(statement, "CREATE TABLE DBPREFIXthreads") || strings.HasPrefix(statement, "CREATE TABLE DBPREFIXfiles") {
			if _, err = m.db.ExecContextSQL(ctx, nil, statement); err != nil {
				errEv.Err(err).Caller().Msg("Failed to create threads table")
				return err
			}
		}
	}

	rows, err := m.db.QueryContextSQL(ctx, nil, threadsQuery+" AND parentid = 0")
	if err != nil {
		errEv.Err(err).Caller().
			Msg("Failed to get threads")
		return err
	}
	defer rows.Close()

	var threads []migrationPost
	for rows.Next() {
		var post migrationPost
		if err = rows.Scan(
			&post.ID, &post.oldBoardID, &post.oldParentID, &post.Name, &post.Tripcode, &post.Email,
			&post.Subject, &post.Message, &post.MessageRaw, &post.Password, &post.filename,
			&post.filenameOriginal, &post.fileChecksum, &post.filesize, &post.imageW, &post.imageH,
			&post.thumbW, &post.thumbH, &post.IP, &post.CreatedOn, &post.autosage,
			&post.bumped, &post.stickied, &post.locked,
		); err != nil {
			errEv.Err(err).Caller().
				Msg("Failed to scan thread")
			return err
		}
		threads = append(threads, post)
	}
	if err = rows.Close(); err != nil {
		errEv.Caller().Msg("Failed to close thread rows")
		return err
	}

	for _, statements := range postAlterStatements {
		if _, err = m.db.ExecContextSQL(ctx, nil, statements); err != nil {
			errEv.Err(err).Caller().Msg("Failed to alter posts table")
			return err
		}
	}

	switch m.db.SQLDriver() {
	case "mysql":
		_, err = m.db.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN ip_new VARBINARY(16) NOT NULL")
	case "postgres", "postgresql":
		_, err = m.db.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN ip_new INET NOT NULL")
	case "sqlite3":
		_, err = m.db.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts ADD COLUMN ip_new VARCHAR(45) NOT NULL DEFAULT '0.0.0.0'")
	}
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to update IP column")
		return err
	}
	if _, err = m.db.ExecContextSQL(ctx, nil, "UPDATE DBPREFIXposts SET ip_new = IP_ATON"); err != nil {
		errEv.Err(err).Caller().Msg("Failed to update IP column")
		return err
	}
	if _, err = m.db.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts RENAME COLUMN ip TO ip_old"); err != nil {
		errEv.Err(err).Caller().Msg("Failed to rename old IP column")
		return err
	}
	if _, err = m.db.ExecContextSQL(ctx, nil, "ALTER TABLE DBPREFIXposts RENAME COLUMN ip_new TO ip"); err != nil {
		errEv.Err(err).Caller().Msg("Failed to rename new IP column")
		return err
	}

	for _, op := range threads {
		if _, err = m.db.ExecContextSQL(ctx, nil,
			`INSERT INTO DBPREFIXthreads(board_id,locked,stickied,anchored,cyclical,last_bump,is_deleted) VALUES(?,?,?,?,?,?,?)`,
			op.oldBoardID, op.locked, op.stickied, op.autosage, false, op.bumped, false,
		); err != nil {
			errEv.Err(err).Caller().
				Int("postID", op.ID).
				Msg("Failed to insert thread")
			return err
		}
		if err = m.db.QueryRowContextSQL(ctx, nil, "SELECT MAX(id) FROM DBPREFIXthreads", nil, []any{&op.ThreadID}); err != nil {
			errEv.Err(err).Caller().
				Int("postID", op.ID).
				Msg("Failed to get thread ID")
			return err
		}

		if _, err = m.db.ExecContextSQL(ctx, nil,
			"UPDATE DBPREFIXposts SET thread_id = ? WHERE (id = ? and is_top_post) or thread_id = ?", op.ThreadID, op.oldID, op.oldID,
		); err != nil {
			errEv.Err(err).Caller().
				Int("postID", op.ID).
				Int("threadID", op.ThreadID).
				Msg("Failed to set thread ID")
			return err
		}
	}
	if rows, err = m.db.QueryContextSQL(ctx, nil,
		"SELECT id,filename,filename_original,file_checksum,filesize,image_w,image_h,thumb_w,thumb_h FROM DBPREFIXposts WHERE filename <> ''",
	); err != nil {
		errEv.Err(err).Caller().Msg("Failed to get uploads")
		return err
	}
	defer rows.Close()

	var uploads []gcsql.Upload
	for rows.Next() {
		var upload gcsql.Upload
		if err = rows.Scan(&upload.PostID, &upload.Filename, &upload.OriginalFilename, &upload.Checksum, &upload.FileSize, &upload.Width,
			&upload.Height, &upload.ThumbnailWidth, &upload.ThumbnailHeight,
		); err != nil {
			errEv.Err(err).Caller().Msg("Failed to scan upload")
			return err
		}
		uploads = append(uploads, upload)
	}
	if err = rows.Close(); err != nil {
		errEv.Caller().Msg("Failed to close upload rows")
		return err
	}

	for _, upload := range uploads {
		if _, err = m.db.ExecContextSQL(ctx, nil,
			`INSERT INTO DBPREFIXfiles(post_id,file_order,filename,original_filename,checksum,file_size,width,height,thumbnail_width,thumbnail_height,is_spoilered) VALUES
			(?,0,?,?,?,?,?,?,?,?,0)`,
			upload.PostID, upload.Filename, upload.OriginalFilename, upload.Checksum, upload.FileSize, upload.Width, upload.Height,
			upload.ThumbnailWidth, upload.ThumbnailHeight,
		); err != nil {
			errEv.Err(err).Caller().
				Int("postID", upload.PostID).
				Msg("Failed to insert upload")
			return err
		}
	}

	return nil
}
