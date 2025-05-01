package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Eggbertx/go-forms"
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
			errEv.Err(err).Caller().
				Str("limit", limitStr).
				Msg("Invalid limit value")
			return "", err
		}
	}
	boardidStr := request.FormValue("boardid")
	var recentposts []*building.Post
	var boardid int
	if boardidStr != "" {
		if boardid, err = strconv.Atoi(boardidStr); err != nil {
			errEv.Err(err).Caller().
				Str("boardid", boardidStr).
				Msg("Invalid boardid value")
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
	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageRecentPosts, map[string]any{
		"recentposts": recentposts,
		"allBoards":   gcsql.AllBoards,
		"boardid":     boardid,
		"limit":       limit,
	}, &buf, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return "", fmt.Errorf("failed executing ban management page template: %w", err)
	}
	return buf.String(), nil
}

func announcementsCallback(_ http.ResponseWriter, _ *http.Request, _ *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (output any, err error) {
	// return an array of announcements (with staff name instead of ID) and any errors
	return getAllAnnouncements()
}

type formMode int

func (fmv formMode) String() string {
	switch fmv {
	case changePasswordForm:
		return "Change Password"
	case changeRankForm:
		return "Change User Rank"
	case newUserForm:
		return "Add New User"
	}
	return ""
}

const (
	noForm formMode = iota
	changePasswordForm
	changeRankForm
	newUserForm
)

type staffForm struct {
	Do                    string `form:"do"`
	ChangePasswordForUser string `form:"changepass" method:"GET"`
	ChangeRankForUser     string `form:"changerank" method:"GET"`
	Username              string `form:"username"`
	Password              string `form:"password" method:"POST"`
	PasswordConfirm       string `form:"passwordconfirm" method:"POST"`
	Rank                  int    `form:"rank" method:"POST"`
}

func (s *staffForm) validate(staff *gcsql.Staff, warnEv *zerolog.Event) (formMode, error) {
	if s.Do == "add" || (s.Do == "changepass" && s.Username != staff.Username) || s.Do == "changerank" || s.Do == "del" {
		if staff.Rank < 3 {
			warnEv.Caller().
				Str("username", s.Username).
				Str("do", s.Do).
				Msg("non-admin tried to modify someone else's account or create a new account")
			return noForm, ErrInsufficientPermission
		}
	}

	if (s.Do == "del" || s.Do == "add") && s.Username == "" {
		warnEv.Caller().Str("do", s.Do).Msg("Missing username field")
		return noForm, errors.New("missing username field")
	}

	if s.Do == "add" && s.Password == "" {
		warnEv.Caller().Str("do", s.Do).Msg("Missing password field")
		return noForm, errors.New("missing password field")
	}
	if s.Do == "add" && s.Password != s.PasswordConfirm {
		warnEv.Caller().Str("do", s.Do).Err(ErrPasswordsDoNotMatch).Send()
		return noForm, ErrPasswordsDoNotMatch
	}

	if s.Do != "" && s.Do != "add" && s.Do != "changepass" && s.Do != "changerank" && s.Do != "del" {
		warnEv.Caller().Str("do", s.Do).Msg("Invalid form action")
		return noForm, errors.New("invalid form action")
	}

	if s.ChangePasswordForUser != "" {
		if s.ChangePasswordForUser != staff.Username && staff.Rank < 3 {
			warnEv.Caller().Str("username", s.Username).Msg("non-admin tried to change a password")
			return noForm, ErrInsufficientPermission
		}
		return changePasswordForm, nil
	}
	if s.ChangeRankForUser != "" {
		if staff.Rank < 3 {
			warnEv.Caller().Str("username", s.Username).Msg("non-admin tried to change a rank")
			return noForm, ErrInsufficientPermission
		}
		return changeRankForm, nil
	}
	if staff.Rank >= 3 {
		return newUserForm, nil
	}
	return noForm, nil
}

func staffCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error) {
	var allStaff []gcsql.Staff
	if wantsJSON {
		allStaff, err = getAllStaffNopass(true)
		if err != nil {
			errEv.Err(err).Caller().Msg("Failed getting staff list")
			return nil, errors.New("unable to get staff list")
		}
		return allStaff, nil
	}

	warnEv := gcutil.LogWarning().
		Str("IP", gcutil.GetRealIP(request)).
		Str("userAgent", request.UserAgent()).
		Str("staff", staff.Username)
	defer warnEv.Discard()

	var form staffForm
	var numErr *strconv.NumError
	err = forms.FillStructFromForm(request, &form)
	if errors.As(err, &numErr) {
		errEv.Err(err).Caller().
			Str("value", numErr.Num).
			Msg("Error parsing form value")
		writer.WriteHeader(http.StatusBadRequest)
		return "", err
	} else if err != nil {
		errEv.Err(err).Caller().
			Str("form", "staffForm").
			Msg("Error filling form struct")
		return "", err
	}
	formMode, err := form.validate(staff, warnEv)
	if errors.Is(err, ErrInsufficientPermission) {
		writer.WriteHeader(http.StatusForbidden)
		return "", err
	}
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return "", err
	}

	if form.Username != "" {
		gcutil.LogStr("username", form.Username, infoEv, errEv, warnEv)
	}

	updateStaff := &gcsql.Staff{
		Username: form.Username,
		Rank:     form.Rank,
	}
	switch formMode {
	case changePasswordForm:
		updateStaff, err = gcsql.GetStaffByUsername(form.ChangePasswordForUser, true)
		if err != nil {
			errEv.Err(err).Caller().
				Str("username", form.ChangePasswordForUser).
				Msg("Error getting staff account")
			return "", err
		}
	case changeRankForm:
		updateStaff, err = gcsql.GetStaffByUsername(form.ChangeRankForUser, true)
		if err != nil {
			errEv.Err(err).Caller().
				Str("username", form.ChangeRankForUser).
				Msg("Error getting staff account")
			return "", err
		}
	case newUserForm:
		updateStaff.Username = form.Username
	}

	switch form.Do {
	case "add":
		if updateStaff, err = gcsql.NewStaff(form.Username, form.Password, form.Rank); err != nil {
			errEv.Err(err).Caller().
				Str("username", form.Username).
				Msg("Error creating new staff account")
			return "", fmt.Errorf("unable to create new staff account: %w", err)
		}
		infoEv.Str("userRank", updateStaff.RankTitle()).Msg("New staff account created")
	case "changepass":
		if err = updateStaff.UpdatePassword(form.Password); err != nil {
			errEv.Err(err).Caller().Msg("Error updating password")
			return "", errors.New("unable to change staff account password")
		}
		infoEv.Msg("Password updated")
	case "changerank":
		if err = updateStaff.UpdateRank(form.Rank); err != nil {
			errEv.Err(err).Caller().Msg("Error updating rank")
			return "", errors.New("unable to change staff account rank")
		}
		infoEv.
			Int("rank", updateStaff.Rank).
			Str("rankTitle", updateStaff.RankTitle()).
			Msg("Staff account rank updated")
	case "del":
		if err = updateStaff.ClearSessions(); err != nil {
			errEv.Err(err).Caller().
				Str("username", form.Username).
				Msg("Unable to clear user login sessions")
			return "", errors.New("unable to clear user login sessions")
		}
		if err = updateStaff.SetActive(false); err != nil {
			errEv.Err(err).Caller().
				Str("username", form.Username).
				Msg("Unable to deactivate user")
			return "", errors.New("unable to deactivate user")
		}
		infoEv.Str("userRank", updateStaff.RankTitle()).Msg("Account deactivated")
	}

	data := map[string]any{
		"username":     updateStaff.Username,
		"rank":         updateStaff.Rank,
		"currentStaff": staff,
		"formMode":     formMode,
	}

	data["allstaff"], err = getAllStaffNopass(true)
	if err != nil {
		errEv.Err(err).Caller().Msg("Failed getting staff list")
		return nil, errors.New("unable to get staff list")
	}

	buffer := bytes.NewBufferString("")
	if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, data, buffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_staff.html").Send()
		writer.WriteHeader(http.StatusInternalServerError)
		return "", errors.New("unable to execute staff management page template")
	}
	return buffer.String(), nil
}

func registerJanitorPages() {
	RegisterManagePage("logout", "Logout", JanitorPerms, NoJSON, logoutCallback)
	RegisterManagePage("clearmysessions", "Log me out everywhere", JanitorPerms, OptionalJSON, clearMySessionsCallback)
	RegisterManagePage("recentposts", "Recent posts", JanitorPerms, OptionalJSON, recentPostsCallback)
	RegisterManagePage("announcements", "Announcements", JanitorPerms, AlwaysJSON, announcementsCallback)
	RegisterManagePage("staff", "Staff", JanitorPerms, OptionalJSON, staffCallback)
}
