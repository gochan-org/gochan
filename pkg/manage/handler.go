package manage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

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
	server.ServeError(writer, message, isJSON, map[string]any{
		"error":   field,
		"action":  action,
		"message": message,
	})
}

// setupManageFunction returns a function used by the HTTP router to serve the action callback's return value
func setupManageFunction(action *Action) bunrouter.HandlerFunc {
	return func(writer http.ResponseWriter, req bunrouter.Request) (err error) {
		request := req.Request
		wantsJSON := serverutil.IsRequestingJSON(request)
		accessEv := gcutil.LogAccess(request)
		infoEv, errEv := gcutil.LogRequest(request)
		defer gcutil.LogDiscard(infoEv, accessEv, errEv)

		gcutil.LogStr("action", action.ID, infoEv, accessEv, errEv)
		if err = req.Request.ParseForm(); err != nil {
			errEv.Err(err).Caller().Msg("Error parsing form data")
			server.ServeError(writer, "Error parsing form data: "+err.Error(), wantsJSON, map[string]any{
				"action": action.ID,
			})
			return
		}

		var staff *gcsql.Staff
		staff, err = GetStaffFromRequest(request)
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get staff from request")
			server.ServeError(writer, "Error getting staff info from request", wantsJSON, nil)
			return
		}
		gcutil.LogStr("staff", staff.Username, infoEv, accessEv, errEv)

		actionCB := action.Callback
		if staff.Username == "" && action.Permissions > NoPerms {
			// action with permissions requested and user is not logged in, have them go to login page
			actionCB = loginCallback
		}
		if staff.Rank < action.Permissions {
			writer.WriteHeader(http.StatusForbidden)
			gcutil.LogWarning().
				Str("ip", gcutil.GetRealIP(request)).
				Str("userAgent", request.UserAgent()).
				Int("rank", staff.Rank).
				Str("action", action.ID).
				Int("requiredRank", action.Permissions).
				Msg("Staff requested page with insufficient permissions")
			serveError(writer, "permission", action.ID, "You do not have permission to access this page", wantsJSON || (action.JSONoutput == AlwaysJSON))
			return
		}

		var output any
		if wantsJSON && action.JSONoutput == NoJSON {
			output = nil
			err = &ErrStaffAction{
				ErrorField: "nojson",
				Action:     action.ID,
				Message:    "Requested mod page does not have a JSON output option",
			}
		} else {
			defer func() {
				if a := recover(); a != nil {
					serveError(writer, "actionerror", action.ID, "Internal server error", wantsJSON)
					gcutil.LogError(nil).
						Str("ip", gcutil.GetRealIP(request)).
						Str("userAgent", req.UserAgent()).
						Interface("recover", a).
						Bytes("stack", debug.Stack()).
						Msg("Recovered from panic while calling manage function")
				}
			}()
			if actionCB == nil {
				output = ""
				err = fmt.Errorf("action %q exists but has no defined callback", action.ID)
			} else {
				output, err = actionCB(writer,
					request.WithContext(context.WithValue(request.Context(), "actionParams", req.Params())),
					staff, wantsJSON, infoEv, errEv)
			}
		}
		if err != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			serveError(writer, "actionerror", action.ID, err.Error(), wantsJSON || (action.JSONoutput == AlwaysJSON))
			return
		}
		if action.JSONoutput == AlwaysJSON || (action.JSONoutput > NoJSON && wantsJSON) {
			writer.Header().Add("Content-Type", "application/json")
			writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
			var outputJSON string
			if outputJSON, err = gcutil.MarshalJSON(output, true); err != nil {
				serveError(writer, "error", action.ID, err.Error(), true)
				errEv.Err(err).Caller().Send()
				return
			}
			serverutil.MinifyWriter(writer, []byte(outputJSON), "application/json")
			return
		}

		headerMap := map[string]any{
			"page_type": "manage",
		}
		if action.ID != "dashboard" && action.ID != "login" && action.ID != "logout" {
			headerMap["includeDashboardLink"] = true
		}

		var buf bytes.Buffer
		if err = building.BuildPageHeader(&buf, action.Title, "", headerMap); err != nil {
			gcutil.LogError(err).
				Str("action", action.ID).
				Str("staff", "pageHeader").Send()
			serveError(writer, "error", action.ID, "Failed writing page header: "+err.Error(), false)
			return
		}
		buf.WriteString("<br />" + fmt.Sprint(output) + "<br /><br />")
		if err = building.BuildPageFooter(&buf); err != nil {
			gcutil.LogError(err).
				Str("action", action.ID).
				Str("staff", "pageFooter").Send()
			serveError(writer, "error", action.ID, "Failed writing page footer: "+err.Error(), false)
			return
		}
		writer.Write(buf.Bytes())

		return nil
	}
}
