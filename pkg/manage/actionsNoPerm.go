package manage

import (
	"bytes"
	"errors"
	"net/http"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

var (
	loginAction = Action{
		ID:          "login",
		Title:       "Login",
		Permissions: NoPerms,
		Callback:    loginCallback,
	}
)

func loginCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, _ bool, _, errEv *zerolog.Event) (output interface{}, err error) {
	systemCritical := config.GetSystemCriticalConfig()
	if staff.Rank > 0 {
		http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage"), http.StatusFound)
	}
	username := request.FormValue("username")
	password := request.FormValue("password")
	redirectAction := request.FormValue("action")
	if redirectAction == "" || redirectAction == "logout" {
		redirectAction = "dashboard"
	}

	if username == "" || password == "" {
		//assume that they haven't logged in
		manageLoginBuffer := bytes.NewBufferString("")
		if err = serverutil.MinifyTemplate(gctemplates.ManageLogin, map[string]interface{}{
			"siteConfig":  config.GetSiteConfig(),
			"sections":    gcsql.AllSections,
			"boards":      gcsql.AllBoards,
			"boardConfig": config.GetBoardConfig(""),
			"redirect":    redirectAction,
		}, manageLoginBuffer, "text/html"); err != nil {
			errEv.Err(err).Str("template", "manage_login.html").Send()
			return "", errors.New("Error executing staff login page template: " + err.Error())
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
		http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage/"+request.FormValue("redirect")), http.StatusFound)
	}
	return
}

type fingerprintingOptions struct {
	FingerprintVideoThumbs bool     `json:"fingerprintVideoThumbs"`
	ImageExtensions        []string `json:"imageExtensions,omitempty"`
	VideoExtensions        []string `json:"videoExtensions,omitempty"`
}

type staffInfoJSON struct {
	Username       string                 `json:"username"`
	Rank           int                    `json:"rank"`
	Actions        []Action               `json:"actions,omitempty"`
	Fingerprinting *fingerprintingOptions `json:"fingerprinting,omitempty"`
}

func staffInfoCallback(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
	info := staffInfoJSON{
		Username: staff.Username,
		Rank:     staff.Rank,
	}
	if staff.Rank >= JanitorPerms {
		info.Actions = getAvailableActions(staff.Rank, false)
	}
	if staff.Rank >= ModPerms {
		info.Fingerprinting = &fingerprintingOptions{
			FingerprintVideoThumbs: config.GetSiteConfig().FingerprintVideoThumbnails,
			ImageExtensions:        uploads.ImageExtensions,
			VideoExtensions:        uploads.VideoExtensions,
		}
	}
	return info, nil
}

func registerNoPermPages() {
	actions = append(actions, loginAction,
		Action{
			ID:          "staffinfo",
			Permissions: NoPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    staffInfoCallback,
		},
	)
}
