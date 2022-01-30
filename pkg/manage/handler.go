package manage

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/gclog"
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
	if isJSON {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
		errJSON, _ := gcutil.MarshalJSON(ErrStaffAction{
			ErrorField: field,
			Action:     action,
			Message:    message,
		}, true)

		serverutil.MinifyWriter(writer, []byte(errJSON), "application/json")
		return
	}
	serverutil.ServeErrorPage(writer, message)
}

func isRequestingJSON(request *http.Request) bool {
	field := request.Form["json"]
	return len(field) == 1 && (field[0] == "1" || field[0] == "true")
}

// CallManageFunction is called when a user accesses /manage to use manage tools
// or log in to a staff account
func CallManageFunction(writer http.ResponseWriter, request *http.Request) {
	var err error
	if err = request.ParseForm(); err != nil {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error parsing form data: ", err.Error()))
		return
	}
	wantsJSON := isRequestingJSON(request)
	actionID := request.FormValue("action")
	staffRank := GetStaffRank(request)

	if actionID == "" {
		if staffRank == NoPerms {
			actionID = "login"
		} else {
			actionID = "dashboard"
		}
	}

	var managePageBuffer bytes.Buffer
	action := getAction(actionID, staffRank)
	if action == nil {
		if wantsJSON {
			serveError(writer, "notfound", actionID, "action not found", wantsJSON)
		} else {
			serverutil.ServeNotFound(writer, request)
		}
		return
	}

	if staffRank < action.Permissions {
		writer.WriteHeader(403)
		staffName, _ := getCurrentStaff(request)
		gclog.Printf(gclog.LStaffLog,
			"Rejected request to manage page %s from %s (insufficient permissions)",
			actionID, staffName)
		serveError(writer, "permission", actionID, "You do not have permission to access this page", wantsJSON || (action.JSONoutput > NoJSON))
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
		output, err = action.Callback(writer, request, wantsJSON)
	}
	if err != nil {
		staffName, _ := getCurrentStaff(request)
		// writer.WriteHeader(500)
		gclog.Printf(gclog.LStaffLog|gclog.LErrorLog,
			"Error accessing manage page %s by %s: %s", actionID, staffName, err.Error())
		serveError(writer, "actionerror", actionID, err.Error(), wantsJSON || (action.JSONoutput > NoJSON))
		return
	}
	if action.JSONoutput == AlwaysJSON || wantsJSON {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
		outputJSON, err := gcutil.MarshalJSON(output, true)
		if err != nil {
			serveError(writer, "error", actionID, err.Error(), true)
			return
		}
		serverutil.MinifyWriter(writer, []byte(outputJSON), "application/json")
		return
	}
	if err = building.BuildPageHeader(&managePageBuffer, action.Title); err != nil {
		serveError(writer, "error", actionID,
			gclog.Print(gclog.LErrorLog, "Failed writing page header: ", err.Error()), false)
		return
	}
	managePageBuffer.WriteString("<br />" + fmt.Sprint(output) + "<br /><br />")
	if err = building.BuildPageFooter(&managePageBuffer); err != nil {
		serveError(writer, "error", actionID,
			gclog.Print(gclog.LErrorLog, "Failed writing page footer: ", err.Error()), false)
		return
	}
	writer.Write(managePageBuffer.Bytes())
}
