package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

func moveThread(checkedPosts []int, moveBtn string, doMove string, writer http.ResponseWriter, request *http.Request) {
	password := request.FormValue("password")
	wantsJSON := serverutil.IsRequestingJSON(request)
	if moveBtn == "Move thread" {
		// user clicked on move thread button on board or thread page

		if len(checkedPosts) == 0 {
			serverutil.ServeError(writer, "You need to select one thread to move.", wantsJSON, nil)
			return
		} else if len(checkedPosts) > 1 {
			serverutil.ServeError(writer, "You can only move one thread at a time.", wantsJSON, nil)
			return
		}
		post, err := gcsql.GetPostFromID(checkedPosts[0], true)

		if err != nil {
			serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Int("postid", checkedPosts[0]).
				Msg("Error getting post from ID")
			return
		}
		if post.ParentID != post.ID {
			serverutil.ServeError(writer, "You appear to be trying to move a post that is not the top post in the thread", wantsJSON, map[string]interface{}{
				"postid":   checkedPosts[0],
				"parentid": post.ParentID,
			})
			return
		}

		srcBoardID, err := strconv.Atoi(request.PostForm.Get("boardid"))
		if err != nil {
			serverutil.ServeError(writer, fmt.Sprintf("Invalid or missing boarid: %q", request.PostForm.Get("boardid")), wantsJSON, map[string]interface{}{
				"boardid": srcBoardID,
			})
		}
		var destBoards []gcsql.Board
		var srcBoard gcsql.Board
		for _, board := range gcsql.AllBoards {
			if board.ID != srcBoardID {
				destBoards = append(destBoards, board)
			} else {
				srcBoard = board
			}
		}
		if err = serverutil.MinifyTemplate(gctemplates.MoveThreadPage, map[string]interface{}{
			"postid":     post.ID,
			"webroot":    config.GetSystemCriticalConfig().WebRoot,
			"destBoards": destBoards,
			"srcBoard":   srcBoard,
		}, writer, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Int("postid", post.ID).Send()
			serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
	} else if doMove == "1" {
		// user got here from the move thread page
		rank := manage.GetStaffRank(request)
		if password == "" && rank == 0 {
			writer.WriteHeader(http.StatusBadRequest)
			serverutil.ServeError(writer, "Password required for post editing", wantsJSON, nil)
			return
		}
		postIDstr := request.PostForm.Get("postid")
		postID, err := strconv.Atoi(postIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			serverutil.ServeError(writer, fmt.Sprintf("Error parsing postid value: %q: %s", postIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"postid": postIDstr,
			})
			return
		}
		srcBoardIDstr := request.PostForm.Get("srcboardid")
		srcBoardID, err := strconv.Atoi(srcBoardIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			serverutil.ServeError(writer, fmt.Sprintf("Error parsing srcboardid value: %q: %s", srcBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardIDstr,
			})
			return
		}
		srcBoard, err := gcsql.GetBoardFromID(srcBoardID)
		if err != nil {
			gcutil.LogError(err).
				Int("srcboardid", srcBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardID,
			})
			return
		}

		destBoardIDstr := request.PostForm.Get("destboardid")
		destBoardID, err := strconv.Atoi(destBoardIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			serverutil.ServeError(writer, fmt.Sprintf("Error parsing destboardid value: %q: %s", destBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"destboardid": destBoardIDstr,
			})
			return
		}
		destBoard, err := gcsql.GetBoardFromID(destBoardID)
		if err != nil {
			gcutil.LogError(err).
				Int("destboardid", destBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"destboardid": destBoardID,
			})
			return
		}

		post, err := gcsql.GetPostFromID(postID, true)
		if err != nil {
			gcutil.LogError(err).Int("postid", postID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"postid": postID,
			})
			return
		}

		passwordMD5 := gcutil.Md5Sum(password)
		if passwordMD5 != post.Password && rank == 0 {
			serverutil.ServeError(writer, "Wrong password", wantsJSON, nil)
			return
		}

		threadUploads, err := getThreadFiles(post)
		if err != nil {
			gcutil.LogError(err).Int("postid", post.ID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			serverutil.ServeError(writer, "Error getting list of files in thread", wantsJSON, map[string]interface{}{
				"postid": post.ID,
			})
		}
		for _, upload := range threadUploads {
			fmt.Println("Upload post ID:", upload.postID)
			fmt.Println("Upload filename:", upload.filename)
			fmt.Println("Upload thumbnail:", gcutil.GetThumbnailPath("thumb", upload.filename))
			fmt.Println("Upload catalog thumbnail:", gcutil.GetThumbnailPath("catalog", upload.filename))

			fmt.Println()
			// threadUploads[f] = path.Join(config.GetSystemCriticalConfig().DocumentRoot, srcBoard.Dir, filename)
		}
		serverutil.ServeJSON(writer, map[string]interface{}{
			"srcBoard":  srcBoard.Dir,
			"destBoard": destBoard.Dir,
			"post":      post,
		})
	}
}

type postUpload struct {
	filename         string
	thumbnail        string
	catalogThumbnail string
	postID           int
}

func getThreadFiles(post *gcsql.Post) ([]postUpload, error) {
	query := `SELECT filename,post_id FROM DBPREFIXfiles WHERE post_id IN (
		SELECT id FROM DBPREFIXposts WHERE thread_id = (
			SELECT thread_id FROM DBPREFIXposts WHERE id = ?)) AND filename != 'deleted'`
	rows, err := gcsql.QuerySQL(query, post.ID)
	if err != nil {
		return nil, err
	}
	var uploads []postUpload
	for rows.Next() {
		var upload postUpload
		if err = rows.Scan(&upload.filename, &upload.postID); err != nil {
			return uploads, err
		}
		upload.thumbnail = gcutil.GetThumbnailPath("thumb", upload.filename)

		var parentID int
		if parentID, err = gcsql.GetThreadIDZeroIfTopPost(upload.postID); err != nil {
			return uploads, err
		}
		if parentID == 0 {
			upload.catalogThumbnail = gcutil.GetThumbnailPath("catalog", upload.filename)
		}
		uploads = append(uploads, upload)
	}
	return uploads, nil
}
