package main

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

type upload struct {
	// postID   int
	filename string
	boardDir string
}

func (u *upload) filePath() string {
	return path.Join(config.GetSystemCriticalConfig().DocumentRoot, u.boardDir, "src", u.filename)
}

func (u *upload) thumbnailPaths() (string, string) {
	return uploads.GetThumbnailFilenames(path.Join(
		config.GetSystemCriticalConfig().DocumentRoot,
		u.boardDir, "thumb", u.filename))
}

func deletePosts(checkedPosts []int, writer http.ResponseWriter, request *http.Request) {
	// Delete a post or thread
	infoEv, errEv := gcutil.LogRequest(request)
	defer func() {
		gcutil.LogDiscard(infoEv, errEv)
	}()

	password := request.FormValue("password")
	passwordMD5 := gcutil.Md5Sum(password)
	wantsJSON := serverutil.IsRequestingJSON(request)
	if wantsJSON {
		writer.Header().Set("Content-Type", "application/json")
	} else {
		writer.Header().Set("Content-Type", "text/plain")
	}
	gcutil.LogBool("wantsJSON", wantsJSON, infoEv, errEv)

	if len(checkedPosts) < 1 {
		errEv.Err(errors.New("no posts selected")).Caller().Send()
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, "No posts selected", wantsJSON, nil)
		return
	}

	staff, err := manage.GetStaffFromRequest(request)
	if err != nil {
		errEv.Err(err).Caller().
			Msg("unable to get staff info")
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, err.Error(), wantsJSON, nil)
	}
	if staff.Rank > 0 {
		gcutil.LogStr("staff", staff.Username, infoEv, errEv)
	} else {
		sumsMatch, err := validatePostPasswords(checkedPosts, passwordMD5)
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Unable to validate post password checksums")
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer,
				"Unable to validate post password checksums (DB error)",
				wantsJSON, nil)
			return
		}
		if !sumsMatch {
			errEv.Caller().
				Msg("One or more post password checksums do not match")
			writer.WriteHeader(http.StatusUnauthorized)
			server.ServeError(writer, "One or more post passwords do not match", wantsJSON, nil)
			return
		}
	}

	fileOnly := request.FormValue("fileonly") == "on"

	boardid, err := strconv.Atoi(request.FormValue("boardid"))
	if err != nil {
		errEv.Err(err).Caller().
			Str("boardid", request.FormValue("boardid")).
			Msg("Invalid form data (boardid)")
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	errEv.Int("boardid", boardid)
	board, err := gcsql.GetBoardDir(boardid)
	if err != nil {
		server.ServeError(writer, "Unable to get board from boardid", wantsJSON, map[string]any{
			"boardid": boardid,
		})
		errEv.Err(err).Caller().Send()
		return
	}

	// delete files, leaving the filename in the db as 'deleted' if the post should remain
	if !deletePostFiles(checkedPosts, !fileOnly, request, writer, errEv) {
		return
	}
	if !fileOnly && !markPostsAsDeleted(checkedPosts, request, writer, errEv) {
		return
	}

	// deletion completed, redirect to board
	http.Redirect(writer, request, config.WebPath(board), http.StatusFound)
}

// should return true if all posts have the same password checksum
func validatePostPasswords(posts []int, passwordMD5 string) (bool, error) {
	var count int
	err := gcsql.QueryRowSQL(`SELECT COUNT(*) FROM DBPREFIXposts WHERE password = ?`,
		[]any{passwordMD5}, []any{&count})
	return count == len(posts), err
}

func markPostsAsDeleted(posts []int, request *http.Request, writer http.ResponseWriter, errEv *zerolog.Event) bool {
	deletePostsSQL := `UPDATE DBPREFIXposts SET is_deleted = TRUE WHERE id IN (`
	deleteThreadSQL := `UPDATE DBPREFIXthreads SET is_deleted = TRUE WHERE id in (
		SELECT thread_id FROM DBPREFIXposts WHERE is_top_post AND id in (`
	var postsInterfaceArr []any
	postsLen := len(posts)
	for p := range posts {
		if p < postsLen-1 {
			deletePostsSQL += "?,"
			deleteThreadSQL += "?,"
		} else {
			deletePostsSQL += "?)"
			deleteThreadSQL += "?))"
		}
		postsInterfaceArr = append(postsInterfaceArr, posts[p])
	}
	tx, err := gcsql.BeginTx()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Unable to delete posts (DB error)", wantsJSON, nil)
		return false
	}
	defer tx.Rollback()

	if _, err = gcsql.ExecTxSQL(tx, deletePostsSQL, postsInterfaceArr...); err != nil {
		errEv.Err(err).Caller().Msg("Unable to mark posts as deleted")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to mark posts as deleted (DB error)", wantsJSON, nil)
		return false
	}

	if _, err = gcsql.ExecTxSQL(tx, deleteThreadSQL, postsInterfaceArr...); err != nil {
		errEv.Err(err).Caller().Msg("Unable to mark threads as deleted")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to mark threads as deleted (DB error)", wantsJSON, nil)
		return false
	}

	if err = tx.Commit(); err != nil {
		errEv.Err(err).Caller().Msg("Unable to commit deletion transaction")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to finalize deletion", wantsJSON, nil)
		return false
	}
	return true
}

func deleteFile(file *upload, wg *sync.WaitGroup, errArr *zerolog.Array, errEv *zerolog.Event) error {
	wg.Add(3)
	var err error
	var errTmp error
	go func() {
		filePath := file.filePath()
		if errTmp = os.Remove(filePath); errTmp != nil {
			errEv.Caller()
			errArr.Err(errTmp)
			if err == nil {
				err = errTmp
			}
		}
		wg.Done()
	}()
	thumbPath, catalogThumbPath := file.thumbnailPaths()
	go func() {
		if errTmp = os.Remove(thumbPath); errTmp != nil {
			errEv.Caller()
			errArr.Err(errTmp)
			if err == nil {
				err = errTmp
			}
		}
		wg.Done()
	}()
	go func() {
		if errTmp = os.Remove(catalogThumbPath); errTmp != nil {
			errEv.Caller()
			errArr.Err(errTmp)
			if err == nil {
				err = errTmp
			}
		}
		wg.Done()
	}()
	return err
}

func deletePostFiles(posts []int, permDelete bool, request *http.Request, writer http.ResponseWriter, errEv *zerolog.Event) bool {
	queryFilenamesSQL := `SELECT post_id,filename,dir,password
	FROM DBPREFIXboards b
	LEFT JOIN DBPREFIXthreads t ON b.id = t.board_id
	LEFT JOIN DBPREFIXposts p ON t.id = p.thread_id
	LEFT JOIN DBPREFIXfiles f ON p.id = f.post_id
	AND p.is_deleted = false
	AND t.is_deleted = false
	AND p.id in (`

	deleteFilesSQL := `UPDATE DBPREFIXfiles SET filename = 'deleted', original_filename = 'deleted' WHERE post_id in (`
	if permDelete {
		deleteFilesSQL = `DELETE FROM DBPREFIXfiles WHERE post_id IN (`
	}

	numChecked := len(posts)
	for i := range posts {
		if i < numChecked-1 {
			deleteFilesSQL += "?,"
			queryFilenamesSQL += "?,"
		} else {
			deleteFilesSQL += "?)"
			queryFilenamesSQL += "?)"
		}
	}
	wantsJSON := serverutil.IsRequestingJSON(request)

	rows, err := gcsql.QuerySQL(queryFilenamesSQL, posts)
	if err != nil {
		errEv.Err(err).Caller().Msg("unable to get files to be deleted")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to get files to be deleted from database", wantsJSON, map[string]any{
			"numPosts": len(posts),
		})
		return false
	}

	errArr := zerolog.Arr()
	var wg sync.WaitGroup
	var toDelete []upload
	var deleteIDs []any
	for rows.Next() {
		var postID int
		var file upload
		var postPasswordMD5 string
		if err = rows.Scan(&postID, &file.filename, &file.boardDir, &postPasswordMD5); err != nil {
			errEv.Err(err).Caller().Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Unable to scan results from database row", wantsJSON, nil)
			return false
		}
		toDelete = append(toDelete, file)
		deleteIDs = append(deleteIDs, postID)
	}

	var errTmp error
	for f := range toDelete {
		errTmp = deleteFile(&toDelete[f], &wg, errArr, errEv)
		if err == nil {
			err = errTmp
		}
	}
	wg.Wait()
	if err != nil {
		errEv.Array("errors", errArr).Msg("Received 1 or more errors while trying to delete files")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Received 1 or more errors while trying to delete post files", wantsJSON, nil)
		return false
	}
	_, err = gcsql.ExecSQL(deleteFilesSQL, deleteIDs...)
	if err != nil {
		errEv.Err(err).Caller().Send()
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to delete file entries from database", wantsJSON, nil)
		return false
	}
	return true
}
