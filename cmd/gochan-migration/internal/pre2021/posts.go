package pre2021

import (
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

type postTable struct {
	gcsql.Post
	// id               int
	// boardID          int
	// parentID         int
	// name             string
	// tripcode         string
	// email            string
	// subject          string
	// message          string
	// messageRaw       string
	// password         string
	filename         string
	filenameOriginal string
	fileChecksum     string
	filesize         int
	imageW           int
	imageH           int
	thumbW           int
	thumbH           int
	// ip               string
	// tag              string
	// timestamp        time.Time
	autosage bool
	bumped   time.Time
	stickied bool
	locked   bool
	// reviewed         bool
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

		if thread.ThreadID, err = gcsql.CreateThread(tx, thread.boardID, thread.locked, thread.stickied, thread.autosage, false); err != nil {
			errEv.Err(err).Caller().
				Int("boardID", thread.boardID).
				Msg("Failed to create thread")
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
