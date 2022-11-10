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
	"github.com/gochan-org/gochan/pkg/serverutil"
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

var (
	chopPortNumRegex = regexp.MustCompile(`(.+|\w+):(\d+)$`)
)

// Action represents the functions accessed by staff members at /manage?action=<functionname>.
type Action struct {
	// the string used when the user requests /manage?action=<id>
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
	Callback func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) `json:"-"`
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
				outputStr += `<form action="/manage?action=cleanup" method="post">` +
					`<input name="run" id="run" type="submit" value="Run Cleanup" />` +
					`</form>`
			}
			return outputStr, nil
		}},
	// {
	// 	ID:          "recentposts",
	// 	Title:       "Recent posts",
	// 	Permissions: JanitorPerms,
	// 	JSONoutput:  OptionalJSON,
	// 	Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool) (output interface{}, err error) {
	// 		limit := gcutil.HackyStringToInt(request.FormValue("limit"))
	// 		if limit == 0 {
	// 			limit = 50
	// 		}
	// 		recentposts, err := gcsql.GetRecentPostsGlobal(limit, false) //only uses boardname, boardid, postid, parentid, message, ip and timestamp
	// 		if wantsJSON || err != nil {
	// 			return recentposts, err
	// 		}
	// 		manageRecentsBuffer := bytes.NewBufferString("")
	// 		if err = serverutil.MinifyTemplate(gctemplates.ManageRecentPosts, map[string]interface{}{
	// 			"recentposts": recentposts,
	// 			"webroot":     config.GetSystemCriticalConfig().WebRoot,
	// 		}, manageRecentsBuffer, "text/html"); err != nil {
	//			errEv.Err(err).Caller().Send()
	// 			return "", errors.New("Error executing ban management page template: " + err.Error())
	// 		}
	// 		return manageRecentsBuffer.String(), nil
	// 	}},
	/* {
		ID:          "filebans",
		Title:       "File bans",
		Permissions: ModPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool) (interface{}, error) {
			var err error
			fileBanType := request.PostForm.Get("bantype")
			delFnbStr := request.Form.Get("delfnb")
			if delFnbStr != "" {
				var delFilenameBanID int
				if delFilenameBanID, err = strconv.Atoi(delFnbStr); err != nil {
					errorEv.Err(err).
						Str("delfnb", delFnbStr).Caller().Send()
					return "", err
				}
				if err = gcsql.DeleteFilenameBanByID(delFilenameBanID); err != nil {
					errorEv.Err(err).
						Int("delfnb", delFilenameBanID).Caller().Send()
					return "", err
				}
				infoEv.Int("delFilenameBan", delFilenameBanID).Send()
			}
			delCsbStr := request.Form.Get("delcsb")
			if delCsbStr != "" {
				var delChecksumBanID int
				if delChecksumBanID, err = strconv.Atoi(delCsbStr); err != nil {
					errorEv.Err(err).
						Str("delcsb", delCsbStr).Send()
					return "", err
				}
				if err = gcsql.DeleteFileBanByID(delChecksumBanID); err != nil {
					errorEv.Err(err).
						Int("delcsb", delChecksumBanID).Send()
					return "", err
				}
				InfoEv.Int("delChecksumBan", delChecksumBanID).Send()
			}
			switch fileBanType {
			case "filename":
				// filename form used
				filename := request.PostForm.Get("filename")
				isWildcard := request.PostForm.Get("iswildcard") == "on"
				board := request.PostForm.Get("board")
				staffNote := request.PostForm.Get("staffnote")
				if filename == "" {
					err = errors.New("missing filename field in filename ban creation")
					errorEv.Err(err).Send()
					return "", err
				}
				if err = gcsql.CreateFileNameBan(filename, isWildcard, staff.Username, staffNote, board); err != nil {
					errorEv.Err(err).
						Str("filename", filename).
						Bool("iswildcard", isWildcard).
						Str("board", board).
						Str("staffnote", staffNote).Send()
					return "", err
				}
				gcutil.LogInfo().
					Str("action", "filebans").
					Str("staff", staff.Username).
					Str("newBanType", "filename").Send()
			case "checksum":
				// file checksum form used
				checksum := request.PostForm.Get("checksum")
				board := request.PostForm.Get("board")
				staffNote := request.PostForm.Get("staffnote")
				if checksum == "" {
					err = errors.New("missing checksum field in filename ban creation")
					errorEv.Err(err).Send()
					return "", err
				}
				if err = gcsql.CreateFileBan(checksum, staff.Username, staffNote, board); err != nil {
					errorEv.Err(err).
						Str("checksum", checksum).
						Str("board", board).
						Str("staffnote", staffNote).Send()
					return "", err
				}
				infoEv.Str("newBanType", "checksum").Send()
			case "":
				// no POST data sent
			default:
				err = fmt.Errorf(`invalid bantype value %q, valid values are "filename" and "checksum"`, fileBanType)
				errorEv.Err(err).Caller().Send()
				return "", err
			}

			filenameBans, err := gcsql.GetFilenameBans("", false)
			if err != nil {
				return "", err
			}
			checksumBans, err := gcsql.GetFileChecksumBans("")
			if err != nil {
				return "", err
			}
			if wantsJSON {
				return map[string]interface{}{
					"filenameBans": filenameBans,
					"checksumBans": checksumBans,
				}, nil
			}

			boardURIs, err := gcsql.GetBoardUris()
			if err != nil {
				return "", err
			}
			manageBansBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageFileBans, map[string]interface{}{
				"webroot":      config.GetSystemCriticalConfig().WebRoot,
				"filenameBans": filenameBans,
				"checksumBans": checksumBans,
				"currentStaff": staff.Username,
				"boardURIs":    boardURIs,
			}, manageBansBuffer, "text/html"); err != nil {
				errorEv.Err(err).Str("template", "manage_filebans.html").Caller().Send()
				return "", errors.New("failed executing file ban management page template: " + err.Error())
			}
			return manageBansBuffer.String(), nil
		},
	}, */
	{
		ID:          "ipbans",
		Title:       "IP Bans",
		Permissions: ModPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			return "", gcutil.ErrNotImplemented
		},
	},
	/* {
	ID:          "bans",
	Title:       "Bans",
	Permissions: ModPerms,
	Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool) (output interface{}, err error) { //TODO whatever this does idk man
		var outputStr string
		var post gcsql.Post
		errorEv := gcutil.LogError(nil).
			Str("action", "bans").
			Str("staff", staff.Username)
		defer errorEv.Discard()

		if request.FormValue("do") == "add" {
			ip := request.FormValue("ip")
			name := request.FormValue("name")
			nameIsRegex := (request.FormValue("nameregex") == "on")
			checksum := request.FormValue("checksum")
			filename := request.FormValue("filename")
			durationForm := request.FormValue("duration")
			permaban := (durationForm == "" || durationForm == "0" || durationForm == "forever")
			duration, err := gcutil.ParseDurationString(durationForm)
			if err != nil {
				errorEv.Err(err).Send()
				return "", err
			}
			expires := time.Now().Add(duration)

			boards := request.FormValue("boards")
			reason := html.EscapeString(request.FormValue("reason"))
			staffNote := html.EscapeString(request.FormValue("staffnote"))

			if filename != "" {
				err = gcsql.CreateFileNameBan(filename, nameIsRegex, staff.Username, staffNote, boards)
			}
			if err != nil {
				outputStr += err.Error()
				errorEv.Err(err).Send()
				err = nil
			}
			if name != "" {
				if err = gcsql.CreateUserNameBan(name, nameIsRegex, staff.Username, permaban, staffNote, boards); err != nil {
					errorEv.
						Str("banType", "username").
						Str("user", name).Send()
					return "", err
				}
				infoEv.
					Str("banType", "username").
					Str("user", name).
					Bool("permaban", permaban).Send()
			}

			if request.FormValue("fullban") == "on" {
				err = gcsql.CreateUserBan(ip, false, staff.Username, boards, expires, permaban, staffNote, reason, true, time.Now())
				if err != nil {
					errorEv.
						Str("banType", "ip").
						Str("banIP", ip).
						Bool("threadBan", false).
						Str("bannedFromBoards", boards).Send()
					return "", err
				}
				infoEv.
					Str("banType", "ip").
					Str("banIP", ip).
					Bool("threadBan", true).
					Str("bannedFromBoards", boards).Send()
			} else {
				if request.FormValue("threadban") == "on" {
					err = gcsql.CreateUserBan(ip, true, staff.Username, boards, expires, permaban, staffNote, reason, true, time.Now())
					if err != nil {
						errorEv.
							Str("banType", "ip").
							Str("banIP", ip).
							Bool("threadBan", true).
							Str("bannedFromBoards", boards).Send()
						return "", err
					}
					infoEv.
						Str("banType", "ip").
						Str("banIP", ip).
						Bool("threadBan", true).
						Str("bannedFromBoards", boards).Send()
				}
				if request.FormValue("imageban") == "on" {
					err = gcsql.CreateFileBan(checksum, staff.Username, staffNote, boards)
					if err != nil {
						errorEv.
							Str("banType", "fileBan").
							Str("checksum", checksum).Send()
						return "", err
					}
					infoEv.
						Str("banType", "fileBan").
						Str("checksum", checksum).Send()
				}
			}
		}

		if request.FormValue("postid") != "" {
			var err error
			post, err = gcsql.GetSpecificPostByString(request.FormValue("postid"), true)
			if err != nil {
				errorEv.Err(err).Str("postid", request.FormValue("postid")).Msg("Error getting post")
				err = errors.New("Error getting post: " + err.Error())
				return "", err
			}
		}

		banlist, err := gcsql.GetAllBans()
		if err != nil {
			errorEv.Err(err).Msg("Error getting ban list")
			err = errors.New("Error getting ban list: " + err.Error())
			return "", err
		}
		manageBansBuffer := bytes.NewBufferString("")

		if err = serverutil.MinifyTemplate(gctemplates.ManageBans, map[string]interface{}{
			// "systemCritical": config.GetSystemCriticalConfig(),
			"banlist": banlist,
			"post":    post,
		}, manageBansBuffer, "text/html"); err != nil {
			errEv.Err(err).Str("template", "manage_bans.html").Caller().Send()
			return "", errors.New("Error executing ban management page template: " + err.Error())
		}
		outputStr += manageBansBuffer.String()
		return outputStr, nil
	}}, */
	{
		ID:          "ipsearch",
		Title:       "IP Search",
		Permissions: ModPerms,
		JSONoutput:  NoJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			ipQuery := request.Form.Get("ip")
			limitStr := request.Form.Get("limit")
			data := map[string]interface{}{
				"webroot": config.GetSystemCriticalConfig().WebRoot,
				"ipQuery": ipQuery,
				"limit":   10,
			}

			if ipQuery != "" && limitStr != "" {
				var limit int
				if limit, err = strconv.Atoi(request.Form.Get("limit")); err != nil || limit < 1 {
					limit = 20
				}
				data["limit"] = limit
				var names []string
				if names, err = net.LookupAddr(ipQuery); err == nil {
					data["reverseAddrs"] = names
				} else {
					data["reverseAddrs"] = []string{err.Error()}
				}

				data["posts"], err = gcsql.GetPostsFromIP(ipQuery, limit, true)
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
					serveError(writer, "permission", "reports", "Only the administrator can block reports", wantsJSON)
					errEv.
						Int("postID", dismissID).
						Str("rejected", "not an admin").
						Caller().Send()
					return "", nil
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
				post_id, ip, reason, is_cleared from DBPREFIXreports WHERE is_cleared = 0`)
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
				"webroot":         config.GetSystemCriticalConfig().WebRoot,
				"currentUsername": staff.Username,
			}, staffBuffer, "text/html"); err != nil {
				errEv.Err(err).Str("template", "manage_staff.html").Send()
				return "", errors.New("Error executing staff management page template: " + err.Error())
			}
			outputStr += staffBuffer.String()
			return outputStr, nil
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
					"webroot":      config.GetSystemCriticalConfig().WebRoot,
					"site_config":  config.GetSiteConfig(),
					"sections":     gcsql.AllSections,
					"boards":       gcsql.AllBoards,
					"board_config": config.GetBoardConfig(""),
					"redirect":     redirectAction,
				}, manageLoginBuffer, "text/html"); err != nil {
					errEv.Err(err).Str("template", "manage_login.html").Send()
					return "", errors.New("Error executing staff login page template: " + err.Error())
				}
				output = manageLoginBuffer.String()
			} else {
				key := gcutil.Md5Sum(request.RemoteAddr + username + password + systemCritical.RandomSeed + gcutil.RandomString(3))[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage?action="+request.FormValue("redirect")), http.StatusFound)
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
			board := new(gcsql.Board)

			requestType, _, _ := boardsRequestType(request)
			switch requestType {
			case "create":
				// create button clicked, create the board with the request fields
				if err = getBoardDataFromForm(board, request); err != nil {
					serveError(writer, err.Error(), "boards", err.Error(), wantsJSON)
					return "", err
				}

				err = gcsql.CreateBoard(board, true)
			case "delete":
				// delete button clicked, delete the board
				boardID, err := getIntField("boardid", staff.Username, request, 0)
				if err != nil {
					serveError(writer, "boardid", "boards", err.Error(), wantsJSON)
					return "", err
				}
				if board, err = gcsql.GetBoardFromID(boardID); err != nil {
					errEv.Err(err).Int("deleteBoardID", boardID).Caller().Send()
					return "", err
				}
				err = board.Delete()
				if err != nil {
					errEv.Err(err).Str("deleteBoard", board.Dir).Caller().Send()
					return "", err
				}
				gcutil.LogInfo().
					Str("staff", staff.Username).
					Str("action", "boards").
					Str("deleteBoard", board.Dir).Send()
				err = os.RemoveAll(board.AbsolutePath())
			case "edit":
				// edit button clicked, fill the input fields with board data to be edited
				err = getBoardDataFromForm(board, request)
			case "modify":
				// save changes button clicked, apply changes to the board based on the request fields
				err = getBoardDataFromForm(board, request)
			case "cancel":
				// cancel button was clicked
				fallthrough
			case "":
				fallthrough
			default:
				// board.SetDefaults("", "", "")
			}
			if err != nil {
				return "", err
			}
			if requestType == "create" || requestType == "modify" && err != nil {
				if err = building.BuildBoardListJSON(); err != nil {
					return "", err
				}
				if err = building.BuildBoards(false, board.ID); err != nil {
					return "", err
				}
				if err = building.BuildBoardPages(board); err != nil {
					return "", err
				}
			}
			pageBuffer := bytes.NewBufferString("")
			if err = serverutil.MinifyTemplate(gctemplates.ManageBoards,
				map[string]interface{}{
					"webroot":      config.GetSystemCriticalConfig().WebRoot,
					"site_config":  config.GetSiteConfig(),
					"sections":     gcsql.AllSections,
					"boards":       gcsql.AllBoards,
					"board_config": config.GetBoardConfig(""),
					"editing":      requestType == "edit",
					"board":        board,
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
		JSONoutput:  NoJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			section := &gcsql.Section{}
			editID := request.Form.Get("edit")
			updateID := request.Form.Get("updatesection")
			deleteID := request.Form.Get("delete")
			if editID != "" {
				if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(editID)); err != nil {
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
			} else if updateID != "" {
				if section, err = gcsql.GetSectionFromID(gcutil.HackyStringToInt(updateID)); err != nil {
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
			} else if deleteID != "" {
				if err = gcsql.DeleteSection(gcutil.HackyStringToInt(deleteID)); err != nil {
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
					return "", &ErrStaffAction{
						ErrorField: "db",
						Action:     "boardsections",
						Message:    err.Error(),
					}
				}
				gcsql.ResetBoardSectionArrays()
			}

			pageBuffer := bytes.NewBufferString("")
			pageMap := map[string]interface{}{
				"webroot":     config.GetSystemCriticalConfig().WebRoot,
				"site_config": config.GetSiteConfig(),
				"sections":    gcsql.AllSections,
			}
			if section.ID > 0 {
				pageMap["edit_section"] = section
			}
			if err = serverutil.MinifyTemplate(gctemplates.ManageSections, pageMap, pageBuffer, "text/html"); err != nil {
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
			gcsql.ResetBoardSectionArrays()
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
	// {
	// 	ID:          "rebuildboard",
	// 	Title:       "Rebuild board",
	// 	Permissions: AdminPerms,
	// 	Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
	// 		if err = gctemplates.InitTemplates(); err != nil {
	// 			return "", err
	// 		}

	// 		for b, board := range request.Form {
	// 			if b == "board" {
	// 				return board[0], nil
	// 			}
	// 		}
	// 		return "", &ErrStaffAction{
	// 			ErrorField: "staffaction",
	// 			Action:     "rebuildboard",
	// 			Message:    fmt.Sprintf("/%s/ is not a board"),
	// 		}
	// 	}},
	{
		ID:          "rebuildboards",
		Title:       "Rebuild boards",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				errEv.Err(err).Caller().Send()
				return "", err
			}
			if wantsJSON {
				return map[string]interface{}{
					"success": true,
					"message": "Boards built successfully",
				}, building.BuildBoards(false)
			}
			return "Boards built successfully", building.BuildBoards(false)
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
	// {
	// 	may end up deleting this
	// 	ID:          "tempposts",
	// 	Title:       "Temporary posts lists",
	// 	Permissions: AdminPerms,
	// 	Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
	// 		outputStr := ""
	// 		if len(gcsql.TempPosts) == 0 {
	// 			outputStr += "No temporary posts"
	// 			return
	// 		}
	// 		for p, post := range gcsql.TempPosts {
	// 			outputStr += fmt.Sprintf("Post[%d]: %#v<br />", p, post)
	// 		}
	// 		return outputStr, nil
	// 	}},
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
			case "Create new wordfilter":
				_, err = gcsql.CreateWordFilter(
					request.FormValue("find"),
					request.FormValue("replace"),
					request.FormValue("isregex") == "on",
					request.FormValue("boarddirs"),
					staff.ID,
					request.FormValue("staffnote"))
			}
			if err != nil {
				return err, err
			}

			wordfilters, err := gcsql.GetWordfilters()
			if err != nil {
				return wordfilters, nil
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
				"webroot":     config.GetSystemCriticalConfig().WebRoot,
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
