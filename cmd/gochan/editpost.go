package main

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/manage"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

func editPost(checkedPosts []int, editBtn string, doEdit string, writer http.ResponseWriter, request *http.Request) {
	password := request.FormValue("password")
	wantsJSON := serverutil.IsRequestingJSON(request)
	errEv := gcutil.LogError(nil).
		Str("IP", gcutil.GetRealIP(request))
	defer errEv.Discard()

	if editBtn == "Edit post" {
		var err error
		if len(checkedPosts) == 0 {
			server.ServeErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(checkedPosts) > 1 {
			server.ServeErrorPage(writer, "You can only edit one post at a time.")
			return
		}

		rank := manage.GetStaffRank(request)
		if password == "" && rank == 0 {
			server.ServeErrorPage(writer, "Password required for post editing")
			return
		}
		passwordMD5 := gcutil.Md5Sum(password)

		post, err := gcsql.GetPostFromID(checkedPosts[0], true)
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Error getting post information")
			return
		}
		errEv.Int("postID", post.ID)

		if post.Password != passwordMD5 && rank == 0 {
			server.ServeErrorPage(writer, "Wrong password")
			return
		}

		board, err := post.GetBoard()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get board ID from post")
			server.ServeErrorPage(writer, "Unable to get board ID from post: "+err.Error())
			return
		}
		errEv.Str("board", board.Dir)
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Send()
			server.ServeErrorPage(writer, "Error getting post upload info: "+err.Error())
			return
		}

		data := map[string]interface{}{
			"boards":         gcsql.AllBoards,
			"systemCritical": config.GetSystemCriticalConfig(),
			"siteConfig":     config.GetSiteConfig(),
			"board":          board,
			"boardConfig":    config.GetBoardConfig(""),
			"password":       password,
			"post":           post,
			"referrer":       request.Referer(),
		}
		if upload != nil {
			data["upload"] = upload
		}
		buf := bytes.NewBufferString("")
		err = serverutil.MinifyTemplate(gctemplates.PostEdit, data, buf, "text/html")
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Error executing edit post template")
			server.ServeError(writer, "Error executing edit post template: "+err.Error(), wantsJSON, nil)
			return
		}
		writer.Write(buf.Bytes())
	}
	if doEdit == "post" || doEdit == "upload" {
		postid, err := strconv.Atoi(request.FormValue("postid"))
		if err != nil {
			errEv.Err(err).Caller().
				Str("postid", request.FormValue("postid")).
				Msg("Invalid form data")
			server.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		post, err := gcsql.GetPostFromID(postid, true)
		if err != nil {
			errEv.Err(err).
				Int("postid", postid).
				Msg("Unable to find post")
			server.ServeError(writer, "Unable to find post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		boardid, err := strconv.Atoi(request.FormValue("boardid"))
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Invalid form data")
			server.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, nil)
			return
		}

		rank := manage.GetStaffRank(request)
		password := request.PostFormValue("password")
		passwordMD5 := gcutil.Md5Sum(password)
		if post.Password != passwordMD5 && rank == 0 {
			server.ServeError(writer, "Wrong password", wantsJSON, nil)
			return
		}

		board, err := gcsql.GetBoardFromID(boardid)
		if err != nil {
			server.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
				"boardid": boardid,
			})
			errEv.Err(err).Caller().Msg("Invalid form data")
			return
		}

		if doEdit == "upload" {
			oldUpload, err := post.GetUpload()
			if err != nil {
				errEv.Err(err).Caller().Send()
				server.ServeError(writer, err.Error(), wantsJSON, nil)
				return
			}

			upload, err := uploads.AttachUploadFromRequest(request, writer, post, board)
			if err != nil {
				server.ServeError(writer, err.Error(), wantsJSON, nil)
				return
			}
			if upload == nil {
				server.ServeError(writer, "Missing upload replacement", wantsJSON, nil)
				return
			}
			documentRoot := config.GetSystemCriticalConfig().DocumentRoot
			var filePath, thumbPath, catalogThumbPath string
			if oldUpload != nil {
				filePath = path.Join(documentRoot, board.Dir, "src", oldUpload.Filename)
				thumbPath, catalogThumbPath = uploads.GetThumbnailFilenames(
					path.Join(documentRoot, board.Dir, "thumb", oldUpload.Filename))
				if err = post.UnlinkUploads(false); err != nil {
					errEv.Err(err).Caller().Send()
					server.ServeError(writer, "Error unlinking old upload from post: "+err.Error(), wantsJSON, nil)
					return
				}
				if oldUpload.Filename != "deleted" {
					os.Remove(filePath)
					os.Remove(thumbPath)
					if post.IsTopPost {
						os.Remove(catalogThumbPath)
					}
				}
			}

			if err = post.AttachFile(upload); err != nil {
				errEv.Err(err).Caller().
					Str("newFilename", upload.Filename).
					Str("newOriginalFilename", upload.OriginalFilename).
					Send()
				server.ServeError(writer, "Error attaching new upload: "+err.Error(), wantsJSON, map[string]interface{}{
					"filename": upload.OriginalFilename,
				})
				filePath = path.Join(documentRoot, board.Dir, "src", upload.Filename)
				thumbPath, catalogThumbPath = uploads.GetThumbnailFilenames(
					path.Join(documentRoot, board.Dir, "thumb", upload.Filename))
				os.Remove(filePath)
				os.Remove(thumbPath)
				if post.IsTopPost {
					os.Remove(catalogThumbPath)
				}
			}
		} else {
			var recovered bool
			_, err, recovered = events.TriggerEvent("message-pre-format", post, request)
			if recovered {
				writer.WriteHeader(http.StatusInternalServerError)
				server.ServeError(writer, "Recovered from a panic in an event handler (message-pre-format)", wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
			if err != nil {
				errEv.Err(err).Caller().
					Str("triggeredEvent", "message-pre-format").
					Send()
				server.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
			if err = post.UpdateContents(
				request.FormValue("editemail"),
				request.FormValue("editsubject"),
				posting.FormatMessage(request.FormValue("editmsg"), board.Dir),
				request.FormValue("editmsg"),
			); err != nil {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Msg("Unable to edit post")
				server.ServeError(writer, "Unable to edit post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
		}

		if err = building.BuildBoards(false, boardid); err != nil {
			server.ServeErrorPage(writer, "Error rebuilding boards: "+err.Error())
		}
		if err = building.BuildFrontPage(); err != nil {
			server.ServeErrorPage(writer, "Error rebuilding front page: "+err.Error())
		}
		http.Redirect(writer, request, post.WebPath(), http.StatusFound)
		return
	}
}
