package pre2021

import (
	"database/sql"
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

func (*Pre2021Migrator) migratePost(tx *sql.Tx, post *migrationPost, errEv *zerolog.Event) error {
	var err error
	opts := &gcsql.RequestOptions{Tx: tx}
	if post.oldParentID == 0 {
		// migrating post was a thread OP, create the row in the threads table
		if post.ThreadID, err = gcsql.CreateThread(opts, post.boardID, false, post.stickied, post.autosage, false); err != nil {
			errEv.Err(err).Caller().
				Int("boardID", post.boardID).
				Msg("Failed to create thread")
		}
	}

	// insert thread top post
	if err = post.Insert(true, post.boardID, false, post.stickied, post.autosage, false, opts); err != nil {
		errEv.Err(err).Caller().
			Int("boardID", post.boardID).
			Int("threadID", post.ThreadID).
			Msg("Failed to insert thread OP")
	}

	if post.filename != "" {
		if err = post.AttachFile(&gcsql.Upload{
			PostID:           post.ID,
			OriginalFilename: post.filenameOriginal,
			Filename:         post.filename,
			Checksum:         post.fileChecksum,
			FileSize:         post.filesize,
			ThumbnailWidth:   post.thumbW,
			ThumbnailHeight:  post.thumbH,
			Width:            post.imageW,
			Height:           post.imageH,
		}, opts); err != nil {
			errEv.Err(err).Caller().
				Int("oldPostID", post.oldID).
				Msg("Failed to attach upload to migrated post")
			return err
		}
	}
	return nil
}

func (m *Pre2021Migrator) MigratePosts() error {
	errEv := common.LogError()
	defer errEv.Discard()

	tx, err := gcsql.BeginTx()
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed to start transaction")
		return err
	}
	defer tx.Rollback()

	rows, err := m.db.Query(nil, threadsQuery)
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
		replyRows, err := m.db.Query(nil, postsQuery+" AND parentid = ?", thread.oldID)
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
			if _, err = gcsql.Exec(&gcsql.RequestOptions{Tx: tx}, "UPDATE DBPREFIXthreads SET locked = TRUE WHERE id = ?", thread.ThreadID); err != nil {
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
