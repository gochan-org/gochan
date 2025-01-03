package pre2021

import (
	"context"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type postTable struct {
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
	for rows.Next() {
		var thread postTable
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

		// create the thread as not locked so migration replies can be inserted. It will be locked after they are all inserted
		if thread.ThreadID, err = gcsql.CreateThread(tx, thread.boardID, false, thread.stickied, thread.autosage, false); err != nil {
			errEv.Err(err).Caller().
				Int("boardID", thread.boardID).
				Msg("Failed to create thread")
		}

		// insert thread top post
		if err = thread.InsertWithContext(context.Background(), tx, true, thread.boardID, false, thread.stickied, thread.autosage, false); err != nil {
			errEv.Err(err).Caller().
				Int("boardID", thread.boardID).
				Int("threadID", thread.ThreadID).
				Msg("Failed to insert thread OP")
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
			var reply postTable
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
			if err = reply.InsertWithContext(context.Background(), tx, true, reply.boardID, false, false, false, false); err != nil {
				errEv.Err(err).Caller().
					Int("parentID", thread.oldID).
					Msg("Failed to insert reply post")
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
	}
	return err
}

func (m *Pre2021Migrator) migratePostsInPlace() error {
	return common.NewMigrationError("pre2021", "not yet implemented")
}
