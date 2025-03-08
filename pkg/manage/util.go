package manage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrBadCredentials        = errors.New("invalid username or password")
	ErrUnableToCreateSession = errors.New("unable to create login session")
	ErrInvalidSession        = errors.New("invalid staff session")
	dashboardAction          = Action{
		ID:          "dashboard",
		Title:       "Dashboard",
		Permissions: JanitorPerms,
		Callback:    dashboardCallback,
	}
)

func createSession(key, username, password string, request *http.Request, writer http.ResponseWriter) error {
	domain := request.Host
	infoEv, warnEv, errEv := gcutil.LogRequest(request)
	defer gcutil.LogDiscard(infoEv, warnEv, errEv)
	gcutil.LogStr("staff", username, infoEv, warnEv, errEv)

	if strings.Contains(domain, ":") {
		domain, _, err := net.SplitHostPort(domain)
		if err != nil {
			warnEv.Err(err).Caller().Str("host", domain).Send()
			return server.NewServerError("Invalid request host", http.StatusBadRequest)
		}
	}

	refererResult, err := serverutil.CheckReferer(request)
	if err != nil {
		warnEv.Err(err).Caller().
			Str("referer", request.Referer()).
			Msg("Invalid referer")
		return err
	}
	if refererResult != serverutil.InternalReferer {
		warnEv.
			Int("refererResult", int(refererResult)).
			Str("referer", request.Referer()).
			Str("SiteHost", config.GetSystemCriticalConfig().SiteHost).
			Msg("Rejected login from possible spambot")
		return serverutil.ErrSpambot
	}

	staff, err := gcsql.GetStaffByUsername(username, true)
	if err != nil {
		if err != sql.ErrNoRows {
			errEv.Err(err).Caller().
				Msg("Unrecognized username")
		}
		return ErrBadCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		// password mismatch
		warnEv.Caller().Msg("Invalid password")
		return ErrBadCredentials
	}

	// successful login, add cookie that expires in one month
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()
	expirationDur, err := durationutil.ParseLongerDuration(siteConfig.StaffSessionDuration)
	if err != nil {
		expirationDur = gcutil.DefaultMaxAge
	}
	http.SetCookie(writer, &http.Cookie{
		Name:     "sessiondata",
		Value:    key,
		Path:     systemCritical.WebRoot,
		Domain:   domain,
		Expires:  time.Now().Add(expirationDur),
		SameSite: http.SameSiteStrictMode,
	})

	if err = staff.CreateLoginSession(key); err != nil {
		errEv.Err(err).Caller().
			Str("staff", username).
			Str("sessionKey", key).
			Msg("Error creating new staff session")
		return ErrUnableToCreateSession
	}

	return nil
}

func getCurrentStaff(request *http.Request) (string, error) {
	staff, err := gcsql.GetStaffFromRequest(request)
	if err != nil {
		return "", err
	}
	return staff.Username, nil
}

// GetStaffFromRequest returns the staff making the request. If the request does not have
// a staff cookie, it will return a staff object with rank 0.
// Deprecated: use gcsql.GetStaffFromRequest
func GetStaffFromRequest(request *http.Request) (*gcsql.Staff, error) {
	return gcsql.GetStaffFromRequest(request)
}

// GetStaffRank returns the rank number of the staff referenced in the request
func GetStaffRank(request *http.Request) int {
	staff, err := gcsql.GetStaffFromRequest(request)
	if err != nil {
		return NoPerms
	}
	return staff.Rank
}

// InitManagePages sets up the built-in manage pages
func InitManagePages() {
	RegisterManagePage("actions", "Staff actions", JanitorPerms, AlwaysJSON, getStaffActions)
	RegisterManagePage(dashboardAction.ID, dashboardAction.Title, dashboardAction.Permissions, dashboardAction.JSONoutput, dashboardAction.Callback)
	server.GetRouter().GET(config.WebPath("/manage"), setupManageFunction(&dashboardAction))
	registerNoPermPages()
	registerJanitorPages()
	registerModeratorPages()
	registerAdminPages()
}

func dashboardCallback(_ http.ResponseWriter, _ *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (any, error) {
	dashBuffer := bytes.NewBufferString("")
	announcements, err := getAllAnnouncements()
	if err != nil {
		return nil, err
	}
	rankString := ""
	switch staff.Rank {
	case AdminPerms:
		rankString = "administrator"
	case ModPerms:
		rankString = "moderator"
	case JanitorPerms:
		rankString = "janitor"
	}

	availableActions := getAvailableActions(staff.Rank, true)
	if err = serverutil.MinifyTemplate(gctemplates.ManageDashboard, map[string]any{
		"actions":       availableActions,
		"rank":          staff.Rank,
		"rankString":    rankString,
		"announcements": announcements,
		"boards":        gcsql.AllBoards,
	}, dashBuffer, "text/html"); err != nil {
		errEv.Err(err).Str("template", "manage_dashboard.html").Caller().Send()
		return "", err
	}
	return dashBuffer.String(), nil
}

// bordsRequestType takes the request and returns "cancel", "create", "delete",
// "edit", or "modify" and the board's ID according to the request
func boardsRequestType(request *http.Request) (string, int, error) {
	var requestType string
	var boardID int
	var err error
	if request.FormValue("docancel") != "" {
		requestType = "cancel"
	} else if request.FormValue("docreate") != "" {
		requestType = "create"
	} else if request.FormValue("dodelete") != "" {
		requestType = "delete"
	} else if request.FormValue("doedit") != "" {
		requestType = "edit"
	} else if request.FormValue("domodify") != "" {
		requestType = "modify"
	}
	boardIDstr := request.FormValue("board")
	if boardIDstr != "" {
		boardID, err = strconv.Atoi(boardIDstr)
	}
	return requestType, boardID, err
}

func getAllStaffNopass(activeOnly bool) ([]gcsql.Staff, error) {
	query := `SELECT
	id, username, global_rank, added_on, last_login, is_active
	FROM DBPREFIXstaff`
	if activeOnly {
		query += " WHERE is_active"
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.GetSQLConfig().DBTimeoutSeconds)*time.Second)
	rows, err := gcsql.Query(&gcsql.RequestOptions{Context: ctx, Cancel: cancel}, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		rows.Close()
		cancel()
	}()
	var staff []gcsql.Staff
	for rows.Next() {
		var s gcsql.Staff
		err = rows.Scan(&s.ID, &s.Username, &s.Rank, &s.AddedOn, &s.LastLogin, &s.IsActive)
		if err != nil {
			return nil, err
		}
		staff = append(staff, s)
	}
	if err = rows.Close(); err != nil {
		return nil, err
	}
	return staff, nil
}

// getBoardDataFromForm parses the relevant form fields into the board and returns any errors for invalid string to int
// or missing required fields. It should only be used for editing and creating boards
func getBoardDataFromForm(board *gcsql.Board, request *http.Request) error {
	requestType, _, _ := boardsRequestType(request)

	staff, err := getCurrentStaff(request)
	if err != nil {
		return err
	}

	if len(request.Form["domodify"]) > 0 || len(request.Form["doedit"]) > 0 || len(request.Form["dodelete"]) > 0 {
		if board.ID, err = getIntField("board", staff, request, 1); err != nil {
			return err
		}
	}
	if board.SectionID, err = getIntField("section", staff, request, 1); err != nil {
		return err
	}
	if requestType == "create" {
		if board.Dir, err = getStringField("dir", staff, request, 1); err != nil {
			return err
		}
	}
	if board.NavbarPosition, err = getIntField("navbarposition", staff, request, 1); err != nil {
		return err
	}
	if board.Title, err = getStringField("title", staff, request, 1); err != nil {
		return err
	}
	if board.Subtitle, err = getStringField("subtitle", staff, request, 1); err != nil {
		return err
	}
	if board.Description, err = getStringField("description", staff, request, 1); err != nil {
		return err
	}
	if board.MaxFilesize, err = getIntField("maxfilesize", staff, request, 1); err != nil {
		return err
	}
	if board.MaxThreads, err = getIntField("maxthreads", staff, request, 1); err != nil {
		return err
	}
	if board.DefaultStyle, err = getStringField("defaultstyle", staff, request, 1); err != nil {
		return err
	}
	board.Locked = request.FormValue("locked") == "on"
	if board.AnonymousName, err = getStringField("anonname", staff, request, 1); err != nil {
		return err
	}
	if board.AnonymousName == "" {
		board.AnonymousName = "Anonymous"
	}
	board.ForceAnonymous = request.FormValue("forcedanonymous") == "on"
	if board.AutosageAfter, err = getIntField("autosageafter", staff, request, 1); err != nil {
		return err
	}
	if board.AutosageAfter < 1 {
		board.AutosageAfter = 200
	}
	if board.NoImagesAfter, err = getIntField("nouploadsafter", staff, request, 1); err != nil {
		return err
	}
	if board.MaxMessageLength, err = getIntField("maxmessagelength", staff, request, 1); err != nil {
		return err
	}
	if board.MaxMessageLength < 1 {
		board.MaxMessageLength = 1024
	}
	if board.MinMessageLength, err = getIntField("minmessagelength", staff, request, 1); err != nil {
		return err
	}
	board.AllowEmbeds = request.FormValue("allowembeds") == "on"
	board.RedirectToThread = request.FormValue("redirecttothread") == "on"
	board.RequireFile = request.FormValue("requirefile") == "on"
	board.EnableCatalog = request.FormValue("enablecatalog") == "on"

	return nil
}
