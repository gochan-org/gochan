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
		return
	}

	action := request.FormValue("action")
	staffRank := GetStaffRank(request)
	var managePageBuffer bytes.Buffer

	if action == "" {
		if staffRank == NoPerms {
			action = "login"
		} else {
			action = "staffmenu"
		}
	}

	handler, ok := actions[action]
	var htmlOut string

	if !ok {
		serverutil.ServeNotFound(writer, request)
		return
	}
	if action == "staffmenu" {
		handler.Callback = getStaffMenu
	}
	if staffRank == NoPerms && handler.Permissions > NoPerms {
		handler = actions["login"]
	} else if staffRank < handler.Permissions {
		writer.WriteHeader(403)
		serverutil.ServeErrorPage(writer, "You don't have permission to access this page.")
		staffName, _ := getCurrentStaff(request)
		gclog.Printf(gclog.LStaffLog,
			"Rejected request to manage page %s from %s (insufficient permissions)", action, staffName)
		return
	}
	htmlOut, err = handler.Callback(writer, request)
	if err != nil {
		staffName, _ := getCurrentStaff(request)
		// writer.WriteHeader(500)
		serverutil.ServeErrorPage(writer, err.Error())
		gclog.Printf(gclog.LStaffLog|gclog.LErrorLog,
			"Error accessing manage page %s by %s: %s", action, staffName, err.Error())
		return
	}
	var footer string

	if !handler.isJSON {
		managePageBuffer.WriteString("<!DOCTYPE html><html><head>")
		criticalCfg := config.GetSystemCriticalConfig()
		if err = gctemplates.ManageHeader.Execute(&managePageBuffer, map[string]interface{}{
			"webroot": criticalCfg.WebRoot,
		}); err != nil {
			serverutil.ServeErrorPage(writer, gclog.Print(gclog.LErrorLog|gclog.LStaffLog,
				"Error executing manage page header template: ", err.Error()))
			return
		}
		footer = "</body></html>"
	} else {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	}
	managePageBuffer.WriteString(htmlOut + footer)
	writer.Write(managePageBuffer.Bytes())
}
