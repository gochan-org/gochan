package main

import (
	"bytes"
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
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func moveThread(checkedPosts []int, moveBtn string, doMove string, writer http.ResponseWriter, request *http.Request) {
	password := request.PostFormValue("password")
	var passwordMD5 string
	if password != "" {
		passwordMD5 = gcutil.Md5Sum(password)
	}
	wantsJSON := serverutil.IsRequestingJSON(request)
	infoEv, errEv := gcutil.LogRequest(request)
	defer func() {
		errEv.Discard()
		infoEv.Discard()
	}()
	rank := manage.GetStaffRank(request)

	if password == "" && rank == 0 {
		errEv.Msg("Thread move request rejected, non-staff didn't provide a password")
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, "Password required for post moving", wantsJSON, nil)
		return
	}

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
		gcutil.LogInt("postid", checkedPosts[0], errEv, infoEv)

		if err != nil {
			errEv.Err(err).Caller().Msg("Error getting post from ID")
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
		if !post.IsTopPost {
			server.ServeError(writer, "You appear to be trying to move a post that is not the top post in the thread", wantsJSON, map[string]interface{}{
				"postid": checkedPosts[0],
			})
			return
		}

		srcBoardID, err := strconv.Atoi(request.PostFormValue("boardid"))
		if err != nil {
			errEv.Err(err).Caller().
				Str("srcBoardIDstr", request.PostFormValue("boardid")).Send()
			server.ServeError(writer, fmt.Sprintf("Invalid or missing boarid: %q", request.PostFormValue("boardid")), wantsJSON, map[string]interface{}{
				"boardid": srcBoardID,
			})
			return
		}
		var destBoards []gcsql.Board
		var srcBoard gcsql.Board
		for _, board := range gcsql.AllBoards {
			if board.ID == srcBoardID {
				srcBoard = board
			} else {
				destBoards = append(destBoards, board)
			}
		}
		gcutil.LogStr("srcBoard", srcBoard.Dir, errEv, infoEv)
		buf := bytes.NewBufferString("")
		if err = serverutil.MinifyTemplate(gctemplates.MoveThreadPage, map[string]interface{}{
			"boardConfig": config.GetBoardConfig(srcBoard.Dir),
			"postid":      post.ID,
			"destBoards":  destBoards,
			"password":    password,
			"pageTitle":   fmt.Sprintf("Move thread #%d", post.ID),
			"srcBoard":    srcBoard,
		}, buf, "text/html"); err != nil {
			errEv.Err(err).Caller().Send()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
		writer.Write(buf.Bytes())
	} else if doMove == "1" {
		// user got here from the move thread page
		postIDstr := request.PostFormValue("postid")
		postID, err := strconv.Atoi(postIDstr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("postIDstr", postIDstr).Send()
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing postid value: %q: %s", postIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"postid": postIDstr,
			})
			return
		}
		gcutil.LogInt("postID", postID, errEv, infoEv)

		srcBoardIDstr := request.PostFormValue("srcboardid")
		srcBoardID, err := strconv.Atoi(srcBoardIDstr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("srcBoardIDstr", srcBoardIDstr).Send()
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing srcboardid value: %q: %s", srcBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardIDstr,
			})
			return
		}
		srcBoard, err := gcsql.GetBoardFromID(srcBoardID)
		if err != nil {
			errEv.Err(err).Caller().
				Int("srcBoardID", srcBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"srcboardid": srcBoardID,
			})
			return
		}
		gcutil.LogStr("srcBoard", srcBoard.Dir, errEv, infoEv)

		destBoardIDstr := request.PostFormValue("destboardid")
		destBoardID, err := strconv.Atoi(destBoardIDstr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("destBoardIDstr", destBoardIDstr).Send()
			writer.WriteHeader(http.StatusBadRequest)
			server.ServeError(writer, fmt.Sprintf("Error parsing destboardid value: %q: %s", destBoardIDstr, err.Error()), wantsJSON, map[string]interface{}{
				"destboardid": destBoardIDstr,
			})
			return
		}

		destBoard, err := gcsql.GetBoardFromID(destBoardID)
		if err != nil {
			errEv.Err(err).Caller().
				Int("destBoardID", destBoardID).Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"destboardid": destBoardID,
			})
			return
		}
		gcutil.LogStr("destBoard", destBoard.Dir, errEv, infoEv)

		post, err := gcsql.GetPostFromID(postID, true)
		if err != nil {
			errEv.Err(err).Caller().Send()
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"postid": postID,
			})
			return
		}

		if passwordMD5 != post.Password && rank == 0 {
			errEv.Msg("Wrong password")
			server.ServeError(writer, "Wrong password", wantsJSON, nil)
			return
		}

		if err = post.ChangeBoardID(destBoardID); err != nil {
			errEv.Err(err).Caller().Msg("Failed changing thread board ID")
			server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
				"postID":      postID,
				"destBoardID": destBoardID,
			})
			return
		}

		threadUploads, err := gcsql.GetThreadFiles(post)
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get upload info")
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
				errEv.Err(err).Caller().
					Str("filename", upload.Filename).
					Msg("Unable to move file from source board to destination board")
				if err == nil {
					// log all errors but only report the first one to the user
					err = tmpErr
				}
			}

			// move the upload thumbnail
			thumbnail, catalogThumbnail := uploads.GetThumbnailFilenames(upload.Filename)
			if tmpErr = moveFileIfExists(
				path.Join(documentRoot, srcBoard.Dir, "thumb", thumbnail),
				path.Join(documentRoot, destBoard.Dir, "thumb", thumbnail),
			); tmpErr != nil {
				errEv.Err(err).Caller().
					Str("thumbnail", thumbnail).
					Msg("Unable to move thumbnail from source board to destination board")
				if err == nil {
					err = tmpErr
				}
			}
			if upload.PostID == post.ID {
				// move the upload catalog thumbnail
				if tmpErr = moveFileIfExists(
					path.Join(documentRoot, srcBoard.Dir, "thumb", catalogThumbnail),
					path.Join(documentRoot, destBoard.Dir, "thumb", catalogThumbnail),
				); tmpErr != nil {
					errEv.Err(err).Caller().
						Str("catalogThumbnail", catalogThumbnail).
						Msg("Unable to move catalog thumbnail from source board to destination board")
				}
				if err == nil {
					err = tmpErr
				}
			}
			if tmpErr == nil {
				// moved file successfully
				infoEv.Str("filename", upload.Filename)
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
			errEv.Err(err).Caller().
				Msg("Failed deleting thread page")
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Failed deleting thread page: "+err.Error(), wantsJSON, map[string]interface{}{
				"postID":   postID,
				"srcBoard": srcBoard.Dir,
			})
			return
		}
		// same for the old JSON file
		if err = os.Remove(path.Join(documentRoot, srcBoard.Dir, "res", postIDstr+".json")); err != nil {
			errEv.Err(err).Caller().
				Msg("Failed deleting thread JSON file")
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Failed deleting thread JSON file: "+err.Error(), wantsJSON, map[string]interface{}{
				"postID":   postID,
				"srcBoard": srcBoard.Dir,
			})
			return
		}

		if err = building.BuildThreadPages(post); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Failed building thread page: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postID,
			})
			return
		}
		if err = building.BuildBoardPages(srcBoard); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Failed building board page: "+err.Error(), wantsJSON, map[string]interface{}{
				"srcBoardID": srcBoardID,
			})
			return
		}
		if err = building.BuildBoardPages(destBoard); err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
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
		infoEv.Send()
	}
}

// move file if it exists on the filesystem and don't throw any errors if it doesn't, returning any other errors
func moveFileIfExists(src string, dest string) error {
	err := os.Rename(src, dest)
	if os.IsExist(err) {
		// file doesn't exist
		return nil
	}
	return err
}
