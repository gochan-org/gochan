package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func moveThread(checkedPosts []int, moveBtn string, doMove string, writer http.ResponseWriter, request *http.Request) {
	password := request.FormValue("postpassword")
	wantsJSON := serverutil.IsRequestingJSON(request)
	if moveBtn == "Move thread" {
		// user clicked on move thread button on board or thread page

		if len(checkedPosts) == 0 {
			server.ServeError(writer, "You need to select one thread to move.", wantsJSON, nil)
			return
		} else if len(checkedPosts) > 1 {
			server.ServeError(writer, "You can only move one thread at a time.", wantsJSON, nil)
			return
		}
		post, err := gcsql.GetPostFromID(checkedPosts[0], true)

		if err != nil {
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Int("postid", checkedPosts[0]).
				Msg("Error getting post from ID")
			return
		}
		if !post.IsTopPost {
			topPostID, err := post.TopPostID()
			if err != nil {
				server.ServeError(writer, "Unable to get top post ID: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				gcutil.LogError(err).
					Str("IP", gcutil.GetRealIP(request)).
					Int("postid", post.ID).
					Msg("Unable to get top post ID")
				return
			}
			server.ServeError(writer, "You appear to be trying to move a post that is not the top post in the thread", wantsJSON, map[string]interface{}{
				"postid":  checkedPosts[0],
				"toppost": topPostID,
			})
			return
		}

		srcBoardID, err := strconv.Atoi(request.PostForm.Get("boardid"))
		if err != nil {
			server.ServeError(writer, fmt.Sprintf("Invalid or missing boarid: %q", request.PostForm.Get("boardid")), wantsJSON, map[string]interface{}{
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
			"boardConfig": config.GetBoardConfig(srcBoard.Dir),
			"postid":      post.ID,
			"destBoards":  destBoards,
			"pageTitle":   fmt.Sprintf("Move thread #%d", post.ID),
			"srcBoard":    srcBoard,
		}, writer, "text/html"); err != nil {
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Int("postid", post.ID).Send()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
	} else if doMove == "1" {
		// user got here from the move thread page
		rank := manage.GetStaffRank(request)
		if password == "" && rank == 0 {
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, "Password required for post moving", wantsJSON, nil)
			return
		}
		postIDstr := request.PostForm.Get("postid")
		postID, err := strconv.Atoi(postIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing postid value: %q: %s", postIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"postid": postIDstr,
			})
			return
		}
		srcBoardIDstr := request.PostForm.Get("srcboardid")
		srcBoardID, err := strconv.Atoi(srcBoardIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing srcboardid value: %q: %s", srcBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardIDstr,
			})
			return
		}
		srcBoard, err := gcsql.GetBoardFromID(srcBoardID)
		if err != nil {
			gcutil.LogError(err).
				Int("srcboardid", srcBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardID,
			})
			return
		}

		destBoardIDstr := request.PostForm.Get("destboardid")
		destBoardID, err := strconv.Atoi(destBoardIDstr)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing destboardid value: %q: %s", destBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"destboardid": destBoardIDstr,
			})
			return
		}
		destBoard, err := gcsql.GetBoardFromID(destBoardID)
		if err != nil {
			gcutil.LogError(err).
				Int("destboardid", destBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"destboardid": destBoardID,
			})
			return
		}

		post, err := gcsql.GetPostFromID(postID, true)
		if err != nil {
			gcutil.LogError(err).Int("postid", postID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"postid": postID,
			})
			return
		}

		passwordMD5 := gcutil.Md5Sum(password)
		if passwordMD5 != post.Password && rank == 0 {
			server.ServeError(writer, "Wrong password", wantsJSON, nil)
			return
		}

		if err = post.ChangeBoardID(destBoardID); err != nil {
			gcutil.LogError(err).
				Int("postID", postID).
				Int("destBoardID", destBoardID).
				Msg("Failed changing thread board ID")
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"postID":      postID,
				"destBoardID": destBoardID,
			})
			return
		}
		threadUploads, err := gcsql.GetThreadFiles(post)
		if err != nil {
			gcutil.LogError(err).Int("postid", post.ID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Error getting list of files in thread", wantsJSON, map[string]interface{}{
				"postid": post.ID,
			})
		}
		documentRoot := config.GetSystemCriticalConfig().DocumentRoot
		for _, upload := range threadUploads {
			// move the upload itself
			tmpErr := moveFileIfExists(
				path.Join(documentRoot, srcBoard.Dir, "src", upload.Filename),
				path.Join(documentRoot, destBoard.Dir, "src", upload.Filename))
			if tmpErr != nil {
				gcutil.LogError(err).
					Str("filename", upload.Filename).
					Str("srcBoard", srcBoard.Dir).
					Str("destBoard", destBoard.Dir).
					Msg("Unable to move file from source board to destination board")
				if err == nil {
					// log all errors but only report the first one to the user
					err = tmpErr
				}
			}

			// move the upload thumbnail
			if tmpErr = moveFileIfExists(
				path.Join(documentRoot, srcBoard.Dir, "thumb", upload.ThumbnailPath("upload")),
				path.Join(documentRoot, destBoard.Dir, "thumb", upload.ThumbnailPath("upload")),
			); tmpErr != nil {
				gcutil.LogError(err).
					Str("thumbnail", upload.ThumbnailPath("upload")).
					Str("srcBoard", srcBoard.Dir).
					Str("destBoard", destBoard.Dir).
					Msg("Unable to move thumbnail from source board to destination board")
				if err == nil {
					err = tmpErr
				}
			}
			if upload.PostID == post.ID {
				// move the upload catalog thumbnail
				if tmpErr = moveFileIfExists(
					path.Join(documentRoot, srcBoard.Dir, "thumb", upload.ThumbnailPath("catalog")),
					path.Join(documentRoot, destBoard.Dir, "thumb", upload.ThumbnailPath("catalog")),
				); tmpErr != nil {
					gcutil.LogError(err).
						Str("catalogThumbnail", upload.ThumbnailPath("catalog")).
						Str("srcBoard", srcBoard.Dir).
						Str("destBoard", destBoard.Dir).
						Msg("Unable to move catalog thumbnail from source board to destination board")
				}
				if err == nil {
					err = tmpErr
				}
			}
			if tmpErr == nil {
				// moved file successfully
				gcutil.LogInfo().
					Int("movedFileForPost", post.ID).
					Str("srcBoard", srcBoard.Dir).
					Str("destBoard", destBoard.Dir).
					Str("filename", upload.Filename).Send()
			}
		}
		if err != nil {
			// got at least one error while trying to move files (if there were any)
			server.ServeError(writer, "Error while moving post upload: "+err.Error(), wantsJSON,
				map[string]interface{}{
					"postID":    postID,
					"srcBoard":  srcBoard.Dir,
					"destBoard": destBoard.Dir,
				})
			return
		}

		// remove the old thread page (new one will be created if no errors)
		if err = os.Remove(path.Join(documentRoot, srcBoard.Dir, "res", postIDstr+".html")); err != nil {
			gcutil.LogError(err).
				Int("postID", postID).
				Str("srcBoard", srcBoard.Dir).
				Msg("Failed deleting thread page")
			writer.WriteHeader(500)
			server.ServeError(writer, "Failed deleting thread page: "+err.Error(), wantsJSON, map[string]interface{}{
				"postID":   postID,
				"srcBoard": srcBoard.Dir,
			})
			return
		}
		// same for the old JSON file
		if err = os.Remove(path.Join(documentRoot, srcBoard.Dir, "res", postIDstr+".json")); err != nil {
			gcutil.LogError(err).
				Int("postID", postID).
				Str("srcBoard", srcBoard.Dir).
				Msg("Failed deleting thread JSON file")
			writer.WriteHeader(500)
			server.ServeError(writer, "Failed deleting thread JSON file: "+err.Error(), wantsJSON, map[string]interface{}{
				"postID":   postID,
				"srcBoard": srcBoard.Dir,
			})
			return
		}

		// oldThreadID := post.ThreadID

		// oldParentID := post.ParentID // hacky, this will likely be fixed when gcsql's handling of ParentID struct properties is changed
		// post.ParentID = 0
		if err = building.BuildThreadPages(post); err != nil {
			gcutil.LogError(err).Int("postID", postID).Msg("Failed moved thread page")
			writer.WriteHeader(500)
			server.ServeError(writer, "Failed building thread page: "+err.Error(), wantsJSON, map[string]interface{}{
				"postID": postID,
			})
			return
		}
		// post.ParentID = oldParentID
		if err = building.BuildBoardPages(srcBoard); err != nil {
			gcutil.LogError(err).Int("srcBoardID", srcBoardID).Send()
			writer.WriteHeader(500)
			server.ServeError(writer, "Failed building board page: "+err.Error(), wantsJSON, map[string]interface{}{
				"srcBoardID": srcBoardID,
			})
			return
		}
		if err = building.BuildBoardPages(destBoard); err != nil {
			gcutil.LogError(err).Int("destBoardID", destBoardID).Send()
			writer.WriteHeader(500)
			server.ServeError(writer, "Failed building destination board page: "+err.Error(), wantsJSON, map[string]interface{}{
				"destBoardID": destBoardID,
			})
			return
		}
		if wantsJSON {
			server.ServeJSON(writer, map[string]interface{}{
				"status":    "success",
				"postID":    postID,
				"srcBoard":  srcBoard.Dir,
				"destBoard": destBoard.Dir,
			})
		} else {
			http.Redirect(writer, request, config.WebPath(destBoard.Dir, "res", postIDstr+".html"), http.StatusMovedPermanently)
		}
	}
}

// move file if it exists on the filesystem and don't throw any errors if it doesn't, returning any other errors
func moveFileIfExists(src string, dest string) error {
	err := os.Rename(src, dest)
	if errors.Is(err, os.ErrNotExist) {
		// file doesn't exist
		return nil
	}
	return err
}
