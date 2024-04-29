package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

type delPost struct {
	postID   int
	opID     int
	threadID int
	isOP     bool
	filename string
	boardDir string
}

func (u *delPost) filePath() string {
	if u.filename == "" {
		return ""
	}
	return path.Join(config.GetSystemCriticalConfig().DocumentRoot, u.boardDir, "src", u.filename)
}

func (u *delPost) thumbnailPaths() (string, string) {
	if u.filename == "" {
		return "", ""
	}
	return uploads.GetThumbnailFilenames(path.Join(
		config.GetSystemCriticalConfig().DocumentRoot,
		u.boardDir, "thumb", u.filename))
}

func firstNonNilError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// deleteFile asynchronously deletes the post's file and thumb (if it has one, it returns nil if not) and
// thread HTML file if it is an OP and "File only" is unchecked, returning an error if one occcured for any file
func (u *delPost) deleteFile(delThread bool) error {
	if u.filename == "" || u.filename == "deleted" {
		return nil
	}
	var errCatalog, errThumb, errFile, errThread error
	var wg sync.WaitGroup
	wg.Add(2)
	thumb, catalogThumb := u.thumbnailPaths()
	if u.isOP {
		wg.Add(1)
		go func() {
			errCatalog = os.Remove(catalogThumb)
			wg.Done()
		}()
		if delThread {
			wg.Add(1)
			go func() {
				threadPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot,
					u.boardDir, "res", fmt.Sprintf("%d.html", u.postID))
				errThread = os.Remove(threadPath)
				wg.Done()
			}()
		}
	}
	go func() {
		errThumb = os.Remove(thumb)
		wg.Done()
	}()
	go func() {
		errFile = os.Remove(u.filePath())
		wg.Done()
	}()
	wg.Wait()
	return firstNonNilError(errThread, errCatalog, errThumb, errFile)
}

// getAllPostsToDelete returns all of the posts and their respective filenames that would be affected by deleting
// the selected posts (including replies if a thread OP is selected). It returns an array of objects representing
// each ID and filename (if there is one), as well as an array of interfaces for future query parameters
func getAllPostsToDelete(postIDs []any, fileOnly bool) ([]delPost, []any, error) {
	setPart := "("
	for i := range postIDs {
		if i < len(postIDs)-1 {
			setPart += "?,"
		} else {
			setPart += "?)"
		}
	}
	query := `SELECT p.id AS postid, (
		SELECT op.id AS opid FROM DBPREFIXposts op
		WHERE op.thread_id = p.thread_id AND is_top_post LIMIT 1
	) as opid, is_top_post, COALESCE(filename, "") AS filename, dir
	FROM DBPREFIXboards b
	LEFT JOIN DBPREFIXthreads t ON t.board_id = b.id
	LEFT JOIN DBPREFIXposts p ON p.thread_id = t.id
	LEFT JOIN DBPREFIXfiles f ON f.post_id = p.id
	WHERE p.id IN ` + setPart + ` OR p.thread_id IN (
		SELECT thread_id from DBPREFIXposts op WHERE op.id IN ` + setPart + ` AND is_top_post)`
	if fileOnly {
		query += " AND filename IS NOT NULL"
	}
	params := append(postIDs, postIDs...)
	rows, err := gcsql.QuerySQL(query, params...)
	if err != nil {
		return nil, nil, err
	}
	var posts []delPost
	var postIDsAny []any
	for rows.Next() {
		var post delPost
		if err = rows.Scan(&post.postID, &post.opID, &post.isOP, &post.filename, &post.boardDir); err != nil {
			rows.Close()
			return nil, nil, err
		}
		posts = append(posts, post)
		postIDsAny = append(postIDsAny, post.postID)
	}
	return posts, postIDsAny, rows.Close()
}

func deletePosts(checkedPosts []int, writer http.ResponseWriter, request *http.Request) {
	// Delete post(s) or thread(s)
	infoEv, errEv := gcutil.LogRequest(request)
	defer gcutil.LogDiscard(infoEv, errEv)

	password := request.FormValue("password")
	passwordMD5 := gcutil.Md5Sum(password)
	wantsJSON := serverutil.IsRequestingJSON(request)
	contentType := "text/plain"
	if wantsJSON {
		contentType = "application/json"
	}
	writer.Header().Set("Content-Type", contentType)
	gcutil.LogBool("wantsJSON", wantsJSON, infoEv, errEv)

	if len(checkedPosts) < 1 {
		errEv.Err(errors.New("no posts selected")).Caller().Send()
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, "No posts selected", wantsJSON, nil)
		return
	}

	staff, err := manage.GetStaffFromRequest(request)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		errEv.Err(err).Caller().
			Msg("unable to get staff info")
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	posts := make([]any, len(checkedPosts))
	for i, p := range checkedPosts {
		posts[i] = p
	}
	fileOnly := request.FormValue("fileonly") == "on"
	gcutil.LogBool("fileOnly", fileOnly, infoEv, errEv)
	gcutil.LogInt("affectedPosts", len(posts), infoEv, errEv)

	if staff.Rank > 0 {
		gcutil.LogStr("staff", staff.Username, infoEv, errEv)
	} else {
		sumsMatch, err := validatePostPasswords(posts, passwordMD5)
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

	delPosts, affectedPostIDs, err := getAllPostsToDelete(posts, fileOnly)
	if err != nil {
		errEv.Err(err).Caller().Send()
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to get post info for one or more checked posts", wantsJSON, nil)
		return
	}

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
	if !deletePostFiles(delPosts, affectedPostIDs, !fileOnly, request, writer, errEv) {
		return
	}
	if !fileOnly && !markPostsAsDeleted(affectedPostIDs, request, writer, errEv) {
		return
	}

	if err = building.BuildBoards(false, boardid); err != nil {
		// BuildBoards logs any errors
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to rebuild /"+board+"/", wantsJSON, map[string]any{
			"board": board,
		})
	}
	if fileOnly {
		infoEv.Msg("file(s) deleted")
	} else {
		infoEv.Msg("post(s) deleted")
	}

	// deletion completed, redirect to board
	http.Redirect(writer, request, config.WebPath(board), http.StatusFound)
}

// should return true if all posts have the same password checksum
func validatePostPasswords(posts []any, passwordMD5 string) (bool, error) {
	var count int
	queryPosts := `SELECT COUNT(*) FROM DBPREFIXposts WHERE password = ? AND id in (`
	numPosts := len(posts)
	params := []any{passwordMD5}
	for p := range posts {
		if p < numPosts-1 {
			queryPosts += "?,"
		} else {
			queryPosts += "?)"
		}
		params = append(params, posts[p])
	}

	err := gcsql.QueryRowSQL(queryPosts, params, []any{&count})
	return count == len(posts), err
}

func markPostsAsDeleted(posts []any, request *http.Request, writer http.ResponseWriter, errEv *zerolog.Event) bool {
	deletePostsSQL := `UPDATE DBPREFIXposts SET is_deleted = TRUE WHERE id IN (`
	deleteThreadSQL := `UPDATE DBPREFIXthreads SET is_deleted = TRUE WHERE id in (
		SELECT thread_id FROM DBPREFIXposts WHERE is_top_post AND id in (`
	postsLen := len(posts)
	for p := range posts {
		if p < postsLen-1 {
			deletePostsSQL += "?,"
			deleteThreadSQL += "?,"
		} else {
			deletePostsSQL += "?)"
			deleteThreadSQL += "?))"
		}
	}
	tx, err := gcsql.BeginTx()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Unable to delete posts (DB error)", wantsJSON, nil)
		return false
	}
	defer tx.Rollback()

	if _, err = gcsql.ExecTxSQL(tx, deletePostsSQL, posts...); err != nil {
		errEv.Err(err).Caller().Msg("Unable to mark posts as deleted")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to mark posts as deleted (DB error)", wantsJSON, nil)
		return false
	}

	if _, err = gcsql.ExecTxSQL(tx, deleteThreadSQL, posts...); err != nil {
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

func deleteFile(file *delPost, wg *sync.WaitGroup, errArr *zerolog.Array, errEv *zerolog.Event) error {
	if file.isOP {
		wg.Add(3)
	} else {
		wg.Add(2)
	}
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
	if file.isOP {
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
	}
	return err
}

func deletePostFiles(posts []delPost, deleteIDs []any, permDelete bool, request *http.Request, writer http.ResponseWriter, errEv *zerolog.Event) bool {
	params := "("
	for i := range posts {
		if i < len(posts)-1 {
			params += "?,"
		} else {
			params += "?)"
		}
	}
	deleteFilesSQL := `UPDATE DBPREFIXfiles SET filename = 'deleted', original_filename = 'deleted' WHERE post_id in (`
	if permDelete {
		deleteFilesSQL = `DELETE FROM DBPREFIXfiles WHERE post_id IN `
	}
	deleteFilesSQL += params
	wantsJSON := serverutil.IsRequestingJSON(request)

	errArr := zerolog.Arr()
	var err error
	var tmpErr error
	for _, post := range posts {
		if tmpErr = post.deleteFile(permDelete); tmpErr != nil {
			errArr.Int(post.postID).Err(err)
			if err == nil {
				err = tmpErr
			}
		}
	}
	if err != nil {
		errEv.Array("errors", errArr).Msg("Received 1 or more errors while trying to delete files")
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Received 1 or more errors while trying to delete post files", wantsJSON, nil)
		return false
	}
	_, err = gcsql.ExecSQL(deleteFilesSQL, deleteIDs...)
	if err != nil {
		fmt.Println(deleteFilesSQL)
		errEv.Err(err).Caller().Send()
		writer.WriteHeader(http.StatusInternalServerError)
		server.ServeError(writer, "Unable to delete file entries from database", wantsJSON, nil)
		return false
	}
	return true
}
