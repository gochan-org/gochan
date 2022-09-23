package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

func deletePosts(checkedPosts []int, writer http.ResponseWriter, request *http.Request) {
	// Delete a post or thread
	password := request.FormValue("password")
	passwordMD5 := gcutil.Md5Sum(password)
	rank := manage.GetStaffRank(request)
	fileOnly := request.FormValue("fileonly") == "on"
	wantsJSON := serverutil.IsRequestingJSON(request)
	if wantsJSON {
		writer.Header().Set("Content-Type", "application/json")
	} else {
		writer.Header().Set("Content-Type", "text/plain")
	}
	boardid, err := strconv.Atoi(request.FormValue("boardid"))
	if err != nil {
		gcutil.LogError(err).
			Str("IP", gcutil.GetRealIP(request)).
			Str("boardid", request.FormValue("boardid")).
			Msg("Invalid form data (boardid)")
		writer.WriteHeader(http.StatusBadRequest)
		serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}

	board, err := gcsql.GetBoardFromID(boardid)
	if err != nil {
		serverutil.ServeErrorPage(writer, "Invalid form data: "+err.Error())
		gcutil.LogError(err).
			Str("IP", gcutil.GetRealIP(request)).
			Int("boardid", boardid).
			Msg("Invalid form data (error populating data")
		return
	}

	if password == "" && rank == 0 {
		serverutil.ServeErrorPage(writer, "Password required for post deletion")
		return
	}

	for _, checkedPostID := range checkedPosts {
		var post gcsql.Post
		var err error
		post.ID = checkedPostID
		post.BoardID = boardid
		post, err = gcsql.GetSpecificPost(post.ID, true)
		if err == sql.ErrNoRows {
			serverutil.ServeError(writer, "Post does not exist", wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": post.BoardID,
			})
			return
		} else if err != nil {
			gcutil.Logger().Error().
				Str("requestType", "deletePost").
				Err(err).
				Int("postid", post.ID).
				Int("boardid", post.BoardID).
				Msg("Error deleting post")
			serverutil.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": post.BoardID,
			})
			return
		}

		if passwordMD5 != post.Password && rank == 0 {
			serverutil.ServeError(writer, fmt.Sprintf("Incorrect password for #%d", post.ID), wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": post.BoardID,
			})
			return
		}

		if fileOnly {
			fileName := post.Filename
			if fileName != "" && fileName != "deleted" {
				var files []string
				if files, err = post.GetFilePaths(); err != nil {
					gcutil.Logger().Error().
						Str("requestType", "deleteFile").
						Int("postid", post.ID).
						Err(err).
						Msg("Error getting file upload info")
					serverutil.ServeError(writer, "Error getting file upload info: "+err.Error(), wantsJSON, map[string]interface{}{
						"postid": post.ID,
					})
					return
				}

				if err = post.UnlinkUploads(true); err != nil {
					gcutil.Logger().Error().
						Str("requestType", "deleteFile").
						Int("postid", post.ID).
						Err(err).
						Msg("Error unlinking post uploads")
					serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
						"postid": post.ID,
					})
					return
				}

				for _, filePath := range files {
					if err = os.Remove(filePath); err != nil {
						fileBase := path.Base(filePath)
						gcutil.Logger().Error().
							Str("requestType", "deleteFile").
							Int("postid", post.ID).
							Str("file", filePath).
							Err(err).
							Msg("Error unlinking post uploads")
						serverutil.ServeError(writer, fmt.Sprintf("Error deleting %s: %s", fileBase, err.Error()), wantsJSON, map[string]interface{}{
							"postid": post.ID,
							"file":   fileBase,
						})
						return
					}
				}
			}
			_board, _ := gcsql.GetBoardFromID(post.BoardID)
			building.BuildBoardPages(&_board)

			var opPost gcsql.Post
			if post.ParentID > 0 {
				// post is a reply, get the OP
				opPost, _ = gcsql.GetSpecificPost(post.ParentID, true)
			} else {
				opPost = post
			}
			building.BuildThreadPages(&opPost)
		} else {
			// delete the post
			if err = gcsql.DeletePost(post.ID, true); err != nil {
				gcutil.Logger().Error().
					Str("requestType", "deleteFile").
					Int("postid", post.ID).
					Err(err).
					Msg("Error deleting post")
				serverutil.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
			if post.ParentID == 0 {
				threadIndexPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, board.WebPath(strconv.Itoa(post.ID), "threadPage"))
				os.Remove(threadIndexPath + ".html")
				os.Remove(threadIndexPath + ".json")
			} else {
				_board, _ := gcsql.GetBoardFromID(post.BoardID)
				building.BuildBoardPages(&_board)
			}
			building.BuildBoards(false, post.BoardID)
		}
		gcutil.LogAccess(request).
			Str("requestType", "deletePost").
			Int("boardid", post.BoardID).
			Int("postid", post.ID).
			Bool("fileOnly", fileOnly).
			Msg("Post deleted")
		if wantsJSON {
			serverutil.ServeJSON(writer, map[string]interface{}{
				"success":  "post deleted",
				"postid":   post.ID,
				"boardid":  post.BoardID,
				"fileOnly": fileOnly,
			})
		} else {
			if post.ParentID == 0 {
				// deleted thread
				http.Redirect(writer, request, board.WebPath("/", "boardPage"), http.StatusFound)
			} else {
				// deleted a post in the thread
				http.Redirect(writer, request, post.GetURL(false), http.StatusFound)
			}
		}
	}
}
