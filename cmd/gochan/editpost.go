package main

import (
	"bytes"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

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
	password := request.PostFormValue("password")
	wantsJSON := serverutil.IsRequestingJSON(request)
	infoEv, errEv := gcutil.LogRequest(request)
	defer gcutil.LogDiscard(infoEv, errEv)

	if editBtn == "Edit post" {
		var err error
		if len(checkedPosts) == 0 {
			server.ServeError(writer, server.NewServerError("You need to select one post to edit", http.StatusBadRequest), wantsJSON, nil)
			return
		} else if len(checkedPosts) > 1 {
			server.ServeError(writer, server.NewServerError("You can only edit one post at a time", http.StatusBadRequest), wantsJSON, nil)
			return
		}

		rank := manage.GetStaffRank(request)
		if password == "" && rank == 0 {
			server.ServeError(writer, server.NewServerError("Password required for post editing", http.StatusUnauthorized), wantsJSON, nil)
			return
		}
		passwordMD5 := gcutil.Md5Sum(password)

		post, err := gcsql.GetPostFromID(checkedPosts[0], true)
		if err != nil {
			errEv.Err(err).Caller().
				Msg("Error getting post information")
			server.ServeError(writer, server.NewServerError("Error getting post information", http.StatusInternalServerError), wantsJSON, nil)
			return
		}
		errEv.Int("postID", post.ID)

		if post.Password != passwordMD5 && rank == 0 {
			server.ServeError(writer, server.NewServerError("Wrong password", http.StatusUnauthorized), wantsJSON, nil)
			return
		}

		board, err := post.GetBoard()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get board ID from post")
			server.ServeError(writer, server.NewServerError("Unable to get board ID from post", http.StatusInternalServerError), wantsJSON, nil)
			return
		}
		if strings.Contains(string(post.Message), `<span class="dice-roll">`) && !config.GetBoardConfig(board.Dir).AllowDiceRerolls {
			server.ServeError(writer, server.NewServerError("Dice rerolls are not allowed on this board", http.StatusBadRequest), wantsJSON, nil)
			return
		}

		errEv.Str("board", board.Dir)
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Send()
			server.ServeError(writer, server.NewServerError("Error getting post upload info: "+err.Error(), http.StatusInternalServerError), wantsJSON, nil)
			return
		}

		data := map[string]any{
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
		var buf bytes.Buffer
		if err = serverutil.MinifyTemplate(gctemplates.PostEdit, data, &buf, "text/html"); err != nil {
			errEv.Err(err).Caller().
				Msg("Error executing edit post template")
			server.ServeError(writer, server.NewServerError("Error executing edit post template: "+err.Error(), http.StatusInternalServerError), wantsJSON, nil)
			return
		}
		writer.Write(buf.Bytes())
	}
	if doEdit != "post" && doEdit != "upload" {
		return
	}
	postIDstr := request.PostFormValue("postid")
	postid, err := strconv.Atoi(postIDstr)
	if err != nil {
		errEv.Err(err).Caller().
			Str("postid", postIDstr).
			Msg("Invalid form data")
		server.ServeError(writer, server.NewServerError("Invalid form data: "+err.Error(), http.StatusBadRequest), wantsJSON, map[string]any{
			"postid": postid,
		})
		return
	}
	gcutil.LogInt("postID", postid, infoEv, errEv)
	post, err := gcsql.GetPostFromID(postid, true)
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to find post")
		server.ServeError(writer, server.NewServerError("Unable to find post", http.StatusBadRequest), wantsJSON, map[string]any{
			"postid": postid,
		})
		return
	}
	boardIDstr := request.PostFormValue("boardid")
	boardid, err := strconv.Atoi(boardIDstr)
	if err != nil {
		errEv.Err(err).Caller().
			Str("boardID", boardIDstr).
			Msg("Invalid form data")
		server.ServeError(writer, server.NewServerError("Invalid form data: "+err.Error(), http.StatusBadRequest), wantsJSON, nil)
		return
	}
	gcutil.LogInt("boardID", boardid, infoEv, errEv)

	rank := manage.GetStaffRank(request)
	passwordMD5 := gcutil.Md5Sum(password)
	if post.Password != passwordMD5 && rank == 0 {
		server.ServeError(writer, server.NewServerError("Wrong password", http.StatusUnauthorized), wantsJSON, nil)
		return
	}

	board, err := gcsql.GetBoardFromID(boardid)
	if err != nil {
		server.ServeError(writer, server.NewServerError("Invalid form data: "+err.Error(), http.StatusBadRequest), wantsJSON, map[string]any{
			"boardid": boardid,
		})
		errEv.Err(err).Caller().Msg("Invalid form data")
		return
	}

	if doEdit == "upload" {
		oldUpload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get post upload")
			server.ServeError(writer, server.NewServerError("Error getting post upload info: "+err.Error(), http.StatusInternalServerError), wantsJSON, nil)
			return
		}

		upload, err := uploads.AttachUploadFromRequest(request, writer, post, board, gcutil.LogInfo(), errEv)
		if err != nil {
			server.ServeError(writer, server.NewServerError("Unable to attach upload:"+err.Error(), http.StatusInternalServerError), wantsJSON, nil)
			return
		}
		if upload == nil {
			server.ServeError(writer, server.NewServerError("Missing upload replacement", http.StatusBadRequest), wantsJSON, nil)
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
				server.ServeError(writer, server.NewServerError("Error unlinking old upload from post: "+err.Error(), http.StatusInternalServerError), wantsJSON, nil)
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
			server.ServeError(writer, server.NewServerError("Error attaching new upload: "+err.Error(), http.StatusInternalServerError), wantsJSON, map[string]any{
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
			server.ServeError(writer, server.NewServerError("Recovered from a panic in an event handler (message-pre-format)",
				http.StatusInternalServerError), wantsJSON, map[string]any{"postid": post.ID})
			return
		}
		if err != nil {
			errEv.Err(err).Caller().
				Str("triggeredEvent", "message-pre-format").
				Send()
			server.ServeError(writer, server.NewServerError(err.Error(), http.StatusInternalServerError), wantsJSON, map[string]any{
				"postid": post.ID,
			})
			return
		}

		// trigger the pre-format event
		_, err, recovered := events.TriggerEvent("message-pre-format", post, request)
		if recovered {
			server.ServeError(writer, server.NewServerError("Recovered from a panic in an event handler (message-pre-format)", http.StatusInternalServerError), wantsJSON, nil)
			return
		}
		if err != nil {
			errEv.Err(err).Caller().
				Str("event", "message-pre-format").
				Send()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}

		formattedStr, err := posting.ApplyWordFilters(request.PostFormValue("editmsg"), board.Dir)
		if err != nil {
			errEv.Err(err).Caller().Msg("Error formatting post")
			server.ServeError(writer, "Unable to format post", wantsJSON, map[string]any{
				"boardDir": board.Dir,
			})
			return
		}

		formatted, err := posting.FormatMessage(formattedStr, board.Dir)
		if err != nil {
			errEv.Err(err).Caller().Send()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
		oldEmail := post.Email
		oldSubject := post.Subject
		oldMessage := post.Message
		oldMessageRaw := post.MessageRaw

		if err = post.UpdateContents(
			request.PostFormValue("editemail"),
			request.PostFormValue("editsubject"),
			formatted,
			request.PostFormValue("editmsg"),
		); err != nil {
			errEv.Err(err).Caller().
				Msg("Unable to edit post")
			server.ServeError(writer, "Unable to edit post: "+err.Error(), wantsJSON, map[string]any{
				"postid": post.ID,
			})
			return
		}

		// get post upload (if any) and do a filter check
		upload, err := post.GetUpload()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get post upload")
			server.ServeError(writer, "unable to get post upload", wantsJSON, nil)
			return
		}

		filter, err := gcsql.DoPostFiltering(post, upload, boardid, request, errEv)
		if err != nil {
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
		if posting.HandleFilterAction(filter, post, upload, board, writer, request) {
			post.UpdateContents(oldEmail, oldSubject, oldMessage, oldMessageRaw)
			return
		}
	}

	if err = building.BuildBoards(false, boardid); err != nil {
		server.ServeError(writer, "Error rebuilding boards: "+err.Error(), wantsJSON, nil)
	} else if err = building.BuildFrontPage(); err != nil {
		server.ServeError(writer, "Error rebuilding front page: "+err.Error(), wantsJSON, nil)
	} else {
		http.Redirect(writer, request, post.WebPath(), http.StatusFound)
		infoEv.Msg("Post edited")
	}
}
