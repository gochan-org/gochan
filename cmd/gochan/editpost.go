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
			gcutil.Logger().Error().
				Err(err).
				Msg("Error getting post information")
			return
		}

		if post.Password != passwordMD5 && rank == 0 {
			serverutil.ServeErrorPage(writer, "Wrong password")
			return
		}

		if err = gctemplates.PostEdit.Execute(writer, map[string]interface{}{
			"systemCritical": config.GetSystemCriticalConfig(),
			"siteConfig":     config.GetSiteConfig(),
			"boardConfig":    config.GetBoardConfig(""),
			"post":           post,
			"referrer":       request.Referer(),
		}); err != nil {
			gcutil.Logger().Error().
				Err(err).
				Str("IP", gcutil.GetRealIP(request)).
				Msg("Error executing edit post template")

			serverutil.ServeError(writer, "Error executing edit post template: "+err.Error(), wantsJSON, nil)
			return
		}
	}
	if doEdit == "1" {
		var password string
		postid, err := strconv.Atoi(request.FormValue("postid"))
		if err != nil {
			gcutil.LogError(err).
				Str("postid", request.FormValue("postid")).
				Str("IP", gcutil.GetRealIP(request)).
				Msg("Invalid form data")
			serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		post, err := gcsql.GetPostFromID(postid, true)
		if err != nil {
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Int("postid", postid).
				Msg("Unable to find post")
			serverutil.ServeError(writer, "Unable to find post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": postid,
			})
			return
		}
		boardid, err := strconv.Atoi(request.FormValue("boardid"))
		if err != nil {
			gcutil.Logger().Error().
				Err(err).
				Str("IP", gcutil.GetRealIP(request)).
				Msg("Invalid form data")
			serverutil.ServeError(writer, "Invalid form data: "+err.Error(), wantsJSON, nil)
			return
		}
		// password, err = gcsql.GetPostPassword(postid)
		// if err != nil {
		// 	gcutil.LogError(err).
		// 		Str("IP", gcutil.GetRealIP(request)).
		// 		Msg("Invalid form data")
		// 	return
		// }

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
			gcutil.LogError(err).
				Str("IP", gcutil.GetRealIP(request)).
				Msg("Invalid form data")
			return
		}

		if err = post.UpdateContents(
			request.FormValue("editemail"),
			request.FormValue("editsubject"),
			posting.FormatMessage(request.FormValue("editmsg"), board.Dir),
			request.FormValue("editmsg"),
		); err != nil {
			gcutil.LogError(err).
				Int("postid", post.ID).
				Str("IP", gcutil.GetRealIP(request)).
				Msg("Unable to edit post")
			serverutil.ServeError(writer, "Unable to edit post: "+err.Error(), wantsJSON, map[string]interface{}{
				"postid": post.ID,
			})
			return
		}

		building.BuildBoards(false, boardid)
		building.BuildFrontPage()
		if request.FormValue("parentid") == "0" {
			http.Redirect(writer, request, "/"+board.Dir+"/res/"+strconv.Itoa(postid)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, "/"+board.Dir+"/res/"+request.FormValue("parentid")+".html#"+strconv.Itoa(postid), http.StatusFound)
		}
		return
	}
}
