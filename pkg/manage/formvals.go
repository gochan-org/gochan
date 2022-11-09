package manage

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

func getStringField(field string, staff string, request *http.Request, logCallerOffset ...int) (string, error) {
	action := request.FormValue("action")
	callerOffset := 1
	if len(logCallerOffset) > 0 {
		callerOffset += logCallerOffset[0]
	}
	if len(request.Form[field]) == 0 {
		gcutil.LogError(nil).
			Str("IP", gcutil.GetRealIP(request)).
			Str("staff", staff).
			Str("action", action).
			Str("field", field).
			Caller(callerOffset).Msg("Missing required field")
		return "", &ErrStaffAction{
			ErrorField: field,
			Action:     action,
			Message:    fmt.Sprintf("Missing required field %q", field),
		}
	}
	return request.FormValue(field), nil
}

func getBooleanField(field string, staff string, request *http.Request, logCallerOffset ...int) (bool, error) {
	action := request.FormValue("action")
	callerOffset := 1
	if len(logCallerOffset) > 0 {
		callerOffset += logCallerOffset[0]
	}
	if len(request.Form[field]) == 0 {
		gcutil.LogError(nil).
			Str("IP", gcutil.GetRealIP(request)).
			Str("staff", staff).
			Str("action", action).
			Str("field", field).
			Caller(callerOffset).Msg("Missing required field")
		return false, &ErrStaffAction{
			ErrorField: field,
			Action:     action,
			Message:    fmt.Sprintf("Missing required field %q", field),
		}
	}
	return request.FormValue(field) == "on", nil
}

// getIntField gets the requested value from the form and tries to convert it to int. If it fails, it logs the error
// and wraps it in ErrStaffAction
func getIntField(field string, staff string, request *http.Request, logCallerOffset ...int) (int, error) {
	action := request.FormValue("action")
	callerOffset := 1
	if len(logCallerOffset) > 0 {
		callerOffset += logCallerOffset[0]
	}

	if len(request.Form[field]) == 0 {
		gcutil.LogError(nil).
			Str("IP", gcutil.GetRealIP(request)).
			Str("staff", staff).
			Str("action", action).
			Str("field", field).
			Caller(callerOffset).Msg("Missing required field")
		return 0, &ErrStaffAction{
			ErrorField: field,
			Action:     action,
			Message:    fmt.Sprintf("Missing required field %q", field),
		}
	}
	strVal := request.FormValue(field)

	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		gcutil.LogError(err).
			Str("IP", gcutil.GetRealIP(request)).
			Str("staff", staff).
			Str("action", action).
			Str("field", field).
			Caller(callerOffset).Msg("Unable to convert field to int")
		return 0, &ErrStaffAction{
			ErrorField: field,
			Action:     action,
			Message:    fmt.Sprintf("Unable to convert form field %q to int"),
		}
	}
	return intVal, nil
}
