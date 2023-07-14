package main

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strconv"

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

type upload struct {
	postID   int
	filename string
	boardDir string
}

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
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	errEv.Int("boardid", boardid)
	board, err := gcsql.GetBoardFromID(boardid)
	if err != nil {
		server.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardid": boardid,
		})
		errEv.Err(err).Caller().
			Msg("Invalid form data (error populating data")
		return
	}

	if password == "" && rank == 0 {
		server.ServeError(writer, "Password required for post deletion", wantsJSON, nil)
		return
	}

	for _, checkedPostID := range checkedPosts {
		post, err := gcsql.GetPostFromID(checkedPostID, true)
		if err == sql.ErrNoRows {
			server.ServeError(writer, "Post does not exist", wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": board.ID,
			})
			return
		} else if err != nil {
			errEv.Err(err).Caller().
				Int("postid", checkedPostID).
				Msg("Error deleting post")
			server.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid":  checkedPostID,
				"boardid": board.ID,
			})
			return
		}

		if passwordMD5 != post.Password && rank == 0 {
			server.ServeError(writer, fmt.Sprintf("Incorrect password for #%d", post.ID), wantsJSON, map[string]interface{}{
				"postid":  post.ID,
				"boardid": board.ID,
			})
			return
		}

		if fileOnly {
			if deletePostUpload(post, board, writer, request, errEv) {
				return
			}
			if err = building.BuildBoardPages(board); err != nil {
				errEv.Err(err).Caller().Send()
				server.ServeError(writer, "Unable to build board pages for /"+board.Dir+"/: "+err.Error(), wantsJSON, map[string]interface{}{
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
					server.ServeError(writer, "Unable to get thread info from post: "+err.Error(), wantsJSON, map[string]interface{}{
						"postid": post.ID,
					})
					return
				}
			}
			if building.BuildThreadPages(opPost); err != nil {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Msg("Unable to build thread pages")
				server.ServeError(writer, "Unable to get board info from post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
		} else {
			if post.IsTopPost {
				rows, err := gcsql.QuerySQL(
					`SELECT filename FROM DBPREFIXfiles
					LEFT JOIN (
						SELECT id FROM DBPREFIXposts WHERE thread_id = ?
					) p
					ON p.id = post_id
					WHERE post_id = p.id AND filename != 'deleted'`,
					post.ThreadID)
				if err != nil {
					errEv.Err(err).Caller().
						Str("requestType", "deleteThread").
						Int("postid", post.ID).
						Int("threadID", post.ThreadID).
						Msg("Unable to get list of filenames in thread")
					server.ServeError(writer, "Unable to get list of filenames in thread", wantsJSON, map[string]interface{}{
						"postid": post.ID,
					})
					return
				}
				defer rows.Close()
				var postUploads []upload
				for rows.Next() {
					var filename string
					if err = rows.Scan(&filename); err != nil {
						errEv.Err(err).Caller().
							Str("requestType", "deleteThread").
							Int("postid", post.ID).
							Int("threadID", post.ThreadID).
							Msg("Unable to get list of filenames in thread")
						server.ServeError(writer, "Unable to get list of filenames in thread", wantsJSON, map[string]interface{}{
							"postid": post.ID,
						})
						return
					}
					postUploads = append(postUploads, upload{
						filename: filename,
						boardDir: board.Dir,
					})
				}
				// done as a goroutine to avoid delays if the thread has a lot of files
				// the downside is of course that if something goes wrong, deletion errors
				// won't be seen in the browser
				go deleteUploads(postUploads)
			} else if deletePostUpload(post, board, writer, request, errEv) {
				return
			}
			// delete the post
			if err = post.Delete(); err != nil {
				errEv.Err(err).Caller().
					Str("requestType", "deletePost").
					Int("postid", post.ID).
					Msg("Error deleting post")
				server.ServeError(writer, "Error deleting post: "+err.Error(), wantsJSON, map[string]interface{}{
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
			server.ServeJSON(writer, map[string]interface{}{
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

func deleteUploads(postUploads []upload) {
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	var filePath string
	var err error
	for _, upload := range postUploads {
		filePath = path.Join(documentRoot, upload.boardDir, "src", upload.filename)
		if err = os.Remove(filePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			gcutil.LogError(err).Caller().
				Str("filePath", filePath).
				Int("postid", upload.postID).Send()
		}
		thumbPath, catalogThumbPath := uploads.GetThumbnailFilenames(
			path.Join(documentRoot, upload.boardDir, "thumb", upload.filename))
		if err = os.Remove(thumbPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			gcutil.LogError(err).Caller().
				Str("thumbPath", thumbPath).
				Int("postid", upload.postID).Send()
		}
		if err = os.Remove(catalogThumbPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			gcutil.LogError(err).Caller().
				Str("catalogThumbPath", catalogThumbPath).
				Int("postid", upload.postID).Send()
		}
	}
}

func deletePostUpload(post *gcsql.Post, board *gcsql.Board, writer http.ResponseWriter, request *http.Request, errEv *zerolog.Event) bool {
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	upload, err := post.GetUpload()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if err != nil {
		errEv.Err(err).Caller().
			Int("postid", post.ID).
			Msg("Unable to get file upload info")
		server.ServeError(writer, "Error getting file uplaod info: "+err.Error(),
			wantsJSON, map[string]interface{}{"postid": post.ID})
		return true
	}
	if upload != nil && upload.Filename != "deleted" {
		filePath := path.Join(documentRoot, board.Dir, "src", upload.Filename)
		if err = os.Remove(filePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errEv.Err(err).Caller().
				Int("postid", post.ID).
				Str("filename", upload.Filename).
				Msg("Unable to delete file")
			server.ServeError(writer, "Unable to delete file: "+err.Error(),
				wantsJSON, map[string]interface{}{"postid": post.ID})
			return true
		}
		// delete the file's thumbnail
		thumbPath, catalogThumbPath := uploads.GetThumbnailFilenames(path.Join(documentRoot, board.Dir, "thumb", upload.Filename))
		if err = os.Remove(thumbPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			errEv.Err(err).Caller().
				Int("postid", post.ID).
				Str("thumbnail", thumbPath).
				Msg("Unable to delete thumbnail")
			server.ServeError(writer, "Unable to delete thumbnail: "+err.Error(),
				wantsJSON, map[string]interface{}{"postid": post.ID})
			return true
		}
		// delete the catalog thumbnail
		if post.IsTopPost {
			if err = os.Remove(catalogThumbPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Str("catalogThumb", catalogThumbPath).
					Msg("Unable to delete catalog thumbnail")
				server.ServeError(writer, "Unable to delete catalog thumbnail: "+err.Error(),
					wantsJSON, map[string]interface{}{"postid": post.ID})
				return true
			}
		}
		// remove the upload from the database
		if err = post.UnlinkUploads(true); err != nil {
			errEv.Err(err).Caller().
				Str("requestType", "deleteFile").
				Int("postid", post.ID).
				Msg("Error unlinking post uploads")
			server.ServeError(writer, "Unable to unlink post uploads"+err.Error(),
				wantsJSON, map[string]interface{}{"postid": post.ID})
			return true
		}
	}
	return false
}
