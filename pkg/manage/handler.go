package manage

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
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
	serverutil.ServeError(writer, message, isJSON, map[string]interface{}{
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
	if err = request.ParseForm(); err != nil {
		gcutil.LogError(err).
			Str("IP", gcutil.GetRealIP(request)).
			Msg("Error parsing form data")
		serverutil.ServeError(writer, "Error parsing form data: "+err.Error(), wantsJSON, nil)
		return
	}
	actionID := request.FormValue("action")
	var staff *gcsql.Staff
	staff, err = getCurrentFullStaff(request)
	if err == http.ErrNoCookie {
		staff = &gcsql.Staff{}
		err = nil
	} else if err != nil && err != sql.ErrNoRows {
		gcutil.LogError(err).
			Str("request", "getCurrentFullStaff").
			Str("action", actionID).Send()
		serverutil.ServeError(writer, "Error getting staff info from request: "+err.Error(), wantsJSON, nil)
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

	var managePageBuffer bytes.Buffer
	action := getAction(actionID, staff.Rank)
	if action == nil {
		if wantsJSON {
			serveError(writer, "notfound", actionID, "action not found", wantsJSON || (action.JSONoutput == AlwaysJSON))
		} else {
			serverutil.ServeNotFound(writer, request)
		}
		return
	}

	if staff.Rank < action.Permissions {
		writer.WriteHeader(403)
		staffName, _ := getCurrentStaff(request)
		gcutil.LogInfo().
			Str("staff", "insufficientPerms").
			Str("IP", gcutil.GetRealIP(request)).
			Str("username", staffName).
			Str("action", actionID)
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
		output, err = action.Callback(writer, request, staff, wantsJSON)
	}
	if err != nil {
		// writer.WriteHeader(500)
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
		headerMap["include_dashboard_link"] = true
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
