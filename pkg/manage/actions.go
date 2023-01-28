package manage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
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

type CallbackFunction func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error)

// Action represents the functions accessed by staff members at /manage/<functionname>.
type Action struct {
	// the string used when the user requests /manage/<ID>
	ID string `json:"id"`

	// The text shown in the staff menu and the window title
	Title string `json:"title"`

	// Permissions represent who can access the page. 0 for anyone,
	// 1 requires the user to have a janitor, mod, or admin account. 2 requires mod or admin,
	// and 3 is only accessible by admins
	Permissions int `json:"perms"`

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

var actions = []Action{
	{
		ID:          "logout",
		Title:       "Logout",
		Permissions: JanitorPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			if err = gcsql.EndStaffSession(writer, request); err != nil {
				return "", err
			}
			http.Redirect(writer, request,
				config.GetSystemCriticalConfig().WebRoot+"manage",
				http.StatusSeeOther)
			return "Logged out successfully", nil
		}},
	{
		ID:          "clearmysessions",
		Title:       "Log me out everywhere",
		Permissions: JanitorPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			session, err := request.Cookie("sessiondata")
			if err != nil {
				// doesn't have a login session cookie, return with no errors
				if !wantsJSON {
					http.Redirect(writer, request,
						config.GetSystemCriticalConfig().WebRoot+"manage",
						http.StatusSeeOther)
					return
				}
				return "You are not logged in", nil
			}

			_, err = gcsql.GetStaffBySession(session.Value)
			if err != nil {
				// staff session doesn't exist, probably a stale cookie
				if !wantsJSON {
					http.Redirect(writer, request,
						config.GetSystemCriticalConfig().WebRoot+"manage",
						http.StatusSeeOther)
					return
				}
				return "You are not logged in", err
			}
			if err = staff.ClearSessions(); err != nil && err != sql.ErrNoRows {
				// something went wrong when trying to clean out sessions for this user
				return nil, err
			}
			serverutil.DeleteCookie(writer, request, "sessiondata")
			gcutil.LogAccess(request).
				Str("clearSessions", staff.Username).
				Send()
			if !wantsJSON {
				http.Redirect(writer, request,
					config.GetSystemCriticalConfig().WebRoot+"manage",
					http.StatusSeeOther)
				return "", nil
			}
			return "Logged out successfully", nil
		},
	},
	{
		ID:          "cleanup",
		Title:       "Cleanup",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			outputStr := ""
			if request.FormValue("run") == "Run Cleanup" {
				outputStr += "Removing deleted posts from the database.<hr />"
				if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
					errEv.Err(err).
						Str("cleanup", "removeDeletedPosts").
						Caller().Send()
					err = errors.New("Error removing deleted posts from database: " + err.Error())
					return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
				}

				outputStr += "Optimizing all tables in database.<hr />"
				err = gcsql.OptimizeDatabase()
				if err != nil {
					errEv.Err(err).
						Str("sql", "optimization").
						Caller().Send()
					err = errors.New("Error optimizing SQL tables: " + err.Error())
					return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
				}
				outputStr += "Cleanup finished"
			} else {
				outputStr += `<form action="` + config.GetSystemCriticalConfig().WebRoot + `manage/cleanup" method="post">` +
					`<input name="run" id="run" type="submit" value="Run Cleanup" />` +
					`</form>`
			}
			return outputStr, nil
		}},
	{
		ID:          "recentposts",
		Title:       "Recent posts",
		Permissions: JanitorPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			limit := 20
			limitStr := request.FormValue("limit")
			if limitStr != "" {
				limit, err = strconv.Atoi(limitStr)
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
			}
			boardidStr := request.FormValue("boardid")
			var recentposts []building.Post
			var boardid int
			if boardidStr != "" {
				if boardid, err = strconv.Atoi(boardidStr); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
			}
			recentposts, err = building.GetRecentPosts(boardid, limit)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			if wantsJSON {
				return recentposts, nil
			}
			manageRecentsBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageRecentPosts, map[string]interface{}{
				"recentposts": recentposts,
				"allBoards":   gcsql.AllBoards,
				"boardid":     boardid,
				"limit":       limit,
			}, manageRecentsBuffer, "text/html"); err != nil {
				errEv.Err(err).Caller().Send()
				return "", errors.New("Error executing ban management page template: " + err.Error())
			}
			return manageRecentsBuffer.String(), nil
		},
	},
	{
		ID:          "bans",
		Title:       "Bans",
		Permissions: ModPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			var outputStr string
			var ban gcsql.IPBan
			ban.StaffID = staff.ID
			deleteIDStr := request.FormValue("delete")
			if deleteIDStr != "" {
				// deleting a ban
				ban.ID, err = strconv.Atoi(deleteIDStr)
				if err != nil {
					errEv.Err(err).
						Str("deleteBan", deleteIDStr).
						Caller().Send()
					return "", err
				}
				if err = ban.Deactivate(staff.ID); err != nil {
					errEv.Err(err).
						Int("deleteBan", ban.ID).
						Caller().Send()
					return "", err
				}

			} else if request.FormValue("do") == "add" {
				err := ipBanFromRequest(&ban, request, errEv)
				if err != nil {
					return "", err
				}
				infoEv.
					Str("bannedIP", ban.IP).
					Str("expires", ban.ExpiresAt.String()).
					Bool("permanent", ban.Permanent).
					Str("reason", ban.Message).
					Msg("Added IP ban")
			}

			filterBoardIDstr := request.FormValue("filterboardid")
			var filterBoardID int
			if filterBoardIDstr != "" {
				if filterBoardID, err = strconv.Atoi(filterBoardIDstr); err != nil {
					errEv.Err(err).
						Str("filterboardid", filterBoardIDstr).Caller().Send()
					return "", err
				}
			}
			limitStr := request.FormValue("limit")
			limit := 200
			if limitStr != "" {
				if limit, err = strconv.Atoi(limitStr); err != nil {
					errEv.Err(err).
						Str("limit", limitStr).Caller().Send()
					return "", err
				}
			}
			banlist, err := gcsql.GetIPBans(filterBoardID, limit, true)
			if err != nil {
				errEv.Err(err).Msg("Error getting ban list")
				err = errors.New("Error getting ban list: " + err.Error())
				return "", err
			}
			manageBansBuffer := bytes.NewBufferString("")

			if err = serverutil.MinifyTemplate(gctemplates.ManageBans, map[string]interface{}{
				"banlist":       banlist,
				"allBoards":     gcsql.AllBoards,
				"ban":           ban,
				"filterboardid": filterBoardID,
			}, manageBansBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_bans.html").Caller().Send()
				return "", errors.New("Error executing ban management page template: " + err.Error())
			}
			outputStr += manageBansBuffer.String()
			return outputStr, nil
		}},
	{
		ID:          "appeals",
		Title:       "Ban appeals",
		Permissions: ModPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			banIDstr := request.FormValue("banid")
			var banID int
			if banIDstr != "" {
				if banID, err = strconv.Atoi(banIDstr); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
			}
			infoEv.Int("banID", banID)

			limitStr := request.FormValue("limit")
			limit := 20
			if limitStr != "" {
				if limit, err = strconv.Atoi(limitStr); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
			}
			approveStr := request.FormValue("approve")
			if approveStr != "" {
				// approving an appeal
				approveID, err := strconv.Atoi(approveStr)
				if err != nil {
					errEv.Err(err).
						Str("approveStr", approveStr).Caller().Send()
				}
				if err = gcsql.ApproveAppeal(approveID, staff.ID); err != nil {
					errEv.Err(err).
						Int("approveAppeal", approveID).
						Caller().Send()
					return "", err
				}
			}

			appeals, err := gcsql.GetAppeals(banID, limit)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", errors.New("Unable to get appeals: " + err.Error())
			}

			manageAppealsBuffer := bytes.NewBufferString("")
			pageData := map[string]interface{}{}
			if len(appeals) > 0 {
				pageData["appeals"] = appeals
			}
			if err = serverutil.MinifyTemplate(gctemplates.ManageAppeals, pageData, manageAppealsBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_appeals.html").Caller().Send()
				return "", errors.New("Error executing appeal management page template: " + err.Error())
			}
			return manageAppealsBuffer.String(), err
		}},
	{
		ID:          "filebans",
		Title:       "Filename and checksum bans",
		Permissions: ModPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			delFilenameBanIDStr := request.FormValue("delfnb") // filename ban deletion
			delChecksumBanIDStr := request.FormValue("delcsb") // checksum ban deletion

			boardidStr := request.FormValue("boardid")
			boardid := 0
			if boardidStr != "" {
				boardid, err = strconv.Atoi(boardidStr)
				if err != nil {
					errEv.Err(err).
						Str("boardid", boardidStr).
						Caller().Send()
					return "", err
				}
			}
			staffnote := request.FormValue("staffnote")

			if request.FormValue("dofilenameban") != "" {
				// creating a new filename ban
				filename := request.FormValue("filename")
				isRegex := request.FormValue("isregex") == "on"
				if isRegex {
					_, err = regexp.Compile(filename)
					if err != nil {
						// invalid regular expression
						errEv.Err(err).
							Str("regex", filename).
							Caller().Send()
						return "", err
					}
				}
				if _, err = gcsql.NewFilenameBan(filename, isRegex, boardid, staff.ID, staffnote); err != nil {
					errEv.Err(err).
						Str("filename", filename).
						Bool("isregex", isRegex).
						Caller().Send()
					return "", err
				}
				infoEv.
					Str("filename", filename).
					Bool("isregex", isRegex).
					Int("boardid", boardid).
					Msg("Created new filename ban")
			} else if delFilenameBanIDStr != "" {
				delFilenameBanID, err := strconv.Atoi(delFilenameBanIDStr)
				if err != nil {
					errEv.Err(err).
						Str("delfnb", delFilenameBanIDStr).
						Caller().Send()
					return "", err
				}
				var fnb gcsql.FilenameBan
				fnb.ID = delFilenameBanID
				if err = fnb.Deactivate(staff.ID); err != nil {
					errEv.Err(err).
						Int("deleteFilenameBanID", delFilenameBanID).
						Caller().Send()
					return "", err
				}
				infoEv.
					Int("deleteFilenameBanID", delFilenameBanID).
					Int("boardid", boardid).
					Msg("Filename ban deleted")
			} else if request.FormValue("dochecksumban") != "" {
				// creating a new file checksum ban
				checksum := request.FormValue("checksum")
				if _, err = gcsql.NewFileChecksumBan(checksum, boardid, staff.ID, staffnote); err != nil {
					errEv.Err(err).
						Str("checksum", checksum).
						Caller().Send()
					return "", err
				}
				infoEv.
					Str("checksum", checksum).
					Int("boardid", boardid).
					Msg("Created new file checksum ban")
			} else if delChecksumBanIDStr != "" {
				// user requested a checksum ban ID to delete
				delChecksumBanID, err := strconv.Atoi(delChecksumBanIDStr)
				if err != nil {
					errEv.Err(err).
						Str("deleteChecksumBanIDStr", delChecksumBanIDStr).
						Caller().Send()
					return "", err
				}
				if err = (gcsql.FileBan{ID: delChecksumBanID}).Deactivate(staff.ID); err != nil {
					errEv.Err(err).
						Int("deleteChecksumBanID", delChecksumBanID).
						Caller().Send()
					return "", err
				}
				infoEv.Int("deleteChecksumBanID", delChecksumBanID).Msg("File checksum ban deleted")
			}
			filterBoardIDstr := request.FormValue("filterboardid")
			var filterBoardID int
			if filterBoardIDstr != "" {
				if filterBoardID, err = strconv.Atoi(filterBoardIDstr); err != nil {
					errEv.Err(err).
						Str("filterboardid", filterBoardIDstr).Caller().Send()
					return "", err
				}
			}
			limitStr := request.FormValue("limit")
			limit := 200
			if limitStr != "" {
				if limit, err = strconv.Atoi(limitStr); err != nil {
					errEv.Err(err).
						Str("limit", limitStr).Caller().Send()
					return "", err
				}
			}
			checksumBans, err := gcsql.GetFileBans(filterBoardID, limit)
			if err != nil {
				return "", err
			}
			filenameBans, err := gcsql.GetFilenameBans(filterBoardID, limit)
			if err != nil {
				return "", err
			}
			manageBansBuffer := bytes.NewBufferString("")

			if err = serverutil.MinifyTemplate(gctemplates.ManageFileBans, map[string]interface{}{
				"allBoards":     gcsql.AllBoards,
				"checksumBans":  checksumBans,
				"filenameBans":  filenameBans,
				"filterboardid": filterBoardID,
			}, manageBansBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_filebans.html").Caller().Send()
				return "", errors.New("Error executing ban management page template: " + err.Error())
			}
			outputStr := manageBansBuffer.String()
			return outputStr, nil
		},
	},
	{
		ID:          "namebans",
		Title:       "Name bans",
		Permissions: ModPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			doNameBan := request.FormValue("donameban")
			deleteIDstr := request.FormValue("del")
			if deleteIDstr != "" {
				deleteID, err := strconv.Atoi(deleteIDstr)
				if err != nil {
					errEv.Err(err).
						Str("delStr", deleteIDstr).
						Caller().Send()
					return "", err
				}
				if err = gcsql.DeleteNameBan(deleteID); err != nil {
					errEv.Err(err).
						Int("deleteID", deleteID).
						Caller().Msg("Unable to delete name ban")
					return "", errors.New("Unable to delete name ban: " + err.Error())
				}
			}
			data := map[string]interface{}{
				"currentStaff": staff.Username,
				"allBoards":    gcsql.AllBoards,
			}
			if doNameBan == "Create" {
				var name string
				if name, err = getStringField("name", staff.Username, request); err != nil {
					return "", err
				}
				if name == "" {
					return "", errors.New("name field must not be empty in name ban submission")
				}
				var boardID int
				if boardID, err = getIntField("boardid", staff.Username, request); err != nil {
					return "", err
				}
				isRegex := request.FormValue("isregex") == "on"
				if _, err = gcsql.NewNameBan(name, isRegex, boardID, staff.ID, request.FormValue("staffnote")); err != nil {
					errEv.Err(err).
						Str("name", name).
						Int("boardID", boardID)
					return "", err
				}
			}
			if data["nameBans"], err = gcsql.GetNameBans(0, 0); err != nil {
				return "", err
			}
			buf := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageNameBans, data, buf, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_namebans.html").Caller().Send()
				return "", errors.New("Error executing name ban management page template: " + err.Error())
			}
			return buf.String(), nil
		},
	},
	{
		ID:          "ipsearch",
		Title:       "IP Search",
		Permissions: ModPerms,
		JSONoutput:  NoJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			ipQuery := request.FormValue("ip")
			limitStr := request.FormValue("limit")
			data := map[string]interface{}{
				"ipQuery": ipQuery,
				"limit":   20,
			}

			if ipQuery != "" && limitStr != "" {
				var limit int
				if limit, err = strconv.Atoi(limitStr); err == nil && limit > 0 {
					data["limit"] = limit
				}
				var names []string
				if names, err = net.LookupAddr(ipQuery); err == nil {
					data["reverseAddrs"] = names
				} else {
					data["reverseAddrs"] = []string{err.Error()}
				}

				data["posts"], err = building.GetBuildablePostsByIP(ipQuery, limit)
				if err != nil {
					errEv.Err(err).
						Str("ipQuery", ipQuery).
						Int("limit", limit).
						Bool("onlyNotDeleted", true).
						Caller().Send()
					return "", fmt.Errorf("Error getting list of posts from %q by staff %s: %s", ipQuery, staff.Username, err.Error())
				}
			}

			manageIpBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageIPSearch, data, manageIpBuffer, "text/html"); err != nil {
				errEv.Err(err).
					Str("template", "manage_ipsearch.html").
					Caller().Send()
				return "", errors.New("Error executing IP search page template:" + err.Error())
			}
			return manageIpBuffer.String(), nil
		}},
	{
		ID:          "reports",
		Title:       "Reports",
		Permissions: ModPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			dismissIDstr := request.FormValue("dismiss")
			if dismissIDstr != "" {
				// staff is dismissing a report
				dismissID := gcutil.HackyStringToInt(dismissIDstr)
				block := request.FormValue("block")
				if block != "" && staff.Rank != 3 {
					errEv.
						Int("postID", dismissID).
						Str("rejected", "not an admin").
						Caller().Send()
					return "", errors.New("only the administrator can block reports")
				}
				found, err := gcsql.ClearReport(dismissID, staff.ID, block != "" && staff.Rank == 3)
				if err != nil {
					errEv.Err(err).
						Int("postID", dismissID).
						Caller().Send()
					return nil, err
				}
				if !found {
					return nil, errors.New("no matching reports")
				}
				infoEv.
					Int("reportID", dismissID).
					Bool("blocked", block != "").
					Msg("Report cleared")
			}
			rows, err := gcsql.QuerySQL(`SELECT id,
				handled_by_staff_id as staff_id,
				(SELECT username FROM DBPREFIXstaff WHERE id = DBPREFIXreports.handled_by_staff_id) as staff_user,
				post_id, ip, reason, is_cleared from DBPREFIXreports WHERE is_cleared = FALSE`)
			if err != nil {
				return nil, err
			}
			defer rows.Close()
			reports := make([]map[string]interface{}, 0)
			for rows.Next() {
				var id int
				var staff_id interface{}
				var staff_user []byte
				var post_id int
				var ip string
				var reason string
				var is_cleared int
				err = rows.Scan(&id, &staff_id, &staff_user, &post_id, &ip, &reason, &is_cleared)
				if err != nil {
					return nil, err
				}

				post, err := gcsql.GetPostFromID(post_id, true)
				if err != nil {
					return nil, err
				}

				staff_id_int, _ := staff_id.(int64)
				reports = append(reports, map[string]interface{}{
					"id":         id,
					"staff_id":   int(staff_id_int),
					"staff_user": string(staff_user),
					"post_link":  post.WebPath(),
					"ip":         ip,
					"reason":     reason,
					"is_cleared": is_cleared,
				})
			}
			if wantsJSON {
				return reports, err
			}
			reportsBuffer := bytes.NewBufferString("")
			err = serverutil.MinifyTemplate(gctemplates.ManageReports,
				map[string]interface{}{
					"reports": reports,
					"staff":   staff,
				}, reportsBuffer, "text/html")
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			output = reportsBuffer.String()
			return
		}},
	{
		ID:          "staff",
		Title:       "Staff",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			var outputStr string
			do := request.FormValue("do")
			allStaff, err := getAllStaffNopass(true)
			if wantsJSON {
				if err != nil {
					errEv.Err(err).Caller().Msg("Failed getting staff list")
				}
				return allStaff, err
			}
			if err != nil {
				errEv.Err(err).Caller().Msg("Failed getting staff list")
				err = errors.New("Error getting staff list: " + err.Error())
				return "", err
			}

			for _, staff := range allStaff {
				username := request.FormValue("username")
				password := request.FormValue("password")
				rank := request.FormValue("rank")
				rankI, _ := strconv.Atoi(rank)
				if do == "add" {
					if _, err = gcsql.NewStaff(username, password, rankI); err != nil {
						errEv.
							Str("newStaff", username).
							Str("newPass", password).
							Int("newRank", rankI).
							Caller().Msg("Error creating new staff account")
						return "", fmt.Errorf("Error creating new staff account %q by %q: %s",
							username, staff.Username, err.Error())
					}
				} else if do == "del" && username != "" {
					if err = gcsql.DeactivateStaff(username); err != nil {
						errEv.Err(err).
							Str("delStaff", username).
							Caller().Msg("Error deleting staff account")
						return "", fmt.Errorf("Error deleting staff account %q by %q: %s",
							username, staff.Username, err.Error())
					}
				}
				allStaff, err = getAllStaffNopass(true)
				if err != nil {
					errEv.Err(err).Caller().Msg("Error getting updated staff list")
					err = errors.New("Error getting updated staff list: " + err.Error())
					return "", err
				}

				switch {
				case staff.Rank == 3:
					rank = "admin"
				case staff.Rank == 2:
					rank = "mod"
				case staff.Rank == 1:
					rank = "janitor"
				}
			}

			staffBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageStaff, map[string]interface{}{
				"allstaff":        allStaff,
				"currentUsername": staff.Username,
			}, staffBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_staff.html").Send()
				return "", errors.New("Error executing staff management page template: " + err.Error())
			}
			outputStr += staffBuffer.String()
			return outputStr, nil
		}},
	{
		ID:          "threadattrs",
		Title:       "View/Update Thread Attributes",
		Permissions: ModPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv, errEv *zerolog.Event) (output interface{}, err error) {
			boardDir := request.FormValue("board")
			attrBuffer := bytes.NewBufferString("")
			if boardDir == "" {
				if wantsJSON {
					return nil, errors.New(`missing required field "board"`)
				}
				if err = serverutil.MinifyTemplate(gctemplates.ManageThreadAttrs, map[string]interface{}{
					"action": "threadattrs",
					"boards": gcsql.AllBoards,
				}, attrBuffer, "text/html"); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				return attrBuffer.String(), nil
			}
			errEv.Str("boardDir", boardDir)
			boardID, err := gcsql.GetBoardIDFromDir(boardDir)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}

			var updateID int
			for name, val := range request.Form {
				if len(val) > 0 && val[0] == "Update attributes" {
					if _, err = fmt.Sscanf(name, "update-%d", &updateID); err != nil {
						return "", fmt.Errorf("invalid input name %q: %s", name, err.Error())
					}
				}
			}

			threads, err := gcsql.GetThreadsWithBoardID(boardID, true)
			var threadIDs []interface{}
			for _, thread := range threads {
				threadIDs = append(threadIDs, thread.ID)
			}
			if err != nil {
				errEv.Err(err).Caller().
					Int("boardID", boardID).Send()
				return "", err
			}
			if wantsJSON {
				return threads, nil
			}
			board := gcsql.Board{
				ID:  boardID,
				Dir: boardDir,
			}

			opIDs, err := gcsql.GetTopPostIDsInThreadIDs(threadIDs...)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			if err = serverutil.MinifyTemplate(gctemplates.ManageThreadAttrs, map[string]interface{}{
				"action":  "threadattrs",
				"boards":  gcsql.AllBoards,
				"board":   board,
				"threads": threads,
				"opIDs":   opIDs,
			}, attrBuffer, "text/html"); err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			return attrBuffer.String(), nil
		}},
	{
		ID:          "login",
		Title:       "Login",
		Permissions: NoPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			systemCritical := config.GetSystemCriticalConfig()
			if staff.Rank > 0 {
				http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage"), http.StatusFound)
			}
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirectAction := request.FormValue("action")
			if redirectAction == "" || redirectAction == "logout" {
				redirectAction = "dashboard"
			}

			if username == "" || password == "" {
				//assume that they haven't logged in
				manageLoginBuffer := bytes.NewBufferString("")
				if err = serverutil.MinifyTemplate(gctemplates.ManageLogin, map[string]interface{}{
					"siteConfig":  config.GetSiteConfig(),
					"sections":    gcsql.AllSections,
					"boards":      gcsql.AllBoards,
					"boardConfig": config.GetBoardConfig(""),
					"redirect":    redirectAction,
				}, manageLoginBuffer, "text/html"); err != nil {
					errEv.Err(err).Str("template", "manage_login.html").Send()
					return "", errors.New("Error executing staff login page template: " + err.Error())
				}
				output = manageLoginBuffer.String()
			} else {
				key := gcutil.Md5Sum(request.RemoteAddr + username + password + systemCritical.RandomSeed + gcutil.RandomString(3))[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage/"+request.FormValue("redirect")), http.StatusFound)
			}
			return
		}},
	{
		ID:          "announcements",
		Title:       "Announcements",
		Permissions: JanitorPerms,
		JSONoutput:  AlwaysJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			// return an array of announcements and any errors
			return gcsql.GetAllAccouncements()
		}},
	{
		ID:          "staffinfo",
		Permissions: NoPerms,
		JSONoutput:  AlwaysJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			return staff, nil
		}},
	{
		ID:          "boards",
		Title:       "Boards",
		Permissions: AdminPerms,
		JSONoutput:  NoJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			board := &gcsql.Board{
				MaxFilesize:      1000 * 1000 * 15,
				AnonymousName:    "Anonymous",
				EnableCatalog:    true,
				MaxMessageLength: 1500,
				AutosageAfter:    200,
				NoImagesAfter:    0,
			}
			requestType, _, _ := boardsRequestType(request)
			switch requestType {
			case "create":
				// create button clicked, create the board with the request fields
				if err = getBoardDataFromForm(board, request); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				if err = gcsql.CreateBoard(board, true); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
				infoEv.
					Str("createBoard", board.Dir).
					Int("boardID", board.ID).
					Msg("New board created")
			case "delete":
				// delete button clicked, delete the board
				boardID, err := getIntField("board", staff.Username, request, 0)
				if err != nil {
					return "", err
				}
				// use a temporary variable so that the form values aren't filled
				var deleteBoard *gcsql.Board
				if deleteBoard, err = gcsql.GetBoardFromID(boardID); err != nil {
					errEv.Err(err).Int("deleteBoardID", boardID).Caller().Send()
					return "", err
				}
				if err = deleteBoard.Delete(); err != nil {
					errEv.Err(err).Str("deleteBoard", deleteBoard.Dir).Caller().Send()
					return "", err
				}
				infoEv.
					Str("deleteBoard", deleteBoard.Dir).Send()
				if err = os.RemoveAll(deleteBoard.AbsolutePath()); err != nil {
					errEv.Err(err).Caller().Send()
					return "", err
				}
			case "edit":
				// edit button clicked, fill the input fields with board data to be edited
				boardID, err := getIntField("board", staff.Username, request, 0)
				if err != nil {
					return "", err
				}
				if board, err = gcsql.GetBoardFromID(boardID); err != nil {
					errEv.Err(err).
						Int("boardID", boardID).
						Caller().Msg("Unable to get board info")
					return "", err
				}
			case "modify":
				// save changes button clicked, apply changes to the board based on the request fields
				if err = getBoardDataFromForm(board, request); err != nil {
					return "", err
				}
				if err = board.ModifyInDB(); err != nil {
					return "", errors.New("Unable to apply changes: " + err.Error())
				}
			case "cancel":
				// cancel button was clicked
				fallthrough
			case "":
				fallthrough
			default:
				// board.SetDefaults("", "", "")
			}

			if requestType == "create" || requestType == "modify" || requestType == "delete" {
				if err = gcsql.ResetBoardSectionArrays(); err != nil {
					errEv.Err(err).Caller().Send()
					return "", errors.New("unable to reset board list: " + err.Error())
				}
				if err = building.BuildBoardListJSON(); err != nil {
					return "", err
				}
				if err = building.BuildBoards(false); err != nil {
					return "", err
				}
			}
			pageBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageBoards,
				map[string]interface{}{
					"siteConfig":  config.GetSiteConfig(),
					"sections":    gcsql.AllSections,
					"boards":      gcsql.AllBoards,
					"boardConfig": config.GetBoardConfig(""),
					"editing":     requestType == "edit",
					"board":       board,
				}, pageBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_boards.html").Caller().Send()
				return "", err
			}

			return pageBuffer.String(), nil
		}},
	{
		ID:          "boardsections",
		Title:       "Board sections",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			section := &gcsql.Section{}
			editID := request.Form.Get("edit")
			updateID := request.Form.Get("updatesection")
			deleteID := request.Form.Get("delete")
			if editID != "" {
				if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(editID)); err != nil {
					errEv.Err(err).Caller().Send()
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
			} else if updateID != "" {
				if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(updateID)); err != nil {
					errEv.Err(err).Caller().Send()
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
			} else if deleteID != "" {
				if err = gcsql.DeleteSection(gcutil.HackyStringToInt(deleteID)); err != nil {
					errEv.Err(err).Caller().Send()
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
			}

			if request.PostForm.Get("save_section") != "" {
				// user is creating a new board section
				if section == nil {
					section = &gcsql.Section{}
				}
				section.Name = request.PostForm.Get("sectionname")
				section.Abbreviation = request.PostForm.Get("sectionabbr")
				section.Hidden = request.PostForm.Get("sectionhidden") == "on"
				section.Position, err = strconv.Atoi(request.PostForm.Get("sectionpos"))
				if section.Name == "" || section.Abbreviation == "" || request.PostForm.Get("sectionpos") == "" {
					return "", &ErrStaffAction{
						ErrorField: "formerror",
						Action:     "boardsections",
						Message:    "Missing section title, abbreviation, or hidden status data",
					}
				} else if err != nil {
					errEv.Err(err).Caller().Send()
					return "", &ErrStaffAction{
						ErrorField: "formerror",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
				if updateID != "" {
					// submitting changes to the section
					err = section.UpdateValues()
				} else {
					// creating a new section
					section, err = gcsql.NewSection(section.Name, section.Abbreviation, section.Hidden, section.Position)
				}
				if err != nil {
					errEv.Err(err).Caller().Send()
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
				gcsql.ResetBoardSectionArrays()
			}

			sections, err := gcsql.GetAllSections(false)
			if err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			pageBuffer := bytes.NewBufferString("")
			pageMap := map[string]interface{}{
				"siteConfig": config.GetSiteConfig(),
				"sections":   sections,
			}
			if section.ID > 0 {
				pageMap["edit_section"] = section
			}
			if err = serverutil.MinifyTemplate(gctemplates.ManageSections, pageMap, pageBuffer, "text/html"); err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			output = pageBuffer.String()
			return
		}},
	{
		ID:          "rebuildfront",
		Title:       "Rebuild front page",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				return "", err
			}
			err = building.BuildFrontPage()
			if wantsJSON {
				return map[string]string{
					"front": "Built front page successfully",
				}, err
			}
			return "Built front page successfully", err
		}},
	{
		ID:          "rebuildall",
		Title:       "Rebuild everything",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			gctemplates.InitTemplates()
			if err = gcsql.ResetBoardSectionArrays(); err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			buildErr := &ErrStaffAction{
				ErrorField: "builderror",
				Action:     "rebuildall",
			}
			buildMap := map[string]string{}
			if err = building.BuildFrontPage(); err != nil {
				buildErr.Message = "Error building front page: " + err.Error()
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["front"] = "Built front page successfully"

			if err = building.BuildBoardListJSON(); err != nil {
				buildErr.Message = "Error building board list: " + err.Error()
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["boardlist"] = "Built board list successfully"

			if err = building.BuildBoards(false); err != nil {
				buildErr.Message = "Error building boards: " + err.Error()
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["boards"] = "Built boards successfully"

			if err = building.BuildJS(); err != nil {
				buildErr.Message = "Error building consts.js: " + err.Error()
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			if wantsJSON {
				return buildMap, nil
			}
			buildStr := ""
			for _, msg := range buildMap {
				buildStr += fmt.Sprintln(msg, "<hr />")
			}
			return buildStr, nil
		}},
	{
		ID:          "rebuildboards",
		Title:       "Rebuild boards",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				errEv.Err(err).Caller().Msg("Unable to initialize templates")
				return "", err
			}
			err = building.BuildBoards(false)
			if err != nil {
				return "", err
			}
			if wantsJSON {
				return map[string]interface{}{
					"success": true,
					"message": "Boards built successfully",
				}, nil
			}
			return "Boards built successfully", nil
		}},
	{
		ID:          "reparsehtml",
		Title:       "Reparse HTML",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			var outputStr string
			tx, err := gcsql.BeginTx()
			if err != nil {
				errEv.Err(err).Msg("Unable to begin transaction")
				return "", errors.New("unable to begin SQL transaction")
			}
			defer tx.Rollback()
			const query = `SELECT
			id, message_raw, thread_id as threadid,
			(SELECT id FROM DBPREFIXposts WHERE is_top_post = TRUE AND thread_id = threadid LIMIT 1) AS op,
			(SELECT board_id FROM DBPREFIXthreads WHERE id = threadid) AS boardid,
			(SELECT dir FROM DBPREFIXboards WHERE id = boardid) AS dir
			FROM DBPREFIXposts WHERE is_deleted = FALSE`
			const updateQuery = `UPDATE DBPREFIXposts SET message = ? WHERE id = ?`

			stmt, err := gcsql.PrepareSQL(query, tx)
			if err != nil {
				errEv.Err(err).Caller().Msg("Unable to prepare SQL query")
				return "", err
			}
			defer stmt.Close()
			rows, err := stmt.Query()
			if err != nil {
				errEv.Err(err).Msg("Unable to query the database")
				return "", err
			}
			defer rows.Close()
			for rows.Next() {
				var postID, threadID, opID, boardID int
				var messageRaw, boardDir string
				if err = rows.Scan(&postID, &messageRaw, &threadID, &opID, &boardID, &boardDir); err != nil {
					errEv.Err(err).Caller().Msg("Unable to scan SQL row")
					return "", err
				}
				formatted := posting.FormatMessage(messageRaw, boardDir)
				gcsql.ExecSQL(updateQuery, formatted, postID)
			}
			outputStr += "Done reparsing HTML<hr />"

			if err = building.BuildFrontPage(); err != nil {
				return "", err
			}
			outputStr += "Done building front page<hr />"

			if err = building.BuildBoardListJSON(); err != nil {
				return "", err
			}
			outputStr += "Done building board list JSON<hr />"

			if err = building.BuildBoards(false); err != nil {
				return "", err
			}
			outputStr += "Done building boards<hr />"
			return outputStr, nil
		}},
	{
		ID:          "postinfo",
		Title:       "Post info",
		Permissions: ModPerms,
		JSONoutput:  AlwaysJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			postIDstr := request.FormValue("postid")
			if postIDstr == "" {
				return "", errors.New("invalid request (missing postid)")
			}
			var postID int
			if postID, err = strconv.Atoi(postIDstr); err != nil {
				return "", err
			}
			post, err := gcsql.GetPostFromID(postID, true)
			if err != nil {
				return "", err
			}

			postInfo := map[string]interface{}{
				"post": post,
				"ip":   post.IP,
			}
			names, err := net.LookupAddr(post.IP)
			if err == nil {
				postInfo["ipFQDN"] = names
			} else {
				postInfo["ipFQDN"] = []string{err.Error()}
			}
			return postInfo, nil
		}},
	{
		ID:          "wordfilters",
		Title:       "Wordfilters",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			managePageBuffer := bytes.NewBufferString("")
			editIDstr := request.FormValue("edit")
			deleteIDstr := request.FormValue("delete")
			if deleteIDstr != "" {
				var result sql.Result
				if result, err = gcsql.ExecSQL(`DELETE FROM DBPREFIXwordfilters WHERE id = ?`, deleteIDstr); err != nil {
					return err, err
				}
				if numRows, _ := result.RowsAffected(); numRows < 1 {
					err = invalidWordfilterID(deleteIDstr)
					errEv.Err(err).Caller().Send()
					return err, err
				}
				infoEv.Str("deletedWordfilterID", deleteIDstr)
			}

			submitBtn := request.FormValue("dowordfilter")
			switch submitBtn {
			case "Edit wordfilter":
				regexCheckStr := request.FormValue("isregex")
				if regexCheckStr == "on" {
					regexCheckStr = "1"
				} else {
					regexCheckStr = "0"
				}
				_, err = gcsql.ExecSQL(`UPDATE DBPREFIXwordfilters
					SET board_dirs = ?,
					staff_note = ?,
					search = ?,
					is_regex = ?,
					change_to = ?
					WHERE id = ?`,
					request.FormValue("boarddirs"),
					request.FormValue("staffnote"),
					request.FormValue("find"),
					regexCheckStr,
					request.FormValue("replace"),
					editIDstr)
				infoEv.Str("do", "update")
			case "Create new wordfilter":
				_, err = gcsql.CreateWordFilter(
					request.FormValue("find"),
					request.FormValue("replace"),
					request.FormValue("isregex") == "on",
					request.FormValue("boarddirs"),
					staff.ID,
					request.FormValue("staffnote"))
				infoEv.Str("do", "create")
			case "":
				infoEv.Discard()
			}
			if err == nil {
				infoEv.
					Str("find", request.FormValue("find")).
					Str("replace", request.FormValue("replace")).
					Str("staffnote", request.FormValue("staffnote")).
					Str("boarddirs", request.FormValue("boarddirs"))
			} else {
				return err, err
			}

			wordfilters, err := gcsql.GetWordfilters()
			if err != nil {
				errEv.Err(err).Caller().Msg("Unable to get wordfilters")
				return wordfilters, err
			}
			var editFilter *gcsql.Wordfilter
			if editIDstr != "" {
				editID := gcutil.HackyStringToInt(editIDstr)
				for _, filter := range wordfilters {
					if filter.ID == editID {
						editFilter = &filter
						break
					}
				}
			}
			filterMap := map[string]interface{}{
				"wordfilters": wordfilters,
				"edit":        editFilter,
			}

			err = serverutil.MinifyTemplate(gctemplates.ManageWordfilters,
				filterMap, managePageBuffer, "text/html")
			if err != nil {
				errEv.Err(err).Str("template", "manage_wordfilters.html").Caller().Send()

			}
			infoEv.Send()
			return managePageBuffer.String(), err
		},
	},
}
