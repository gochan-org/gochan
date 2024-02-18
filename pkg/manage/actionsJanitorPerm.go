package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

// manage actions that require at least janitor-level permission go here

func logoutCallback(writer http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
	if err = gcsql.EndStaffSession(writer, request); err != nil {
		return "", err
	}
	http.Redirect(writer, request,
		config.GetSystemCriticalConfig().WebRoot+"manage",
		http.StatusSeeOther)
	return "Logged out successfully", nil
}

func clearMySessionsCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
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
}

func recentPostsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, wantsJSON bool, _, errEv *zerolog.Event) (output interface{}, err error) {
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
}

type staffOptionsJSON struct {
	FingerprintVideoThumbs bool     `json:"fingerprintVideoThumbs"`
	ImageExtensions        []string `json:"imageExtensions,omitempty"`
	VideoExtensions        []string `json:"videoExtensions,omitempty"`
}

func staffOptionsCallback(_ http.ResponseWriter, _ *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
	staffOptions := staffOptionsJSON{}
	if staff.Rank > JanitorPerms {
		staffOptions.FingerprintVideoThumbs = config.GetSiteConfig().FingerprintVideoThumbnails
		staffOptions.ImageExtensions = uploads.ImageExtensions
		staffOptions.VideoExtensions = uploads.VideoExtensions
	}
	return staffOptions, nil
}

func announcementsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
	// return an array of announcements (with staff name instead of ID) and any errors
	return getAllAnnouncements()
}

func staffCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	var outputStr string
	do := request.FormValue("do")
	allStaff, err := getAllStaffNopass(true)
	if wantsJSON {
		if err != nil {
			errEv.Err(err).Caller().Msg("Failed getting staff list")
		}
		return allStaff, err
	}
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed getting staff list")
		err = errors.New("Error getting staff list: " + err.Error())
		return "", err
	}

	updateUsername := request.FormValue("update")
	username := request.FormValue("username")
	password := request.FormValue("password")
	passwordConfirm := request.FormValue("passwordconfirm")
	if (do == "add" || do == "update") && password != passwordConfirm {
		return "", ErrPasswordConfirm
	}

	rankStr := request.FormValue("rank")
	var rank int
	if rankStr != "" {
		if rank, err = strconv.Atoi(rankStr); err != nil {
			errEv.Err(err).Caller().
				Str("rank", rankStr).Send()
			return "", err
		}
	}

	if do == "add" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			errEv.Err(ErrInsufficientPermission).Caller().
				Int("rank", staff.Rank).Send()
			return "", ErrInsufficientPermission
		}
		if _, err = gcsql.NewStaff(username, password, rank); err != nil {
			errEv.Caller().
				Str("newStaff", username).
				Str("newPass", password).
				Int("newRank", rank).
				Msg("Error creating new staff account")
			return "", fmt.Errorf("Error creating new staff account %q by %q: %s",
				username, staff.Username, err.Error())
		}
	} else if do == "del" && username != "" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			errEv.Err(ErrInsufficientPermission).Caller().
				Int("rank", staff.Rank).Send()
			return "", ErrInsufficientPermission
		}
		if err = gcsql.DeactivateStaff(username); err != nil {
			errEv.Err(err).Caller().
				Str("delStaff", username).
				Msg("Error deleting staff account")
			return "", fmt.Errorf("Error deleting staff account %q by %q: %s",
				username, staff.Username, err.Error())
		}
	} else if do == "update" && updateUsername != "" {
		if staff.Username != updateUsername && staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			errEv.Err(ErrInsufficientPermission).Caller().
				Int("rank", staff.Rank).Send()
			return "", ErrInsufficientPermission
		}
		if err = gcsql.UpdatePassword(updateUsername, password); err != nil {
			errEv.Err(err).Caller().
				Str("updateStaff", username).
				Msg("Error updating password")
			return "", err
		}
	}
	if do == "add" || do == "del" {
		allStaff, err = getAllStaffNopass(true)
		if err != nil {
			errEv.Err(err).Caller().Msg("Error getting updated staff list")
			err = errors.New("Error getting updated staff list: " + err.Error())
			return "", err
		}
	}

	staffBuffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, map[string]interface{}{
		"do":             do,
		"updateUsername": updateUsername,
		"allstaff":       allStaff,
		"currentStaff":   staff,
	}, staffBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_staff.html").Send()
		return "", errors.New("Error executing staff management page template: " + err.Error())
	}
	outputStr += staffBuffer.String()
	return outputStr, nil
}

func registerJanitorPages() {
	actions = append(actions,
		Action{
			ID:          "logout",
			Title:       "Logout",
			Permissions: JanitorPerms,
			Callback:    logoutCallback,
		},
		Action{
			ID:          "clearmysessions",
			Title:       "Log me out everywhere",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback:    clearMySessionsCallback,
		},
		Action{
			ID:          "recentposts",
			Title:       "Recent posts",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback:    recentPostsCallback,
		},
		Action{
			ID:          "announcements",
			Title:       "Announcements",
			Permissions: JanitorPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    announcementsCallback,
		},
		Action{
			ID:          "staff",
			Title:       "Staff",
			Permissions: JanitorPerms,
			JSONoutput:  OptionalJSON,
			Callback:    staffCallback,
		},
		Action{
			ID:          "staffoptions",
			Title:       "Staff-specific options",
			Permissions: JanitorPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    staffOptionsCallback,
		},
	)
}
