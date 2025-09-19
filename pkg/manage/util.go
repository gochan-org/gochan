package manage

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"net"
	"net/http"
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

const (
	boardRequestTypeViewBoards boardRequestType = iota
	boardRequestTypeViewSingleBoard
	boardRequestTypeCancel
	boardRequestTypeCreate
	boardRequestTypeModify
	boardRequestTypeDelete
)

type boardRequestType int

func (brt boardRequestType) String() string {
	switch brt {
	case boardRequestTypeViewBoards:
		return "viewBoards"
	case boardRequestTypeViewSingleBoard:
		return "viewSingleBoard"
	case boardRequestTypeCancel:
		return "cancel"
	case boardRequestTypeCreate:
		return "create"
	case boardRequestTypeModify:
		return "modify"
	case boardRequestTypeDelete:
		return "delete"
	}
	return "unknown"
}

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
	if errors.Is(err, sql.ErrNoRows) {
		warnEv.Caller().
			Msg("Invalid username")
		return ErrBadCredentials
	}
	if err != nil {
		errEv.Err(err).Caller().
			Msg("Error getting staff by username")
		return errors.New("unable to get staff info")
	}

	err = bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		// password mismatch
		warnEv.Caller().Msg("Invalid password")
		return ErrBadCredentials
	} else if err != nil {
		errEv.Err(err).Caller().
			Str("staff", username).
			Msg("Error comparing password")
		return errors.New("unable to compare credentials")
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

func boardsRequestType(request *http.Request) boardRequestType {
	if request.PostFormValue("docreate") != "" {
		return boardRequestTypeCreate
	} else if request.FormValue("dodelete") != "" {
		return boardRequestTypeDelete
	} else if request.PostFormValue("domodify") != "" {
		return boardRequestTypeModify
	}
	if request.URL.Path != config.WebPath("/manage/boards") {
		return boardRequestTypeViewSingleBoard
	}
	return boardRequestTypeViewBoards
}

func getAllStaffNopass(activeOnly bool) ([]gcsql.Staff, error) {
	query := `SELECT id, username, global_rank, added_on, last_login, is_active FROM DBPREFIXstaff`
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

type createOrModifyBoardForm struct {
	Dir              string `form:"dir,notempty" method:"POST"`
	Title            string `form:"title,required,notempty" method:"POST"`
	Subtitle         string `form:"subtitle,required" method:"POST"`
	Description      string `form:"description,required" method:"POST"`
	Section          int    `form:"section,required" method:"POST"`
	NavBarPosition   int    `form:"navbarposition,required" method:"POST"`
	MaxFileSize      int    `form:"maxfilesize,required" method:"POST"`
	MaxThreads       int    `form:"maxthreads,required" method:"POST"`
	DefaultStyle     string `form:"defaultstyle,required,notempty" method:"POST"`
	Locked           bool   `form:"locked" method:"POST"`
	AnonName         string `form:"anonname" method:"POST"`
	AutosageAfter    int    `form:"autosageafter,required" method:"POST"`
	NoUploadsAfter   int    `form:"nouploadsafter,required" method:"POST"`
	MaxMessageLength int    `form:"maxmessagelength,required" method:"POST"`
	MinMessageLength int    `form:"minmessagelength,required" method:"POST"`
	EmbedsAllowed    bool   `form:"embedsallowed" method:"POST"`
	RedirectToThread bool   `form:"redirecttothread" method:"POST"`
	RequireFile      bool   `form:"requirefile" method:"POST"`
	EnableCatalog    bool   `form:"enablecatalog" method:"POST"`

	// create or modify submit button
	DoCreate string `form:"docreate" method:"POST"`
	DoModify string `form:"domodify" method:"POST"`
	DoDelete string `form:"dodelete" method:"POST"`
	DoCancel string `form:"docancel" method:"POST"`
}

func (brf *createOrModifyBoardForm) validate(warnEv *zerolog.Event) (err error) {
	defer func() {
		if err != nil {
			warnEv.Err(err).Caller(1).Send()
		}
	}()
	if strings.Contains(brf.Dir, "/") || strings.Contains(brf.Dir, "\\") {
		warnEv.Str("dir", brf.Dir)
		return server.NewServerError("board directory field must not contain slashes", http.StatusBadRequest)
	}
	if brf.MaxMessageLength < brf.MinMessageLength {
		warnEv.Int("maxMessageLength", brf.MaxMessageLength).Int("minMessageLength", brf.MinMessageLength)
		return server.NewServerError("maximum message length must be greater than minimum message length", http.StatusBadRequest)
	}

	return nil
}

func (brf *createOrModifyBoardForm) requestType() boardRequestType {
	if brf.DoCreate != "" {
		return boardRequestTypeCreate
	}
	if brf.DoModify != "" {
		return boardRequestTypeModify
	}
	if brf.DoDelete != "" {
		return boardRequestTypeDelete
	}
	if brf.DoCancel != "" {
		return boardRequestTypeCancel
	}
	return boardRequestTypeViewBoards
}

func (brf *createOrModifyBoardForm) fillBoard(board *gcsql.Board) {
	board.Dir = brf.Dir
	board.Title = brf.Title
	board.Subtitle = brf.Subtitle
	board.Description = brf.Description
	board.SectionID = brf.Section
	board.NavbarPosition = brf.NavBarPosition
	board.MaxFilesize = brf.MaxFileSize
	board.MaxThreads = brf.MaxThreads
	board.DefaultStyle = brf.DefaultStyle
	board.Locked = brf.Locked
	board.AnonymousName = brf.AnonName
	board.AutosageAfter = brf.AutosageAfter
	board.NoImagesAfter = brf.NoUploadsAfter
	board.MaxMessageLength = brf.MaxMessageLength
	board.MinMessageLength = brf.MinMessageLength
	board.AllowEmbeds = brf.EmbedsAllowed
	board.RedirectToThread = brf.RedirectToThread
	board.RequireFile = brf.RequireFile
	board.EnableCatalog = brf.EnableCatalog
}
