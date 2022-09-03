package manage

import (
	"bytes"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
	"golang.org/x/crypto/bcrypt"
)

const (
	sSuccess = iota
	sInvalidPassword
	sOtherError
)

func createSession(key, username, password string, request *http.Request, writer http.ResponseWriter) int {
	//returns 0 for successful, 1 for password mismatch, and 2 for other
	domain := request.Host
	var err error
	domain = chopPortNumRegex.Split(domain, -1)[0]

	if !serverutil.ValidReferer(request) {
		gclog.Print(gclog.LStaffLog, "Rejected login from possible spambot @ "+request.RemoteAddr)
		return sOtherError
	}
	staff, err := gcsql.GetStaffByName(username)
	if err != nil {
		gclog.Print(gclog.LErrorLog, err.Error())
		return sInvalidPassword
	}

	success := bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
	if success == bcrypt.ErrMismatchedHashAndPassword {
		// password mismatch
		gclog.Print(gclog.LStaffLog, "Failed login (password mismatch) from "+request.RemoteAddr+" at "+time.Now().Format(gcsql.MySQLDatetimeFormat))
		return sInvalidPassword
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

	if err = gcsql.CreateSession(key, username); err != nil {
		gclog.Print(gclog.LErrorLog, "Error creating new staff session: ", err.Error())
		return sOtherError
	}

	return sSuccess
}

func getCurrentStaff(request *http.Request) (string, error) { //TODO after refactor, check if still used
	sessionCookie, err := request.Cookie("sessiondata")
	if err != nil {
		return "", err
	}
	return gcsql.GetStaffName(sessionCookie.Value)
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

// returns the action by its ID, or nil if it doesn't exist
func getAction(id string, rank int) *Action {
	for a := range actions {
		if rank == NoPerms && actions[a].Permissions > NoPerms {
			id = "login"
		}
		if actions[a].ID == id {
			return &actions[a]
		}
	}
	return nil
}

func init() {
	actions = append(actions,
		Action{
			ID:          "actions",
			Title:       "Staff actions",
			Permissions: JanitorPerms,
			JSONoutput:  AlwaysJSON,
			Callback:    getStaffActions,
		},
		Action{
			ID:          "dashboard",
			Title:       "Dashboard",
			Permissions: JanitorPerms,
			Callback:    dashboardCallback,
		})
}

func dashboardCallback(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (interface{}, error) {
	dashBuffer := bytes.NewBufferString("")
	announcements, err := gcsql.GetAllAccouncements()
	if err != nil {
		return nil, err
	}
	rank := GetStaffRank(request)
	rankString := ""
	switch rank {
	case AdminPerms:
		rankString = "administrator"
	case ModPerms:
		rankString = "moderator"
	case JanitorPerms:
		rankString = "janitor"
	}

	availableActions := getAvailableActions(rank, true)
	if err = serverutil.MinifyTemplate(gctemplates.ManageDashboard,
		map[string]interface{}{
			"actions":       availableActions,
			"rank":          rank,
			"rankString":    rankString,
			"announcements": announcements,
			"boards":        gcsql.AllBoards,
			"webroot":       config.GetSystemCriticalConfig().WebRoot,
		}, dashBuffer, "text/html"); err != nil {
		gclog.Printf(gclog.LErrorLog|gclog.LStaffLog,
			"Error executing dashboard template: %q", err.Error())
		return "", err
	}
	return dashBuffer.String(), nil
}

func getAvailableActions(rank int, noJSON bool) []Action {
	available := []Action{}
	for _, action := range actions {
		if (rank < action.Permissions || action.Permissions == NoPerms) ||
			(noJSON && action.JSONoutput == AlwaysJSON) {
			continue
		}
		available = append(available, action)
	}
	return available
}

func getStaffActions(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (interface{}, error) {
	rank := GetStaffRank(request)
	availableActions := getAvailableActions(rank, false)
	return availableActions, nil
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

func invalidWordfilterID(id interface{}) error {
	return fmt.Errorf("wordfilter with id %q does not exist", id)
}
