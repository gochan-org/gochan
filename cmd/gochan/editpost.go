package main

import (
	"bytes"
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
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/serverutil"
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
			serverutil.ServeErrorPage(writer, "You need to select one post to edit.")
			return
		} else if len(checkedPosts) > 1 {
			serverutil.ServeErrorPage(writer, "You can only edit one post at a time.")
			return
		}

		rank := manage.GetStaffRank(request)
		if password == "" && rank == 0 {
			serverutil.ServeErrorPage(writer, "Password required for post editing")
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
			serverutil.ServeErrorPage(writer, "Wrong password")
			return
		}

		board, err := post.GetBoard()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get board ID from post")
			serverutil.ServeErrorPage(writer, "Unable to get board ID from post: "+err.Error())
			return
		}
		errEv.Str("board", board.Dir)
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Send()
			serverutil.ServeErrorPage(writer, "Error getting post upload info: "+err.Error())
			return
		}

		data := map[string]interface{}{
			"boards":         gcsql.AllBoards,
			"systemCritical": config.GetSystemCriticalConfig(),
			"siteConfig":     config.GetSiteConfig(),
			"board":          board,
			"boardConfig":    config.GetBoardConfig(""),
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
			serverutil.ServeError(writer, "Error executing edit post template: "+err.Error(), wantsJSON, nil)
			return
		}
		writer.Write(buf.Bytes())
	}
	if doEdit == "post" || doEdit == "upload" {
		var password string
		postid, err := strconv.Atoi(request.FormValue("postid"))
		if err != nil {
			errEv.Err(err).Caller().
				Str("postid", request.FormValue("postid")).
				Msg("Invalid form data")
			serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		post, err := gcsql.GetPostFromID(postid, true)
		if err != nil {
			errEv.Err(err).
				Int("postid", postid).
				Msg("Unable to find post")
			serverutil.ServeError(writer, "Unable to find post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		boardid, err := strconv.Atoi(request.FormValue("boardid"))
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Invalid form data")
			serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, nil)
			return
		}

		rank := manage.GetStaffRank(request)
		if request.FormValue("password") != password && rank == 0 {
			serverutil.ServeError(writer, "Wrong password", wantsJSON, nil)
			return
		}

		board, err := gcsql.GetBoardFromID(boardid)
		if err != nil {
			serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
				"boardid": boardid,
			})
			errEv.Err(err).Caller().Msg("Invalid form data")
			return
		}

		if doEdit == "upload" {
			oldUpload, err := post.GetUpload()
			if err != nil {
				errEv.Err(err).Caller().Send()
				serverutil.ServeError(writer, err.Error(), wantsJSON, nil)
				return
			}

			upload, gotErr := posting.AttachUploadFromRequest(request, writer, post, board)
			if gotErr {
				// AttachUploadFromRequest handles error serving/logging
				return
			}
			if upload == nil {
				serverutil.ServeError(writer, "Missing upload replacement", wantsJSON, nil)
				return
			}
			documentRoot := config.GetSystemCriticalConfig().DocumentRoot
			var filePath, thumbPath, catalogThumbPath string
			if oldUpload != nil && oldUpload.Filename != "deleted" {
				filePath = path.Join(documentRoot, board.Dir, "src", oldUpload.Filename)
				thumbPath = path.Join(documentRoot, board.Dir, "thumb", oldUpload.ThumbnailPath("thumb"))
				catalogThumbPath = path.Join(documentRoot, board.Dir, "thumb", oldUpload.ThumbnailPath("catalog"))
				if err = post.UnlinkUploads(false); err != nil {
					errEv.Err(err).Caller().Send()
					serverutil.ServeError(writer, "Error unlinking old upload from post: "+err.Error(), wantsJSON, nil)
					return
				}
				os.Remove(filePath)
				os.Remove(thumbPath)
				if post.IsTopPost {
					os.Remove(catalogThumbPath)
				}
			}
			if err = post.AttachFile(upload); err != nil {
				errEv.Err(err).Caller().
					Str("newFilename", upload.Filename).
					Str("newOriginalFilename", upload.OriginalFilename).
					Send()
				serverutil.ServeError(writer, "Error attaching new upload: "+err.Error(), wantsJSON, map[string]interface{}{
					"filename": upload.OriginalFilename,
				})
				filePath = path.Join(documentRoot, board.Dir, "src", upload.Filename)
				thumbPath = path.Join(documentRoot, board.Dir, "thumb", upload.ThumbnailPath("thumb"))
				catalogThumbPath = path.Join(documentRoot, board.Dir, "thumb", upload.ThumbnailPath("catalog"))
				os.Remove(filePath)
				os.Remove(thumbPath)
				if post.IsTopPost {
					os.Remove(catalogThumbPath)
				}
			}
		} else {
			if err = post.UpdateContents(
				request.FormValue("editemail"),
				request.FormValue("editsubject"),
				posting.FormatMessage(request.FormValue("editmsg"), board.Dir),
				request.FormValue("editmsg"),
			); err != nil {
				errEv.Err(err).Caller().
					Int("postid", post.ID).
					Msg("Unable to edit post")
				serverutil.ServeError(writer, "Unable to edit post: "+err.Error(), wantsJSON, map[string]interface{}{
					"postid": post.ID,
				})
				return
			}
		}

		if err = building.BuildBoards(false, boardid); err != nil {
			serverutil.ServeErrorPage(writer, "Error rebuilding boards: "+err.Error())
		}
		if err = building.BuildFrontPage(); err != nil {
			serverutil.ServeErrorPage(writer, "Error rebuilding front page: "+err.Error())
		}
		http.Redirect(writer, request, post.WebPath(), http.StatusFound)
		return
	}
}
