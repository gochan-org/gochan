package manage

import (
	"net/http"
	"path"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

const (
	// NoPerms allows anyone to access this Action
	NoPerms = iota
	// JanitorPerms allows anyone with at least a janitor-level account to access this Action
	JanitorPerms
	// ModPerms allows anyone with at least a moderator-level account to access this Action
	ModPerms
	// AdminPerms allows only the site administrator to view this Action
	AdminPerms
)

const (
	// NoJSON actions will return an error if JSON is requested by the user
	NoJSON = iota
	// OptionalJSON actions have an optional JSON output if requested
	OptionalJSON
	// AlwaysJSON actions always return JSON whether or not it is requested
	AlwaysJSON
)

type CallbackFunction func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error)

// Action represents the functions accessed by staff members at /manage/<functionname>.
type Action struct {
	// ID is the string used when the user requests /manage/<ID>
	ID string `json:"id"`

	// Title is used for the text shown in the staff menu and the window title
	Title string `json:"title"`

	// Permissions represent who can access the page. 0 for anyone,
	// 1 requires the user to have a janitor, mod, or admin account. 2 requires mod or admin,
	// and 3 is only accessible by admins
	Permissions int `json:"perms"`

	// Hidden is used to hide the action from the staff menu
	Hidden bool `json:"-"`

	// JSONoutput sets what the action can output. If it is 0, it will throw an error if
	// JSON is requested. If it is 1, it can output JSON if requested, and if 2, it always
	// outputs JSON whether it is requested or not
	JSONoutput int `json:"jsonOutput"` // if it can sometimes return JSON, this should still be false

	// Callback executes the staff page. if wantsJSON is true, it should return an object
	// to be marshalled into JSON. Otherwise, a string assumed to be valid HTML is returned.
	//
	// IMPORTANT: the writer parameter should only be written to if absolutely necessary (for example,
	// if a redirect wouldn't work in handler.go) and even then, it should be done sparingly
	Callback CallbackFunction `json:"-"`
}

var actions []Action

// RegisterManagePage registers a new manage page at /manage/<id> with the given title, accessible by staff
// who have at least the given permissions level. The page can return JSON if jsonOutput is OptionalJSON or AlwaysJSON.
// The callback function is called when the page is accessed via both GET and POST.
func RegisterManagePage(id string, title string, permissions int, jsonOutput int, callback CallbackFunction) {
	RegisterManagePageWithMethods(id, title, permissions, jsonOutput, false, callback)
}

// RegisterStaffAction registers a new staff action and its associated HTTP handler(s). The callback is called when
// the page is accessed at /manage/<action.ID>. If no methods are specified, it defaults to GET and POST.
func RegisterStaffAction(action Action, methods ...string) {
	actions = append(actions, action)
	handlerFunc := setupManageFunction(&action)
	server.GetRouter().WithGroup(config.WebPath("/manage"), func(g *bunrouter.Group) {
		groupPath := bunrouter.CleanPath(path.Join("/", action.ID))
		if len(methods) == 0 {
			methods = []string{http.MethodGet, http.MethodPost}
		}
		for _, method := range methods {
			switch method {
			case http.MethodGet:
				g.GET(groupPath, handlerFunc)
			case http.MethodPost:
				g.POST(groupPath, handlerFunc)
			case http.MethodDelete:
				g.DELETE(groupPath, handlerFunc)
			case http.MethodPut:
				g.PUT(groupPath, handlerFunc)
			}
		}
	})
}

// RegisterManagePageWithMethods is like RegisterManagePage but allows specifying which HTTP methods to register the page with, and whether
// the page should be hidden from the staff menu (for example, if it is a sub-page of another page). If no methods are specified, it defaults to GET and POST.
func RegisterManagePageWithMethods(id string, title string, permissions int, jsonOutput int, hidden bool, callback CallbackFunction, methods ...string) {
	action := Action{
		ID:          id,
		Title:       title,
		Permissions: permissions,
		Hidden:      hidden,
		JSONoutput:  jsonOutput,
		Callback:    callback,
	}
	RegisterStaffAction(action, methods...)
}

func getAvailableActions(rank int, noJSON bool) []Action {
	var available []Action

	for _, action := range actions {
		if rank >= action.Permissions && action.Permissions != NoPerms && (action.JSONoutput != AlwaysJSON || !noJSON) && !action.Hidden {
			available = append(available, action)
		}
	}
	return available
}

func getPageTitle(actionID string, staff *gcsql.Staff) string {
	notLoggedIn := staff == nil || staff.Rank == 0
	var useAction Action
	for _, action := range actions {
		if action.ID == actionID {
			useAction = action
			break
		}
	}

	if notLoggedIn && useAction.Permissions > NoPerms {
		return loginTitle
	}
	return useAction.Title
}

func getStaffActions(_ http.ResponseWriter, _ *http.Request, staff *gcsql.Staff, _ bool, _ *zerolog.Event, _ *zerolog.Event) (any, error) {
	availableActions := getAvailableActions(staff.Rank, false)
	return availableActions, nil
}
