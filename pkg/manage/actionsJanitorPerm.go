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
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

var (
	ErrPasswordsDoNotMatch = errors.New("passwords do not match")
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
	var recentposts []*building.Post
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

func announcementsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output interface{}, err error) {
	// return an array of announcements (with staff name instead of ID) and any errors
	return getAllAnnouncements()
}

func staffCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
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
	warnEv := gcutil.LogWarning().
		Str("IP", gcutil.GetRealIP(request)).
		Str("userAgent", request.UserAgent()).
		Str("staff", staff.Username)
	defer warnEv.Discard()

	updateUsername := request.FormValue("update")
	username := request.PostFormValue("username")
	password := request.PostFormValue("password")
	passwordConfirm := request.FormValue("passwordconfirm")
	if (do == "add" || do == "update") && password != passwordConfirm {
		return "", ErrPasswordsDoNotMatch
	}

	rankStr := request.PostFormValue("rank")
	var rank int
	if rankStr != "" {
		if rank, err = strconv.Atoi(rankStr); err != nil {
			errEv.Err(err).Caller().
				Str("rank", rankStr).Send()
			return "", err
		}
	}

	data := map[string]any{
		"do":             do,
		"updateUsername": updateUsername,
		"allstaff":       allStaff,
		"currentStaff":   staff,
	}
	if updateUsername != "" && staff.Rank == AdminPerms {
		var found bool
		for _, user := range allStaff {
			if user.Username == updateUsername {
				data["updateRank"] = user.Rank
				found = true
				break
			}
		}
		if !found {
			writer.WriteHeader(http.StatusBadRequest)
			errEv.Err(gcsql.ErrUnrecognizedUsername).Caller().Str("username", updateUsername).Send()
			return "", gcsql.ErrUnrecognizedUsername
		}
	}

	if do == "add" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			warnEv.Caller().Str("username", username).Msg("non-admin tried to create a new account")
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
			warnEv.Msg("non-admin tried to deactivate an account")
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
		if (staff.Username != updateUsername || rank > 0) && staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			warnEv.Caller().Str("username", username).Msg("non-admin tried to modify a staff account's rank")
			return "", ErrInsufficientPermission
		}
		if rank > 0 {
			err = gcsql.UpdateStaff(updateUsername, rank, password)
		} else {
			err = gcsql.UpdatePassword(updateUsername, password)
		}
		if err != nil {
			logRank := rank
			if logRank == 0 {
				// user does not have admin rank and is updating their own account
				logRank = staff.Rank
			}
			errEv.Err(err).Caller().
				Str("updateStaff", username).
				Int("updateRank", logRank).
				Msg("Error updating account")
			writer.WriteHeader(http.StatusInternalServerError)
			return "", errors.New("unable to update staff account")
		}
	} else if do == "add" || do == "del" {
		allStaff, err = getAllStaffNopass(true)
		if err != nil {
			errEv.Err(err).Caller().Msg("Error getting updated staff list")
			writer.WriteHeader(http.StatusInternalServerError)
			err = errors.New("Unable to get updated staff list")
			return "", err
		}
	}

	staffBuffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, data, staffBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_staff.html").Send()
		writer.WriteHeader(http.StatusInternalServerError)
		return "", errors.New("Unable to execute staff management page template")
	}
	return staffBuffer.String(), nil
}

func registerJanitorPages() {
	RegisterManagePage("logout", "Logout", JanitorPerms, NoJSON, logoutCallback)
	RegisterManagePage("clearmysessions", "Log me out everywhere", JanitorPerms, OptionalJSON, clearMySessionsCallback)
	RegisterManagePage("recentposts", "Recent posts", JanitorPerms, OptionalJSON, recentPostsCallback)
	RegisterManagePage("announcements", "Announcements", JanitorPerms, AlwaysJSON, announcementsCallback)
	RegisterManagePage("staff", "Staff", JanitorPerms, OptionalJSON, staffCallback)
}
