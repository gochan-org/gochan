package manage

import (
	"bytes"
	"net/http"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

// CallManageFunction is called when a user accesses /manage to use manage tools
// or log in to a staff account
func CallManageFunction(writer http.ResponseWriter, request *http.Request) {
	var err error
	if err = request.ParseForm(); err != nil {
		serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog,
			"Error parsing form data: ", err.Error()))
	}

	action := request.FormValue("action")
	staffRank := GetStaffRank(request)
	var managePageBuffer bytes.Buffer
	if action == "postinfo" {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	}

	if action != "getstaffjquery" && action != "postinfo" {
		managePageBuffer.WriteString("<!DOCTYPE html><html><head>")
		if err = gctemplates.ManageHeader.Execute(&managePageBuffer, config.Config); err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog|gclog.LStaffLog,
				"Error executing manage page header template: ", err.Error()))
			return
		}
	}

	if action == "" {
		managePageBuffer.Write([]byte(actionHTMLLinker(manageFunctions)))
	} else {
		if _, ok := manageFunctions[action]; ok {
			if staffRank >= manageFunctions[action].Permissions {
				managePageBuffer.Write([]byte(manageFunctions[action].Callback(writer, request)))
			} else if staffRank == 0 && manageFunctions[action].Permissions == 0 {
				managePageBuffer.Write([]byte(manageFunctions[action].Callback(writer, request)))
			} else if staffRank == 0 {
				managePageBuffer.Write([]byte(manageFunctions["login"].Callback(writer, request)))
			} else {
				managePageBuffer.Write([]byte(action + " is undefined."))
			}
		} else {
			managePageBuffer.Write([]byte(action + " is undefined."))
		}
	}
	if action != "getstaffjquery" && action != "postinfo" {
		managePageBuffer.Write([]byte("</body></html>"))
	}

	writer.Write(managePageBuffer.Bytes())
}
