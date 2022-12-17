package main

import (
	"net/http"
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
		if post.Password != passwordMD5 && rank == 0 {
			serverutil.ServeErrorPage(writer, "Wrong password")
			return
		}

		boardID, err := post.GetBoardID()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get board ID from post")
			serverutil.ServeErrorPage(writer, "Unable to get board ID from post: "+err.Error())
			return
		}

		if err = serverutil.MinifyTemplate(gctemplates.PostEdit, map[string]interface{}{
			"boards":         gcsql.AllBoards,
			"systemCritical": config.GetSystemCriticalConfig(),
			"siteConfig":     config.GetSiteConfig(),
			"boardID":        boardID,
			"boardConfig":    config.GetBoardConfig(""),
			"post":           post,
			"referrer":       request.Referer(),
		}, writer, "text/html"); err != nil {
			errEv.Err(err).Caller().
				Msg("Error executing edit post template")
			serverutil.ServeError(writer, "Error executing edit post template: "+err.Error(), wantsJSON, nil)
			return
		}
	}
	if doEdit == "1" {
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
