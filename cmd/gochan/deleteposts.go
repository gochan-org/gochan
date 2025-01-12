package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

type delPost struct {
	postID   int
	threadID int
	opID     int
	isOP     bool
	filename string
	boardDir string
}

func (u *delPost) filePath() string {
	if u.filename == "" || u.filename == "deleted" {
		return ""
	}
	return path.Join(config.GetSystemCriticalConfig().DocumentRoot, u.boardDir, "src", u.filename)
}

func (u *delPost) thumbnailPaths() (string, string) {
	if u.filename == "" || u.filename == "deleted" {
		return "", ""
	}
	return uploads.GetThumbnailFilenames(path.Join(
		config.GetSystemCriticalConfig().DocumentRoot,
		u.boardDir, "thumb", u.filename))
}

func coalesceErrors(errs ...error) error {
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
	var errCatalog, errThumb, errFile, errThread, errJSON error
	var wg sync.WaitGroup
	wg.Add(2)
	file := u.filePath()
	thumb, catalogThumb := u.thumbnailPaths()
	if u.isOP {
		wg.Add(1)
		go func() {
			if catalogThumb != "" {
				errCatalog = os.Remove(catalogThumb)
			}
			wg.Done()
		}()
		if delThread && u.isOP {
			wg.Add(2)
			threadBase := path.Join(config.GetSystemCriticalConfig().DocumentRoot,
				u.boardDir, "res", strconv.Itoa(u.postID))
			go func() {
				errThread = os.Remove(threadBase + ".html")
				wg.Done()
			}()
			go func() {
				errJSON = os.Remove(threadBase + ".json")
				wg.Done()
			}()
		}
	}
	go func() {
		if thumb != "" {
			errThumb = os.Remove(thumb)
		}
		wg.Done()
	}()
	go func() {
		if file != "" {
			errFile = os.Remove(file)
		}
		wg.Done()
	}()
	wg.Wait()
	return coalesceErrors(errThread, errJSON, errCatalog, errThumb, errFile)
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
	var query string
	params := postIDs
	if fileOnly {
		// only deleting this post's file, not subfiles if it's an OP
		query = "SELECT post_id, thread_id, op_id, is_top_post, filename, dir FROM DBPREFIXv_posts_to_delete_file_only WHERE post_id IN " + setPart
	} else {
		// deleting everything, including subfiles
		params = append(params, postIDs...)
		query = "SELECT post_id, thread_id, op_id, is_top_post, filename, dir FROM DBPREFIXv_posts_to_delete WHERE post_id IN " +
			setPart + " OR thread_id IN (SELECT thread_id from DBPREFIXposts op WHERE op_id IN " + setPart + " AND is_top_post)"
	}
	rows, err := gcsql.QuerySQL(query, params...)
	if err != nil {
		return nil, nil, err
	}
	var posts []delPost
	var postIDsAny []any
	for rows.Next() {
		var post delPost
		if err = rows.Scan(&post.postID, &post.threadID, &post.opID, &post.isOP, &post.filename, &post.boardDir); err != nil {
			rows.Close()
			return nil, nil, err
		}
		posts = append(posts, post)
		postIDsAny = append(postIDsAny, post.postID)
	}
	return posts, postIDsAny, rows.Close()
}

func serveError(writer http.ResponseWriter, errStr string, statusCode int, wantsJSON bool, errEv *zerolog.Event) {
	if errEv != nil {
		errEv.Msg(errStr)
	}
	writer.WriteHeader(statusCode)
	server.ServeError(writer, errStr, wantsJSON, nil)
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
		serveError(writer, "No posts selected", http.StatusBadRequest, wantsJSON, errEv.Err(nil).Caller())
		return
	}

	staff, err := gcsql.GetStaffFromRequest(request)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		serveError(writer, "Unable to get staff info", http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
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
			serveError(writer, "Unable to validate post password checksums",
				http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
			return
		}
		if !sumsMatch {
			serveError(writer, "One or more post passwords do not match", http.StatusUnauthorized, wantsJSON, errEv.Caller())
			return
		}
	}

	delPosts, affectedPostIDs, err := getAllPostsToDelete(posts, fileOnly)
	if err != nil {
		serveError(writer, "Unable to get post info for one or more checked posts",
			http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
		return
	}

	boardid, err := strconv.Atoi(request.FormValue("boardid"))
	if err != nil {
		serveError(writer, "Invalid boardid value", http.StatusBadRequest, wantsJSON, errEv.Err(err).Caller().
			Str("boardid", request.FormValue("boardid")))
		return
	}
	errEv.Int("boardid", boardid)
	board, err := gcsql.GetBoardDir(boardid)
	if err != nil {
		serveError(writer, "Unable to get board from boardid", http.StatusInternalServerError,
			wantsJSON, errEv.Err(err).Caller())
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
		serveError(writer, fmt.Sprintf("Unable to rebuild /%s/", board),
			http.StatusInternalServerError, wantsJSON, nil)
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
	sqlCfg := config.GetSQLConfig()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sqlCfg.DBTimeoutSeconds)*time.Second)
	defer cancel()

	tx, err := gcsql.BeginContextTx(ctx)

	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		serveError(writer, "Unable to delete posts", http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
		return false
	}
	defer tx.Rollback()
	const postsError = "Unable to mark post(s) as deleted"
	const threadsError = "Unable to mark thread(s) as deleted"
	if _, err = gcsql.ExecTxSQL(tx, deletePostsSQL, posts...); err != nil {
		serveError(writer, postsError, http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
		return false
	}

	if _, err = gcsql.ExecTxSQL(tx, deleteThreadSQL, posts...); err != nil {
		serveError(writer, threadsError, http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
		return false
	}

	if err = tx.Commit(); err != nil {
		errEv.Err(err).Caller().Msg("Unable to commit deletion transaction")
		serveError(writer, "Unable to finalize deletion", http.StatusInternalServerError, wantsJSON, nil)
		return false
	}

	return true
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
	deleteFilesSQL := `UPDATE DBPREFIXfiles SET filename = 'deleted', original_filename = 'deleted' WHERE post_id in `
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
			gcutil.LogWarning().Err(tmpErr).Caller().
				Int("postID", post.postID).
				Int("opID", post.opID).
				Str("filename", post.filename).
				Str("board", post.boardDir).
				Msg("Got error when trying to delete file")
			if os.IsNotExist(tmpErr) {
				tmpErr = nil
			} else {
				errArr.Int(post.postID).Err(err)
				if err == nil {
					err = tmpErr
				}
			}
		}
	}
	if err != nil {
		serveError(writer, "Received 1 or more errors while trying to delete post files",
			http.StatusInternalServerError, wantsJSON, errEv.Array("errors", errArr))
		return false
	}
	_, err = gcsql.ExecSQL(deleteFilesSQL, deleteIDs...)
	if err != nil {
		serveError(writer, "Unable to delete file entries from database",
			http.StatusInternalServerError, wantsJSON, errEv.Err(err).Caller())
		return false
	}
	return true
}
