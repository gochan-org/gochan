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

func logoutCallback(writer http.ResponseWriter, request *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output any, err error) {
	if err = gcsql.EndStaffSession(writer, request); err != nil {
		return "", err
	}
	http.Redirect(writer, request,
		config.GetSystemCriticalConfig().WebRoot+"manage",
		http.StatusSeeOther)
	return "Logged out successfully", nil
}

func clearMySessionsCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, _ *zerolog.Event, _ *zerolog.Event) (output any, err error) {
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

func recentPostsCallback(_ http.ResponseWriter, request *http.Request, _ *gcsql.Staff, wantsJSON bool, _, errEv *zerolog.Event) (output any, err error) {
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
	if err = serverutil.MinifyTemplate(gctemplates.ManageRecentPosts, map[string]any{
		"recentposts": recentposts,
		"allBoards":   gcsql.AllBoards,
		"boardid":     boardid,
		"limit":       limit,
	}, manageRecentsBuffer, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return "", fmt.Errorf("failed executing ban management page template: %w", err)
	}
	return manageRecentsBuffer.String(), nil
}

func announcementsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output any, err error) {
	// return an array of announcements (with staff name instead of ID) and any errors
	return getAllAnnouncements()
}

type formMode int

func (fmv formMode) String() string {
	switch fmv {
	case updateOwnPasswordForm:
		return "Update Password"
	case updateUserForm:
		return "Update User"
	case newUserForm:
		return "Add New User"
	}
	return ""
}

const (
	noForm formMode = iota
	updateOwnPasswordForm
	updateUserForm
	newUserForm
)

func staffCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error) {
	var allStaff []gcsql.Staff
	allStaff, err = getAllStaffNopass(true)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed getting staff list")
		return nil, errors.New("unable to get staff list")
	}
	if wantsJSON {
		return allStaff, nil
	}

	warnEv := gcutil.LogWarning().
		Str("IP", gcutil.GetRealIP(request)).
		Str("userAgent", request.UserAgent()).
		Str("staff", staff.Username)
	defer warnEv.Discard()

	do := request.PostFormValue("do")
	updateUsername := request.FormValue("update")
	username := request.PostFormValue("username")
	password := request.PostFormValue("password")
	passwordConfirm := request.FormValue("passwordconfirm")
	if (do == "add" || do == "update") && password != passwordConfirm {
		return "", ErrPasswordsDoNotMatch
	}
	if username != "" {
		gcutil.LogStr("username", username, infoEv, errEv, warnEv)
	}

	rankStr := request.PostFormValue("rank")
	var rank int
	if rankStr != "" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			warnEv.Caller().Str("username", username).Msg("non-admin tried to modify a staff account's rank")
			return "", ErrInsufficientPermission
		}
		if rank, err = strconv.Atoi(rankStr); err != nil {
			errEv.Err(err).Caller().
				Str("rank", rankStr).Send()
			return "", err
		}
		if rank < 0 || rank > 3 {
			errEv.Caller().Int("rank", rank).Send()
			return "", errors.New("invalid rank")
		}
	}

	var formMode = noForm
	if staff.Rank == 3 {
		if updateUsername == "" {
			formMode = newUserForm
		} else {
			formMode = updateUserForm
		}
	} else {
		if updateUsername == staff.Username {
			formMode = updateOwnPasswordForm
		} else if updateUsername != "" {
			// user is a moderator or janitor and is trying to update someone else's account
			writer.WriteHeader(http.StatusUnauthorized)
			return nil, ErrInsufficientPermission
		}
	}

	data := map[string]any{
		"updateUsername": updateUsername,
		"currentStaff":   staff,
		"formMode":       formMode,
		"updateRank":     -1,
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
		gcutil.LogStr("updateUsername", updateUsername, infoEv, errEv, warnEv)
		if !found {
			writer.WriteHeader(http.StatusBadRequest)
			warnEv.Err(gcsql.ErrUnrecognizedUsername).Caller().Send()
			return "", gcsql.ErrUnrecognizedUsername
		}
	}

	if do == "add" {
		if staff.Rank < 3 {
			writer.WriteHeader(http.StatusUnauthorized)
			warnEv.Caller().Str("username", username).Msg("non-admin tried to create a new account")
			return "", ErrInsufficientPermission
		}
		var newStaff *gcsql.Staff
		if newStaff, err = gcsql.NewStaff(username, password, rank); err != nil {
			errEv.Err(err).Caller().
				Str("userRank", newStaff.RankTitle()).
				Msg("Error creating new staff account")
			return "", fmt.Errorf("unable to create new staff account %q by %q: %s",
				username, staff.Username, err.Error())
		}
		infoEv.Str("userRank", newStaff.RankTitle()).Msg("New staff account created")
	} else if do == "update" || do == "del" {
		if updateUsername == "" {
			warnEv.Caller().Str("do", do).Msg("Missing username field")
			return nil, errors.New("missing username field")
		}
		if (do == "update" && staff.Rank < AdminPerms && updateUsername != staff.Username) || (do == "del" && staff.Rank < AdminPerms) {
			// user is not an admin and is trying to update someone else's account (rank change already checked)
			writer.WriteHeader(http.StatusUnauthorized)
			warnEv.Err(ErrInsufficientPermission).Send()
			return nil, ErrInsufficientPermission
		}

		var user *gcsql.Staff
		if user, err = gcsql.GetStaffByUsername(updateUsername, true); err != nil {
			errEv.Err(err).Caller().Bool("onlyActive", true).Msg("Unable to get staff by username")
			return nil, err
		}
		gcutil.LogStr("userRank", user.RankTitle(), infoEv, errEv, warnEv)

		if do == "update" {
			if password != "" {
				if err = user.UpdatePassword(password); err != nil {
					writer.WriteHeader(http.StatusInternalServerError)
					errEv.Err(err).Caller().
						Msg("Error updating password")
					return "", errors.New("unable to update staff account password")
				}
				infoEv.Msg("Password updated")
			} else if rank > 0 {
				if err = user.UpdateRank(rank); err != nil {
					writer.WriteHeader(http.StatusInternalServerError)
					errEv.Err(err).Caller().
						Msg("Error updating rank")
					return "", errors.New("unable to update staff account rank")
				}
				infoEv.
					Int("rank", user.Rank).
					Str("rankTitle", user.RankTitle()).
					Msg("Staff account rank updated")
			}
			data["formMode"] = newUserForm
			data["updateUsername"] = ""
			data["updateRank"] = -1
		} else {
			// deactivate account
			if err = user.ClearSessions(); err != nil {
				errEv.Err(err).Caller().
					Msg("Unable to clear user login sessions")
				return nil, errors.New("unable to clear user login sessions")
			}

			if err = user.SetActive(false); err != nil {
				errEv.Err(err).Caller().
					Msg("Unable to deactivate user")
				return nil, errors.New("unable to deactivate user")
			}
			infoEv.Msg("Account deactivated")
		}

	}

	data["allstaff"], err = getAllStaffNopass(true)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed getting staff list")
		return nil, errors.New("unable to get staff list")
	}

	staffBuffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, data, staffBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_staff.html").Send()
		writer.WriteHeader(http.StatusInternalServerError)
		return "", errors.New("unable to execute staff management page template")
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
