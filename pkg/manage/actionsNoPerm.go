package manage

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

const (
	loginTitle = "Login"
)

type loginRedirectAction string

func loginCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	systemCritical := config.GetSystemCriticalConfig()
	if staff.Rank > 0 {
		http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage"), http.StatusFound)
	}
	username := request.PostFormValue("username")
	password := request.PostFormValue("password")
	redirectAction := request.PostFormValue("redirect")
	if redirectAction == "" {
		loginRefererAny := request.Context().Value(loginRedirectAction("redirect"))

		if loginRefererAny != nil {
			redirectAction = loginRefererAny.(string)
		}
	}

	if username == "" || password == "" {
		//assume that they haven't logged in
		manageLoginBuffer := bytes.NewBufferString("")
		if err = serverutil.MinifyTemplate(gctemplates.ManageLogin, map[string]any{
			"siteConfig":  config.GetSiteConfig(),
			"sections":    gcsql.AllSections,
			"boards":      gcsql.AllBoards,
			"boardConfig": config.GetBoardConfig(""),
			"redirect":    redirectAction,
		}, manageLoginBuffer, "text/html"); err != nil {
			logger.Err(err).Str("template", "manage_login.html").Send()
			return "", fmt.Errorf("failed executing staff login page template: %w", err)
		}
		output = manageLoginBuffer.String()
	} else {
		key := gcutil.Md5Sum(request.RemoteAddr + username + password + systemCritical.RandomSeed + gcutil.RandomString(3))[0:10]
		if err = createSession(key, username, password, request, writer); err != nil {
			if errors.Is(err, ErrBadCredentials) {
				writer.WriteHeader(http.StatusUnauthorized)
			}
			return "", err
		}
		logger.Info().
			Str("redirectAction", redirectAction).
			Str("username", username).
			Msg("Logged in, redirecting to manage page")
		http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage/"+redirectAction), http.StatusFound)
	}
	return
}

type staffInfoJSON struct {
	Username string           `json:"username"`
	Rank     int              `json:"rank"`
	Actions  []Action         `json:"actions,omitempty"`
	Reports  []reportWithLink `json:"reports,omitempty"`
	Appeals  []gcsql.Appeal   `json:"appeals,omitempty"`
}

func staffInfoCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, logger zerolog.Logger) (output any, err error) {
	info := staffInfoJSON{
		Username: staff.Username,
		Rank:     staff.Rank,
	}
	if staff.Rank >= JanitorPerms && request.FormValue("noactions") != "1" {
		info.Actions = getAvailableActions(staff.Rank, false)
	}
	if staff.Rank >= ModPerms {
		var err error
		if info.Reports, err = getReportsWithLinks(); err != nil {
			logger.Err(err).Caller().Send()
			return nil, fmt.Errorf("unable to get open reports: %w", err)
		}

		if info.Appeals, err = gcsql.GetAppeals(gcsql.AppealsQueryOptions{
			Active:    gcsql.OnlyTrue,
			Unexpired: gcsql.OnlyTrue,
			Limit:     4,
		}); err != nil {
			logger.Err(err).Caller().Send()
			return nil, fmt.Errorf("unable to get the number of open appeals: %w", err)
		}
	}
	return info, nil
}

func registerNoPermPages() {
	RegisterManagePage("staffinfo", "", NoPerms, AlwaysJSON, staffInfoCallback)
	RegisterManagePage("login", loginTitle, NoPerms, NoJSON, loginCallback)
}
