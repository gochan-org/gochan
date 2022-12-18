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
	errEv := gcutil.LogError(nil).
		Str("IP", gcutil.GetRealIP(request))

	defer errEv.Discard()
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
		errEv.Err(err).Caller().
			Str("boardid", request.FormValue("boardid")).
			Msg("Invalid form data (boardid)")
		writer.WriteHeader(http.StatusBadRequest)
		serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	errEv.Int("boardid", boardid)
	board, err := gcsql.GetBoardFromID(boardid)
	if err != nil {
		serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardid": boardid,
		})
		errEv.Err(err).Caller().
			Msg("Invalid form data (error populating data")
		return
	}

	if password == "" && rank == 0 {
		serverutil.ServeError(writer, "Password required for post deletion", wantsJSON, nil)
		return
	}

	for _, checkedPostID := range checkedPosts {
		post, err := gcsql.GetPostFromID(checkedPostID, true)
		if err == sql.ErrNoRows {
			serverutil.ServeError(writer, "Post does not exist", wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": board.ID,
			})
			return
		} else if err != nil {
			errEv.Err(err).Caller().
				Int("postid", checkedPostID).
				Msg("Error deleting post")
			serverutil.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid":  checkedPostID,
				"boardid": board.ID,
			})
			return
		}

		if passwordMD5 != post.Password && rank == 0 {
			serverutil.ServeError(writer, fmt.Sprintf("Incorrect password for #%d", post.ID), wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": board.ID,
			})
			return
		}

		if fileOnly {
			upload, err := post.GetUpload()
			if err != nil {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Msg("Unable to get file upload info")
				serverutil.ServeError(writer, "Error getting file uplaod info: "+err.Error(),
					wantsJSON, map[string]interface{}{"postid": post.ID})
				return
			}
			documentRoot := config.GetSystemCriticalConfig().DocumentRoot
			if upload != nil && upload.Filename != "deleted" {
				filePath := path.Join(documentRoot, board.Dir, "src", upload.Filename)
				if err = os.Remove(filePath); err != nil {
					errEv.Err(err).Caller().
						Int("postid", post.ID).
						Str("filename", upload.Filename).
						Msg("Unable to delete file")
					serverutil.ServeError(writer, "Unable to delete file: "+err.Error(),
						wantsJSON, map[string]interface{}{"postid": post.ID})
					return
				}
				// delete the file's thumbnail
				thumbPath := path.Join(documentRoot, board.Dir, "thumb", upload.ThumbnailPath("thumb"))
				if err = os.Remove(thumbPath); err != nil {
					errEv.Err(err).Caller().
						Int("postid", post.ID).
						Str("thumbnail", upload.ThumbnailPath("thumb")).
						Msg("Unable to delete thumbnail")
					serverutil.ServeError(writer, "Unable to delete thumbnail: "+err.Error(),
						wantsJSON, map[string]interface{}{"postid": post.ID})
					return
				}
				// delete the catalog thumbnail
				if post.IsTopPost {
					thumbPath := path.Join(documentRoot, board.Dir, "thumb", upload.ThumbnailPath("catalog"))
					if err = os.Remove(thumbPath); err != nil {
						errEv.Err(err).Caller().
							Int("postid", post.ID).
							Str("catalogThumb", upload.ThumbnailPath("catalog")).
							Msg("Unable to delete catalog thumbnail")
						serverutil.ServeError(writer, "Unable to delete catalog thumbnail: "+err.Error(),
							wantsJSON, map[string]interface{}{"postid": post.ID})
						return
					}
				}
				if err = post.UnlinkUploads(true); err != nil {
					errEv.Err(err).Caller().
						Str("requestType", "deleteFile").
						Int("postid", post.ID).
						Msg("Error unlinking post uploads")
					serverutil.ServeError(writer, "Unable to unlink post uploads"+err.Error(),
						wantsJSON, map[string]interface{}{"postid": post.ID})
					return
				}
			}
			if err = building.BuildBoardPages(board); err != nil {
				errEv.Err(err).Caller().Send()
				serverutil.ServeError(writer, "Unable to build board pages for /"+board.Dir+"/: "+err.Error(), wantsJSON, map[string]interface{}{
					"boardDir": board.Dir,
				})
				return
			}

			var opPost *gcsql.Post
			if post.IsTopPost {
				opPost = post
			} else {
				if opPost, err = post.GetTopPost(); err != nil {
					errEv.Err(err).Caller().
						Int("postid", post.ID).
						Msg("Unable to get thread information from post")
					serverutil.ServeError(writer, "Unable to get thread info from post: "+err.Error(), wantsJSON, map[string]interface{}{
						"postid": post.ID,
					})
					return
				}
			}
			if building.BuildThreadPages(opPost); err != nil {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Msg("Unable to build thread pages")
				serverutil.ServeError(writer, "Unable to get board info from post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
		} else {
			// delete the post
			if err = post.Delete(); err != nil {
				errEv.Err(err).Caller().
					Str("requestType", "deletePost").
					Int("postid", post.ID).
					Msg("Error deleting post")
				serverutil.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
			if post.IsTopPost {
				threadIndexPath := path.Join(config.GetSystemCriticalConfig().DocumentRoot, board.WebPath(strconv.Itoa(post.ID), "threadPage"))
				os.Remove(threadIndexPath + ".html")
				os.Remove(threadIndexPath + ".json")
			} else {
				building.BuildBoardPages(board)
				// _board, _ := gcsql.GetBoardFromID(post.BoardID)
				// building.BuildBoardPages(&_board)
			}
			building.BuildBoards(false, boardid)
		}
		gcutil.LogAccess(request).
			Str("requestType", "deletePost").
			Int("boardid", boardid).
			Int("postid", post.ID).
			Bool("fileOnly", fileOnly).
			Msg("Post deleted")
		if wantsJSON {
			serverutil.ServeJSON(writer, map[string]interface{}{
				"success":  "post deleted",
				"postid":   post.ID,
				"boardid":  boardid,
				"fileOnly": fileOnly,
			})
		} else {
			if post.IsTopPost {
				// deleted thread
				http.Redirect(writer, request, board.WebPath("/", "boardPage"), http.StatusFound)
			} else {
				// deleted a post in the thread
				http.Redirect(writer, request, post.WebPath(), http.StatusFound)
			}
		}
	}
}
