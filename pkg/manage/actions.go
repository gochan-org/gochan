package manage

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting"
	"github.com/gochan-org/gochan/pkg/serverutil"
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

	// Callback executes the staff page. if wantsJSON is true, it returns an object to be marshalled
	// into JSON. Otherwise, a string assumed to be valid HTML is returned.
	Callback func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) `json:"-"`
}

var actions = []Action{
	{
		ID:          "logout",
		Title:       "Logout",
		Permissions: JanitorPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			cookie, _ := request.Cookie("sessiondata")
			cookie.MaxAge = 0
			cookie.Expires = time.Now().Add(-7 * 24 * time.Hour)
			http.SetCookie(writer, cookie)
			return "<br />Logged out successfully", nil
		}},
	{
		ID:          "cleanup",
		Title:       "Cleanup",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			outputStr := `<h2>Cleanup</h2><br />`
			if request.FormValue("run") == "Run Cleanup" {
				outputStr += "Removing deleted posts from the database.<hr />"
				if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
					err = errors.New(
						gclog.Print(gclog.LErrorLog, "Error removing deleted posts from database: ", err.Error()))
					return outputStr + "<tr><td>" + err.Error() + "</td></tr></table>", err
				}
				// TODO: remove orphaned replies and uploads

				outputStr += "Optimizing all tables in database.<hr />"
				err = gcsql.OptimizeDatabase()
				if err != nil {
					err = errors.New(
						gclog.Print(gclog.LErrorLog, "Error optimizing SQL tables: ", err.Error()))
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
	{
		ID:          "recentposts",
		Title:       "Recent posts",
		Permissions: JanitorPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			var outputStr string
			systemCritical := config.GetSystemCriticalConfig()
			limit := gcutil.HackyStringToInt(request.FormValue("limit"))
			if limit == 0 {
				limit = 50
			}
			output = `<h1 class="manage-header">Recent posts</h1>` +
				`Limit by: <select id="limit">` +
				`<option>25</option><option>50</option><option>100</option><option>200</option>` +
				`</select><br /><table width="100%%d" border="1">` +
				`<colgroup><col width="25%%" /><col width="50%%" /><col width="17%%" /></colgroup>` +
				`<tr><th></th><th>Message</th><th>Time</th></tr>`
			recentposts, err := gcsql.GetRecentPostsGlobal(limit, false) //only uses boardname, boardid, postid, parentid, message, ip and timestamp
			if wantsJSON {
				return recentposts, err
			}

			if err != nil {
				errMsg := gclog.Println(gclog.LErrorLog, "Error getting recent posts:", err.Error())
				err = errors.New(errMsg)
				if wantsJSON {
					return ErrStaffAction{
						ErrorField: "recentpostserror",
						Action:     "recentposts",
						Message:    errMsg,
					}, err
				}
				return errMsg, err
			}

			for _, recentpost := range recentposts {
				outputStr += fmt.Sprintf(
					`<tr><td><b>Post:</b> <a href="%s">%s/%d</a><br /><b>IP:</b> %s</td><td>%s</td><td>%s</td></tr>`,
					path.Join(systemCritical.WebRoot, recentpost.BoardName, "/res/", strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID)),
					recentpost.BoardName, recentpost.PostID, recentpost.IP, string(recentpost.Message),
					recentpost.Timestamp.Format("01/02/06, 15:04"),
				)
			}
			outputStr += "</table>"
			return
		}},
	{
		ID:          "bans",
		Title:       "Bans",
		Permissions: ModPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) { //TODO whatever this does idk man
			var outputStr string
			var post gcsql.Post
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
					return "", err
				}
				expires := time.Now().Add(duration)

				boards := request.FormValue("boards")
				reason := html.EscapeString(request.FormValue("reason"))
				staffNote := html.EscapeString(request.FormValue("staffnote"))
				currentStaff, _ := getCurrentStaff(request)

				if filename != "" {
					err = gcsql.CreateFileNameBan(filename, nameIsRegex, currentStaff, permaban, staffNote, boards)
				}
				if err != nil {
					outputStr += err.Error()
					err = nil
				}
				if name != "" {
					if err = gcsql.CreateUserNameBan(name, nameIsRegex, currentStaff, permaban, staffNote, boards); err != nil {
						return "", err
					}
				}

				if request.FormValue("fullban") == "on" {
					err = gcsql.CreateUserBan(ip, false, currentStaff, boards, expires, permaban, staffNote, reason, true, time.Now())
					if err != nil {
						return "", err
					}
				} else {
					if request.FormValue("threadban") == "on" {
						err = gcsql.CreateUserBan(ip, true, currentStaff, boards, expires, permaban, staffNote, reason, true, time.Now())
						if err != nil {
							return "", err

						}
					}
					if request.FormValue("imageban") == "on" {
						err = gcsql.CreateFileBan(checksum, currentStaff, permaban, staffNote, boards)
						if err != nil {
							return "", err
						}
					}
				}
			}

			if request.FormValue("postid") != "" {
				var err error
				post, err = gcsql.GetSpecificPostByString(request.FormValue("postid"))
				if err != nil {
					err = errors.New("Error getting post: " + err.Error())
					return "", err
				}
			}

			banlist, err := gcsql.GetAllBans()
			if err != nil {
				err = errors.New("Error getting ban list: " + err.Error())
				return "", err
			}
			manageBansBuffer := bytes.NewBufferString("")

			if err = serverutil.MinifyTemplate(gctemplates.ManageBans,
				map[string]interface{}{
					// "systemCritical": config.GetSystemCriticalConfig(),
					"banlist": banlist,
					"post":    post,
				},
				manageBansBuffer, "text/html"); err != nil {
				return "", errors.New(gclog.Print(gclog.LErrorLog,
					"Error executing ban management page template: "+err.Error()))
			}
			outputStr += manageBansBuffer.String()
			return outputStr, nil
		}},
	{
		ID:          "staff",
		Title:       "Staff",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			var outputStr string
			do := request.FormValue("do")
			allStaff, err := gcsql.GetAllStaffNopass(true)
			if wantsJSON {
				return allStaff, err
			}
			if err != nil {
				err = errors.New(gclog.Print(gclog.LErrorLog, "Error getting staff list: ", err.Error()))
				return "", err
			}

			for _, staff := range allStaff {
				username := request.FormValue("username")
				password := request.FormValue("password")
				rank := request.FormValue("rank")
				rankI, _ := strconv.Atoi(rank)
				if do == "add" {
					if err = gcsql.NewStaff(username, password, rankI); err != nil {
						serverutil.ServeErrorPage(writer, gclog.Printf(gclog.LErrorLog,
							"Error creating new staff account %q: %s", username, err.Error()))
						return
					}
				} else if do == "del" && username != "" {
					if err = gcsql.DeleteStaff(username); err != nil {
						serverutil.ServeErrorPage(writer, gclog.Printf(gclog.LErrorLog,
							"Error deleting staff account %q : %s", username, err.Error()))
						return
					}
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
			if err = serverutil.MinifyTemplate(gctemplates.ManageStaff,
				map[string]interface{}{
					"allstaff": allStaff,
				},
				staffBuffer, "text/html"); err != nil {
				return "", errors.New(gclog.Print(gclog.LErrorLog,
					"Error executing staff management page template: ", err.Error()))
			}
			outputStr += staffBuffer.String()
			return outputStr, nil
		}},
	{
		ID:          "login",
		Title:       "Login",
		Permissions: NoPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			systemCritical := config.GetSystemCriticalConfig()
			if GetStaffRank(request) > 0 {
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
				if err = serverutil.MinifyTemplate(gctemplates.ManageLogin,
					map[string]interface{}{
						"webroot":      config.GetSystemCriticalConfig().WebRoot,
						"site_config":  config.GetSiteConfig(),
						"sections":     gcsql.AllSections,
						"boards":       gcsql.AllBoards,
						"board_config": config.GetBoardConfig(""),
						"redirect":     redirectAction,
					}, manageLoginBuffer, "text/html"); err != nil {
					return "", errors.New(gclog.Print(gclog.LErrorLog,
						"Error executing staff login page template: ", err.Error()))
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
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			// return an array of announcements and any errors
			return gcsql.GetAllAccouncements()
		}},
	{
		ID:          "staffinfo",
		Permissions: NoPerms,
		JSONoutput:  AlwaysJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			staff, err := getCurrentFullStaff(request)
			return staff, err
		}},
	{
		ID:          "boards",
		Title:       "Boards",
		Permissions: AdminPerms,
		JSONoutput:  NoJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			pageBuffer := bytes.NewBufferString("")
			var board gcsql.Board
			requestType, boardID, err := boardsRequestType(request)
			if err != nil {
				return "", err
			}
			if requestType == "cancel" || requestType == "" {
				board.SetDefaults("", "", "")
			}
			switch requestType {
			case "create":
				// create button clicked, create the board with the request fields
				board.ChangeFromRequest(request, false)
				err = board.Create()
			case "delete":
				// delete button clicked, delete the board
				if board, err = gcsql.GetBoardFromID(boardID); err != nil {
					return "", err
				}
				err = board.Delete()
			case "edit":
				// edit button clicked, fill the input fields with board data to be edited
				board, err = gcsql.GetBoardFromID(boardID)
				if err != nil {
					return "", err
				}
			case "modify":
				// save changes button clicked, apply changes to the board based on the request fields
				board, err = gcsql.GetBoardFromID(boardID)
				if err != nil {
					return "", err
				}
				if err = board.ChangeFromRequest(request, true); err != nil {
					return "", err
				}
			case "cancel":
				// cancel button was clicked
				fallthrough
			case "":
				fallthrough
			default:
				board.SetDefaults("", "", "")
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
				if err = building.BuildBoardPages(&board); err != nil {
					return "", err
				}
			}
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
				gclog.Printf(gclog.LErrorLog|gclog.LStaffLog,
					"Error executing manage boards template: %q", err.Error())
				return "", err
			}

			return pageBuffer.String(), nil
		}},
	{
		ID:          "rebuildfront",
		Title:       "Rebuild front page",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				return "", err
			}
			err = building.BuildFrontPage()
			if wantsJSON {
				return map[string]string{
					"front": "Built front page successfully",
				}, err
			}
			return "<h2>Build front page</h2>Built front page successfully", err
		}},
	{
		ID:          "rebuildall",
		Title:       "Rebuild everything",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			gctemplates.InitTemplates()
			gcsql.ResetBoardSectionArrays()
			buildErr := &ErrStaffAction{
				ErrorField: "builderror",
				Action:     "rebuildall",
			}
			buildMap := map[string]string{}
			if err = building.BuildFrontPage(); err != nil {
				buildErr.Message = gclog.Println(gclog.LErrorLog,
					"Error building front page:", err.Error())
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["front"] = "Built front page successfully"

			if err = building.BuildBoardListJSON(); err != nil {
				buildErr.Message = gclog.Println(gclog.LErrorLog,
					"Error building board list:", err.Error())
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["boardlist"] = "Built board list successfully"

			if err = building.BuildBoards(false); err != nil {
				buildErr.Message = gclog.Println(gclog.LErrorLog,
					"Error building boards:", err.Error())
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			buildMap["boards"] = "Built boards successfully"

			if err = building.BuildJS(); err != nil {
				buildErr.Message = gclog.Println(gclog.LErrorLog,
					"Error building consts.js:", err.Error())
				if wantsJSON {
					return buildErr, buildErr
				}
				return buildErr.Message, buildErr
			}
			if wantsJSON {
				return buildMap, nil
			}
			buildStr := "<h2>Rebuilding everything</h2>"
			for _, msg := range buildMap {
				buildStr += fmt.Sprintln(msg, "<hr />")
			}
			return buildStr, nil
		}},
	{
		ID:          "rebuildboard",
		Title:       "Rebuild board",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			return "Not implemented (yet)", gcutil.ErrNotImplemented
			// if err = gctemplates.InitTemplates(); err != nil {
			// 	return "", err
			// }

			// for b, board := range request.Form {
			// 	if b == "board" {
			// 		return board[0], nil
			// 	}
			// }
			// return "", &ErrStaffAction{
			// 	ErrorField: "staffaction",
			// 	Action:     "rebuildboard",
			// 	Message:    fmt.Sprintf("/%s/ is not a board"),
			// }
		}},
	{
		ID:          "rebuildboards",
		Title:       "Rebuild boards",
		Permissions: AdminPerms,
		JSONoutput:  OptionalJSON,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				return "", err
			}
			if wantsJSON {
				return map[string]interface{}{
					"success": true,
					"message": "Boards built successfully",
				}, building.BuildBoards(false)
			}
			return "<h2>Rebuild boards</h2>Boards built successfully", building.BuildBoards(false)
		}},
	{
		ID:          "reparsehtml",
		Title:       "Reparse HTML",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			var outputStr string

			messages, err := gcsql.GetAllNondeletedMessageRaw()
			if err != nil {
				return "", err
			}

			for i := range messages {
				messages[i].Message = posting.FormatMessage(messages[i].MessageRaw)
			}
			if err = gcsql.SetFormattedInDatabase(messages); err != nil {
				return "", err
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
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			post, err := gcsql.GetSpecificPost(gcutil.HackyStringToInt(request.FormValue("postid")), false)
			return post, err
		}},
	{
		ID:          "tempposts",
		Title:       "Temporary posts lists",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			outputStr := `<h1>Temporary posts</h1>`
			if len(gcsql.TempPosts) == 0 {
				outputStr += "No temporary posts<br />"
				return
			}
			for p, post := range gcsql.TempPosts {
				outputStr += fmt.Sprintf("Post[%d]: %#v<br />", p, post)
			}
			return outputStr, nil
		}},
}
