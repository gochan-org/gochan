package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
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
	if u.filename == "" || u.filename == "deleted" || strings.HasPrefix(u.filename, "embed:") {
		// no file to delete
		return nil
	}
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
	rows, cancel, err := gcsql.QueryTimeoutSQL(nil, query, params...)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		rows.Close()
		cancel()
	}()
	var posts []delPost
	var postIDsAny []any
	for rows.Next() {
		var post delPost
		if err = rows.Scan(&post.postID, &post.threadID, &post.opID, &post.isOP, &post.filename, &post.boardDir); err != nil {
			return nil, nil, err
		}
		posts = append(posts, post)
		postIDsAny = append(postIDsAny, post.postID)
	}
	return posts, postIDsAny, rows.Close()
}

func deletePosts(checkedPosts []int, writer http.ResponseWriter, request *http.Request) {
	// Delete post(s) or thread(s)
	infoEv, warnEv, errEv := gcutil.LogRequest(request)
	defer gcutil.LogDiscard(infoEv, warnEv, errEv)

	password := request.FormValue("password")
	passwordMD5 := gcutil.Md5Sum(password)
	wantsJSON := serverutil.IsRequestingJSON(request)
	contentType := "text/plain"
	if wantsJSON {
		contentType = "application/json"
	}
	writer.Header().Set("Content-Type", contentType)
	gcutil.LogBool("wantsJSON", wantsJSON, infoEv, warnEv, errEv)

	if len(checkedPosts) < 1 {
		warnEv.Msg("No posts selected")
		server.ServeError(writer, server.NewServerError("No posts selected", http.StatusBadRequest), wantsJSON, nil)
		return
	}

	staff, err := gcsql.GetStaffFromRequest(request)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		errEv.Err(err).Caller().Msg("Unable to get staff info")
		server.ServeError(writer, server.NewServerError("Unable to get staff info", http.StatusInternalServerError), wantsJSON, nil)
		return
	}

	posts := make([]any, len(checkedPosts))
	for i, p := range checkedPosts {
		posts[i] = p
	}
	fileOnly := request.FormValue("fileonly") == "on"
	gcutil.LogBool("fileOnly", fileOnly, infoEv, warnEv, errEv)
	gcutil.LogInt("affectedPosts", len(posts), infoEv, warnEv, errEv)

	if staff.Rank > 0 {
		gcutil.LogStr("staff", staff.Username, infoEv, errEv)
	} else {
		sumsMatch, err := validatePostPasswords(posts, passwordMD5)
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to validate post password checksums")
			server.ServeError(writer,
				server.NewServerError("Unable to validate post password checksums", http.StatusInternalServerError),
				wantsJSON, nil)
			return
		}
		if !sumsMatch {
			warnEv.Msg("One or more post passwords do not match")
			server.ServeError(writer,
				server.NewServerError("One or more post passwords do not match", http.StatusUnauthorized),
				wantsJSON, nil)
			return
		}
	}

	delPosts, affectedPostIDs, err := getAllPostsToDelete(posts, fileOnly)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get post info for one or more checked posts")
		server.ServeError(writer,
			server.NewServerError("Unable to get post info for one or more checked posts", http.StatusInternalServerError),
			wantsJSON, nil)
		return
	}

	boardid, err := strconv.Atoi(request.PostFormValue("boardid"))
	if err != nil {
		warnEv.Str("boardid", request.PostFormValue("boardid")).Msg("Invalid boardid value")
		server.ServeError(writer, server.NewServerError("Invalid boardid value", http.StatusBadRequest), wantsJSON, nil)
		return
	}
	errEv.Int("boardid", boardid)
	board, err := gcsql.GetBoardDir(boardid)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to get board from boardid")
		server.ServeError(writer,
			server.NewServerError("Unable to get board from boardid", http.StatusInternalServerError),
			wantsJSON, nil)
		return
	}

	// delete files, leaving the filename in the db as 'deleted' if the post should remain
	if !deletePostFiles(delPosts, affectedPostIDs, !fileOnly, request, writer, errEv) {
		return
	}
	if !fileOnly && !markPostsAsDeleted(affectedPostIDs, request, writer, errEv) {
		// markPostsAsDeleted logs any errors
		return
	}

	if err = building.BuildBoards(false, boardid); err != nil {
		// BuildBoards logs any errors
		server.ServeError(writer, server.NewServerError("Unable to rebuild /"+board+"/", http.StatusInternalServerError), wantsJSON, nil)
		return
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

	err := gcsql.QueryRow(nil, queryPosts, params, []any{&count})
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
	opts := &gcsql.RequestOptions{Context: ctx, Tx: tx, Cancel: cancel}

	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to start deletion transaction")
		server.ServeError(writer, server.NewServerError("Unable to start deletion transaction", http.StatusInternalServerError), wantsJSON, nil)
		return false
	}
	defer tx.Rollback()
	const postsError = "Unable to delete post(s)"
	const threadsError = "Unable to delete thread(s)"
	if _, err = gcsql.Exec(opts, deletePostsSQL, posts...); err != nil {
		errEv.Err(err).Caller().Msg("Unable to mark post(s) as deleted")
		server.ServeError(writer, server.NewServerError(postsError, http.StatusInternalServerError), wantsJSON, nil)
		return false
	}

	if _, err = gcsql.Exec(opts, deleteThreadSQL, posts...); err != nil {
		errEv.Err(err).Caller().Msg("Unable to mark thread(s) as deleted")
		server.ServeError(writer, server.NewServerError(threadsError, http.StatusInternalServerError), wantsJSON, nil)
		return false
	}

	if err = tx.Commit(); err != nil {
		errEv.Err(err).Caller().Msg("Unable to commit deletion transaction")
		server.ServeError(writer, server.NewServerError("Unable to finalize deletion", http.StatusInternalServerError), wantsJSON, nil)
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
	wantsJSON := serverutil.IsRequestingJSON(request)

	errArr := zerolog.Arr()
	var err error
	var tmpErr error
	for _, post := range posts {
		if tmpErr = post.deleteFile(permDelete); tmpErr != nil {
			gcutil.LogError(tmpErr).Err(tmpErr).Caller().
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
		errEv.Array("errors", errArr).Caller().Msg("Received 1 or more errors while trying to delete post files")
		server.ServeError(writer,
			server.NewServerError("Received 1 or more errors while trying to delete post files", http.StatusInternalServerError),
			wantsJSON, nil)
		return false
	}

	if permDelete {
		_, err = gcsql.ExecTimeoutSQL(nil, "DELETE FROM DBPREFIXfiles WHERE post_id IN "+params, deleteIDs...)
	} else {
		_, err = gcsql.ExecTimeoutSQL(nil, "DELETE FROM DBPREFIXfiles WHERE post_id IN "+params+" AND filename like 'embed:%'", deleteIDs...)
		if err == nil {
			_, err = gcsql.ExecTimeoutSQL(nil, "UPDATE DBPREFIXfiles SET filename = 'deleted', original_filename = 'deleted' WHERE post_id in "+params, deleteIDs...)
		}
	}
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to delete file entries from database")
		server.ServeError(writer,
			server.NewServerError("Unable to delete file entries from database", http.StatusInternalServerError),
			wantsJSON, nil)
		return false
	}
	return true
}
