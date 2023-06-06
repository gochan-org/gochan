package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var (
	chopPortNumRegex         = regexp.MustCompile(`(.+|\w+):(\d+)$`)
	ErrSpambot               = errors.New("request looks like a spambot")
	ErrBadCredentials        = errors.New("invalid username or password")
	ErrUnableToCreateSession = errors.New("unable to create login session")
)

func createSession(key, username, password string, request *http.Request, writer http.ResponseWriter) error {
	domain := request.Host
	errEv := gcutil.LogError(nil).
		Str("staff", username).
		Str("IP", gcutil.GetRealIP(request))
	defer errEv.Discard()

	domain = chopPortNumRegex.Split(domain, -1)[0]

	if !serverutil.ValidReferer(request) {
		gcutil.LogWarning().
			Str("staff", username).
			Str("IP", gcutil.GetRealIP(request)).
			Str("remoteAddr", request.Response.Request.RemoteAddr).
			Msg("Rejected login from possible spambot")
		return ErrSpambot
	}
	staff, err := gcsql.GetStaffByUsername(username, true)
	if err != nil {
		if err != sql.ErrNoRows {
			errEv.Err(err).
				Str("remoteAddr", request.RemoteAddr).
				Caller().Msg("Unrecognized username")
		}
		return ErrBadCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
	if err == bcrypt.ErrMismatchedHashAndPassword {
		// password mismatch
		errEv.Caller().
			Msg("Invalid password")
		return ErrBadCredentials
	}

	// successful login, add cookie that expires in one month
	systemCritical := config.GetSystemCriticalConfig()
	siteConfig := config.GetSiteConfig()
	maxAge, err := gcutil.ParseDurationString(siteConfig.CookieMaxAge)
	if err != nil {
		maxAge = gcutil.DefaultMaxAge
	}
	http.SetCookie(writer, &http.Cookie{
		Name:   "sessiondata",
		Value:  key,
		Path:   systemCritical.WebRoot,
		Domain: domain,
		MaxAge: int(maxAge),
	})

	if err = staff.CreateLoginSession(key); err != nil {
		gcutil.LogError(err).
			Str("staff", username).
			Str("sessionKey", key).
			Caller().Msg("Error creating new staff session")
		return ErrUnableToCreateSession
	}

	return nil
}

func getCurrentStaff(request *http.Request) (string, error) { //TODO after refactor, check if still used
	sessionCookie, err := request.Cookie("sessiondata")
	if err != nil {
		return "", err
	}
	staff, err := gcsql.GetStaffBySession(sessionCookie.Value)
	if err != nil {
		return "", err
	}
	return staff.Username, nil
}

func getCurrentFullStaff(request *http.Request) (*gcsql.Staff, error) {
	sessionCookie, err := request.Cookie("sessiondata")
	if err != nil {
		return nil, err
	}
	return gcsql.GetStaffBySession(sessionCookie.Value)
}

// GetStaffRank returns the rank number of the staff referenced in the request
func GetStaffRank(request *http.Request) int {
	staff, err := getCurrentFullStaff(request)
	if err != nil {
		return NoPerms
	}
	return staff.Rank
}

func init() {
	RegisterManagePage("actions", "Staff actions", JanitorPerms, AlwaysJSON, getStaffActions)
	RegisterManagePage("dashboard", "Dashboard", JanitorPerms, NoJSON, dashboardCallback)
	registerNoPermPages()
	registerJanitorPages()
	registerModeratorPages()
	registerAdminPages()
}

func dashboardCallback(_ http.ResponseWriter, _ *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, errEv *zerolog.Event) (interface{}, error) {
	dashBuffer := bytes.NewBufferString("")
	announcements, err := gcsql.GetAllAccouncements()
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
	if err = serverutil.MinifyTemplate(gctemplates.ManageDashboard, map[string]interface{}{
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
	rows, err := gcsql.QuerySQL(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var staff []gcsql.Staff
	for rows.Next() {
		var s gcsql.Staff
		err = rows.Scan(&s.ID, &s.Username, &s.Rank, &s.AddedOn, &s.LastLogin, &s.IsActive)
		if err != nil {
			return nil, err
		}
		staff = append(staff, s)
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

func invalidWordfilterID(id interface{}) error {
	return fmt.Errorf("wordfilter with id %q does not exist", id)
}
