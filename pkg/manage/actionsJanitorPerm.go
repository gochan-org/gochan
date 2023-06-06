package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"net/http"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

// manage actions that require at least janitor-level permission go here

func registerJanitorPages() {
	actions = append(actions,
		Action{
			ID:          "logout",
			Title:       "Logout",
			Permissions: JanitorPerms,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				if err = gcsql.EndStaffSession(writer, request); err != nil {
					return "", err
				}
				http.Redirect(writer, request,
					config.GetSystemCriticalConfig().WebRoot+"manage",
					http.StatusSeeOther)
				return "Logged out successfully", nil
			}},
		Action{
			ID:          "clearmysessions",
			Title:       "Log me out everywhere",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				session, err := request.Cookie("sessiondata")
				if err != nil {
					// doesn't have a login session cookie, return with no errors
					if !wantsJSON {
						http.Redirect(writer, request,
							config.GetSystemCriticalConfig().WebRoot+"manage",
							http.StatusSeeOther)
						return
					}
					return "You are not logged in", nil
				}

				_, err = gcsql.GetStaffBySession(session.Value)
				if err != nil {
					// staff session doesn't exist, probably a stale cookie
					if !wantsJSON {
						http.Redirect(writer, request,
							config.GetSystemCriticalConfig().WebRoot+"manage",
							http.StatusSeeOther)
						return
					}
					return "You are not logged in", err
				}
				if err = staff.ClearSessions(); err != nil && err != sql.ErrNoRows {
					// something went wrong when trying to clean out sessions for this user
					return nil, err
				}
				serverutil.DeleteCookie(writer, request, "sessiondata")
				gcutil.LogAccess(request).
					Str("clearSessions", staff.Username).
					Send()
				if !wantsJSON {
					http.Redirect(writer, request,
						config.GetSystemCriticalConfig().WebRoot+"manage",
						http.StatusSeeOther)
					return "", nil
				}
				return "Logged out successfully", nil
			}},
		Action{
			ID:          "recentposts",
			Title:       "Recent posts",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
				limit := 20
				limitStr := request.FormValue("limit")
				if limitStr != "" {
					limit, err = strconv.Atoi(limitStr)
					if err != nil {
						errEv.Err(err).Caller().Send()
						return "", err
					}
				}
				boardidStr := request.FormValue("boardid")
				var recentposts []building.Post
				var boardid int
				if boardidStr != "" {
					if boardid, err = strconv.Atoi(boardidStr); err != nil {
						errEv.Err(err).Caller().Send()
						return "", err
					}
				}
				recentposts, err = building.GetRecentPosts(boardid, limit)
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				if wantsJSON {
					return recentposts, nil
				}
				manageRecentsBuffer := bytes.NewBufferString("")
				if err = serverutil.MinifyTemplate(gctemplates.ManageRecentPosts, map[string]interface{}{
					"recentposts": recentposts,
					"allBoards":   gcsql.AllBoards,
					"boardid":     boardid,
					"limit":       limit,
				}, manageRecentsBuffer, "text/html"); err != nil {
					errEv.Err(err).Caller().Send()
					return "", errors.New("Error executing ban management page template: " + err.Error())
				}
				return manageRecentsBuffer.String(), nil
			}},
		Action{
			ID:          "announcements",
			Title:       "Announcements",
			Permissions: JanitorPerms,
			JSONoutput:  AlwaysJSON,
			Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
				// return an array of announcements and any errors
				return gcsql.GetAllAccouncements()
			}},
	)
}
