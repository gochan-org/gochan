package serverutil

import (
	"net/http"
	"time"
)

// DeleteCookie deletes the given cookie if it exists. It returns true if it exists and false
// with no errors if it doesn't
func DeleteCookie(writer http.ResponseWriter, request *http.Request, cookieName string) bool {
	cookie, err := request.Cookie(cookieName)
	if err != nil {
		return false
	}
	cookie.MaxAge = 0
	cookie.Expires = time.Now().Add(-7 * 24 * time.Hour)
	http.SetCookie(writer, cookie)
	return true
}

func IsRequestingJSON(request *http.Request) bool {
	jsonField := request.FormValue("json")
	if jsonField == "" {
		jsonField = request.PostFormValue("json")
	}
	return jsonField == "1" || jsonField == "true"
}
