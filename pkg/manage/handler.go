package manage

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/uptrace/bunrouter"
)

type ErrStaffAction struct {
	// ErrorField can be used in the frontend for giving more specific info about the error
	ErrorField string `json:"error"`
	Action     string `json:"action"`
	Message    string `json:"message"`
}

func (esa *ErrStaffAction) Error() string {
	return esa.Message
}

func serveError(writer http.ResponseWriter, field string, action string, message string, isJSON bool) {
	server.ServeError(writer, message, isJSON, map[string]interface{}{
		"error":   field,
		"action":  action,
		"message": message,
	})
}

// CallManageFunction is called when a user accesses /manage to use manage tools
// or log in to a staff account
func CallManageFunction(writer http.ResponseWriter, request *http.Request) {
	var err error
	wantsJSON := serverutil.IsRequestingJSON(request)
	errEv := gcutil.LogError(nil)
	accessEv := gcutil.LogAccess(request)
	infoEv := gcutil.LogInfo()
	defer gcutil.LogDiscard(infoEv, accessEv, errEv)

	errEv.Str("IP", gcutil.GetRealIP(request))
	if err = request.ParseForm(); err != nil {
		errEv.Err(err).Caller().Msg("Error parsing form data")
		server.ServeError(writer, "Error parsing form data: "+err.Error(), wantsJSON, nil)
		return
	}
	params := bunrouter.ParamsFromContext(request.Context())
	actionID := params.ByName("action")
	gcutil.LogStr("action", actionID, infoEv, accessEv, errEv)

	var staff *gcsql.Staff
	staff, err = getCurrentFullStaff(request)
	if err == http.ErrNoCookie {
		staff = &gcsql.Staff{}
		err = nil
	} else if err != nil && err != sql.ErrNoRows {
		errEv.Err(err).
			Str("request", "getCurrentFullStaff").
			Caller().Send()
		server.ServeError(writer, "Error getting staff info from request: "+err.Error(), wantsJSON, nil)
		return
	}
	if actionID == "" {
		if staff.Rank == NoPerms {
			// no action requested and user is not logged in, have them go to login page
			actionID = "login"
		} else {
			actionID = "dashboard"
		}
	}
	gcutil.LogStr("staff", staff.Username, infoEv, accessEv, errEv)
	var managePageBuffer bytes.Buffer
	action := getAction(actionID, staff.Rank)
	if action == nil {
		if wantsJSON {
			serveError(writer, "notfound", actionID, "action not found", wantsJSON || (action.JSONoutput == AlwaysJSON))
		} else {
			server.ServeNotFound(writer, request)
		}
		return
	}

	if staff.Rank < action.Permissions {
		writer.WriteHeader(http.StatusForbidden)
		errEv.
			Int("rank", staff.Rank).
			Int("requiredRank", action.Permissions).
			Msg("Insufficient permissions")
		serveError(writer, "permission", actionID, "You do not have permission to access this page", wantsJSON || (action.JSONoutput == AlwaysJSON))
		return
	}

	var output interface{}
	if wantsJSON && action.JSONoutput == NoJSON {
		output = nil
		err = &ErrStaffAction{
			ErrorField: "nojson",
			Action:     actionID,
			Message:    "Requested mod page does not have a JSON output option",
		}
	} else {
		output, err = action.Callback(writer, request, staff, wantsJSON, infoEv, errEv)
	}
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		serveError(writer, "actionerror", actionID, err.Error(), wantsJSON || (action.JSONoutput == AlwaysJSON))
		return
	}
	if action.JSONoutput == AlwaysJSON || (action.JSONoutput > NoJSON && wantsJSON) {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
		outputJSON, err := gcutil.MarshalJSON(output, true)
		if err != nil {
			serveError(writer, "error", actionID, err.Error(), true)
			gcutil.LogError(err).
				Str("action", actionID).Send()
			return
		}
		serverutil.MinifyWriter(writer, []byte(outputJSON), "application/json")
		return
	}

	headerMap := map[string]interface{}{
		"page_type": "manage",
	}
	if action.ID != "dashboard" && action.ID != "login" && action.ID != "logout" {
		headerMap["includeDashboardLink"] = true
	}
	if err = building.BuildPageHeader(&managePageBuffer, action.Title, "", headerMap); err != nil {
		gcutil.LogError(err).
			Str("action", actionID).
			Str("staff", "pageHeader").Send()
		serveError(writer, "error", actionID, "Failed writing page header: "+err.Error(), false)
		return
	}
	managePageBuffer.WriteString("<br />" + fmt.Sprint(output) + "<br /><br />")
	if err = building.BuildPageFooter(&managePageBuffer); err != nil {
		gcutil.LogError(err).
			Str("action", actionID).
			Str("staff", "pageFooter").Send()
		serveError(writer, "error", actionID, "Failed writing page footer: "+err.Error(), false)
		return
	}
	writer.Write(managePageBuffer.Bytes())
}
