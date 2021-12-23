package manage

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
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

var (
	chopPortNumRegex = regexp.MustCompile(`(.+|\w+):(\d+)$`)
)

// Action represents the functions accessed by staff members at /manage?action=<functionname>.
type Action struct {
	Title       string `json:"title"`
	Permissions int    `json:"perms"` // 0 = non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	isJSON      bool   // if it can sometimes return JSON, this should still be false
	// Callback executes the staff page. if wantsJSON is true, it returns an object to be marshalled
	// into JSON. Otherwise, a string assumed to be valid HTML is returned.
	Callback func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) `json:"-"`
}

var actions = map[string]Action{
	"cleanup": {
		Title:       "Cleanup",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			outputStr := `<h2 class="manage-header">Cleanup</h2><br />`
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
	"config": {
		Title:       "Configuration",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			// do := request.FormValue("do")
			// siteCfg := config.GetSiteConfig()
			// boardCfg := config.GetBoardConfig("")
			// var status string
			// if do == "save" {
			// 	configJSON, err := json.MarshalIndent(config.Config, "", "\t")
			// 	if err != nil {
			// 		status += gclog.Println(gclog.LErrorLog, err.Error()) + "<br />"
			// 	} else if err = ioutil.WriteFile("gochan.json", configJSON, 0777); err != nil {
			// 		status += gclog.Println(gclog.LErrorLog,
			// 			"Error backing up old gochan.json, cancelling save:", err.Error())
			// 	} else {
			// 		siteCfg.CookieMaxAge = request.PostFormValue("CookieMaxAge")
			// 		if _, err = gcutil.ParseDurationString(config.Config.CookieMaxAge); err != nil {
			// 			status += err.Error()
			// 			siteCfg.CookieMaxAge = "1y"
			// 		}
			// 		siteCfg.Lockdown = (request.PostFormValue("Lockdown") == "on")
			// 		siteCfg.LockdownMessage = request.PostFormValue("LockdownMessage")
			// 		SillytagsArr := strings.Split(request.PostFormValue("Sillytags"), "\n")
			// 		var Sillytags []string
			// 		for _, tag := range SillytagsArr {
			// 			Sillytags = append(Sillytags, strings.Trim(tag, " \n\r"))
			// 		}

			// 		boardCfg.Sillytags = Sillytags
			// 		boardCfg.UseSillytags = (request.PostFormValue("UseSillytags") == "on")
			// 		siteCfg.Modboard = request.PostFormValue("Modboard")
			// 		siteCfg.SiteName = request.PostFormValue("SiteName")
			// 		siteCfg.SiteSlogan = request.PostFormValue("SiteSlogan")
			// 		// boardCfg.WebRoot = request.PostFormValue("WebRoot")
			// 		// TODO: Change this to match the new Style type in gochan.json
			// 		/* Styles_arr := strings.Split(request.PostFormValue("Styles"), "\n")
			// 		var Styles []string
			// 		for _, style := range Styles_arr {
			// 			Styles = append(Styles, strings.Trim(style, " \n\r"))
			// 		}
			// 		config.Styles = Styles */
			// 		boardCfg.DefaultStyle = request.PostFormValue("DefaultStyle")
			// 		boardCfg.RejectDuplicateImages = (request.PostFormValue("RejectDuplicateImages") == "on")
			// 		NewThreadDelay, err := strconv.Atoi(request.PostFormValue("NewThreadDelay"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.NewThreadDelay = NewThreadDelay
			// 		}

			// 		ReplyDelay, err := strconv.Atoi(request.PostFormValue("ReplyDelay"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ReplyDelay = ReplyDelay
			// 		}

			// 		MaxLineLength, err := strconv.Atoi(request.PostFormValue("MaxLineLength"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.MaxLineLength = MaxLineLength
			// 		}

			// 		ReservedTripsArr := strings.Split(request.PostFormValue("ReservedTrips"), "\n")
			// 		var ReservedTrips []string
			// 		for _, trip := range ReservedTripsArr {
			// 			ReservedTrips = append(ReservedTrips, strings.Trim(trip, " \n\r"))

			// 		}
			// 		boardCfg.ReservedTrips = ReservedTrips

			// 		ThumbWidth, err := strconv.Atoi(request.PostFormValue("ThumbWidth"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbWidth = ThumbWidth
			// 		}

			// 		ThumbHeight, err := strconv.Atoi(request.PostFormValue("ThumbHeight"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbHeight = ThumbHeight
			// 		}

			// 		ThumbWidthReply, err := strconv.Atoi(request.PostFormValue("ThumbWidthReply"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbWidthReply = ThumbWidthReply
			// 		}

			// 		ThumbHeightReply, err := strconv.Atoi(request.PostFormValue("ThumbHeightReply"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbHeightReply = ThumbHeightReply
			// 		}

			// 		ThumbWidthCatalog, err := strconv.Atoi(request.PostFormValue("ThumbWidthCatalog"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbWidthCatalog = ThumbWidthCatalog
			// 		}

			// 		ThumbHeightCatalog, err := strconv.Atoi(request.PostFormValue("ThumbHeightCatalog"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.ThumbHeightCatalog = ThumbHeightCatalog
			// 		}

			// 		RepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("RepliesOnBoardPage"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.RepliesOnBoardPage = RepliesOnBoardPage
			// 		}

			// 		StickyRepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("StickyRepliesOnBoardPage"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.StickyRepliesOnBoardPage = StickyRepliesOnBoardPage
			// 		}

			// 		boardCfg.BanMessage = request.PostFormValue("BanMessage")
			// 		EmbedWidth, err := strconv.Atoi(request.PostFormValue("EmbedWidth"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.EmbedWidth = EmbedWidth
			// 		}

			// 		EmbedHeight, err := strconv.Atoi(request.PostFormValue("EmbedHeight"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.EmbedHeight = EmbedHeight
			// 		}

			// 		boardCfg.EnableEmbeds = (request.PostFormValue("EnableEmbeds") == "on")
			// 		boardCfg.ImagesOpenNewTab = (request.PostFormValue("ImagesOpenNewTab") == "on")
			// 		boardCfg.NewTabOnOutlinks = (request.PostFormValue("NewTabOnOutlinks") == "on")
			// 		boardCfg.DateTimeFormat = request.PostFormValue("DateTimeFormat")
			// 		siteCfg.MinifyHTML = (request.PostFormValue("MinifyHTML") == "on")
			// 		siteCfg.MinifyJS = (request.PostFormValue("MinifyJS") == "on")
			// 		AkismetAPIKey := request.PostFormValue("AkismetAPIKey")

			// 		if err = serverutil.CheckAkismetAPIKey(AkismetAPIKey); err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			siteCfg.AkismetAPIKey = AkismetAPIKey
			// 		}

			// 		boardCfg.UseCaptcha = (request.PostFormValue("UseCaptcha") == "on")
			// 		CaptchaWidth, err := strconv.Atoi(request.PostFormValue("CaptchaWidth"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.CaptchaWidth = CaptchaWidth
			// 		}
			// 		CaptchaHeight, err := strconv.Atoi(request.PostFormValue("CaptchaHeight"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			boardCfg.CaptchaHeight = CaptchaHeight
			// 		}

			// 		boardCfg.EnableGeoIP = (request.PostFormValue("EnableGeoIP") == "on")
			// 		siteCfg.GeoIPDBlocation = request.PostFormValue("GeoIPDBlocation")

			// 		MaxRecentPosts, err := strconv.Atoi(request.PostFormValue("MaxRecentPosts"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			siteCfg.MaxRecentPosts = MaxRecentPosts
			// 		}

			// 		MaxLogDays, err := strconv.Atoi(request.PostFormValue("MaxLogDays"))
			// 		if err != nil {
			// 			status += err.Error() + "<br />"
			// 		} else {
			// 			siteCfg.MaxLogDays = MaxLogDays
			// 		}

			// 		if err = config.WriteConfig(); err != nil {
			// 			status = gclog.Print(gclog.LErrorLog, "Error writing gochan.json: ", err.Error()) + "<br />"
			// 		} else {
			// 			status = "Wrote gochan.json successfully<br />"
			// 		}
			// 	}
			// }
			// manageConfigBuffer := bytes.NewBufferString("")
			// if err = serverutil.MinifyTemplate(gctemplates.ManageConfig,
			// 	map[string]interface{}{
			// 		"siteCfg":  siteCfg,
			// 		"boardCfg": boardCfg,
			// 		"status":   status,
			// 	},
			// 	manageConfigBuffer, "text/html"); err != nil {
			// 	return "", errors.New(gclog.Print(gclog.LErrorLog,
			// 		"Error executing config management page: ", err.Error()))
			// }
			// output += manageConfigBuffer.String()
			// return output, nil
			errStr := "web-based configuration tool has been temporarily disabled"
			return errStr, errors.New(errStr)
		}},
	"dashboard": {
		Title:       "Dashboard",
		Permissions: JanitorPerms,
		isJSON:      false,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			if wantsJSON {
				return "", createNoJSONError("dashboard")
			}
			dashBuffer := bytes.NewBufferString("")

			if err = serverutil.MinifyTemplate(gctemplates.ManageDashboard,
				nil, dashBuffer, "text/html"); err != nil {
				gclog.Printf(gclog.LErrorLog|gclog.LStaffLog,
					"Error executing dashboard template: %q", err.Error())
				return "", err
			}
			return dashBuffer.String(), nil
		}},
	"login": {
		Title:       "Login",
		Permissions: NoPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			if wantsJSON {
				return "", createNoJSONError("dashboard")
			}

			systemCritical := config.GetSystemCriticalConfig()
			if GetStaffRank(request) > 0 {
				http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage"), http.StatusFound)
			}
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirectAction := request.FormValue("action")
			if redirectAction == "" {
				redirectAction = "announcements"
			}
			if username == "" || password == "" {
				//assume that they haven't logged in
				output = `<form method="POST" action="` + systemCritical.WebRoot + `manage?action=login" id="login-box" class="staff-form">` +
					`<input type="hidden" name="redirect" value="` + redirectAction + `" />` +
					`<input type="text" name="username" class="logindata" /><br />` +
					`<input type="password" name="password" class="logindata" /><br />` +
					`<input type="submit" value="Login" />` +
					`</form>`
			} else {
				key := gcutil.Md5Sum(request.RemoteAddr + username + password + systemCritical.RandomSeed + gcutil.RandomString(3))[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(systemCritical.WebRoot, "manage?action="+request.FormValue("redirect")), http.StatusFound)
			}
			return
		}},
	"logout": {
		Title:       "Logout",
		Permissions: JanitorPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			cookie, _ := request.Cookie("sessiondata")
			cookie.MaxAge = 0
			cookie.Expires = time.Now().Add(-7 * 24 * time.Hour)
			http.SetCookie(writer, cookie)
			return "Logged out successfully", nil
		}},
	"announcements": {
		Title:       "Announcements",
		Permissions: JanitorPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			outputStr := `<h1 class="manage-header">Announcements</h1><br />`

			//get all announcements to announcement list
			//loop to html if exist, no announcement if empty
			announcements, err := gcsql.GetAllAccouncements()
			if err != nil {
				return "", err
			}
			if len(announcements) == 0 {
				outputStr += "No announcements"
			} else {
				boardConfig := config.GetBoardConfig("")
				for _, announcement := range announcements {
					outputStr += `<div class="section-block">` +
						`<div class="section-title-block"><b>` + announcement.Subject + `</b> by ` + announcement.Poster + ` at ` + announcement.Timestamp.Format(boardConfig.DateTimeFormat) + `</div>` +
						`<div class="section-body">` + announcement.Message + `</div></div>`
				}
			}
			return outputStr, nil
		}},
	"bans": {
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

			if err = serverutil.MinifyTemplate(gctemplates.ManageConfig,
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
	"staffinfo": {
		Permissions: NoPerms,
		isJSON:      true,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			staff, err := getCurrentFullStaff(request)
			return staff, err
		}},
	"boards": {
		Title:       "Boards",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			var outputStr string
			do := request.FormValue("do")
			var done bool
			board := new(gcsql.Board)
			var boardCreationStatus string
			systemCritical := config.GetSystemCriticalConfig()

			for !done {
				switch {
				case do == "add":
					board.Dir = request.FormValue("dir")
					if board.Dir == "" {
						boardCreationStatus = `Error: "Directory" cannot be blank`
						do = ""
						continue
					}
					orderStr := request.FormValue("order")
					if board.ListOrder, err = strconv.Atoi(orderStr); err != nil {
						board.ListOrder = 0
					}
					board.Title = request.FormValue("title")
					if board.Title == "" {
						boardCreationStatus = `Error: "Title" cannot be blank`
						do = ""
						continue
					}
					board.Subtitle = request.FormValue("subtitle")
					board.Description = request.FormValue("description")
					sectionStr := request.FormValue("section")
					if sectionStr == "none" {
						sectionStr = "0"
					}

					board.CreatedOn = time.Now()
					board.Section, err = strconv.Atoi(sectionStr)
					if err != nil {
						board.Section = 0
					}
					board.MaxFilesize, err = strconv.Atoi(request.FormValue("maximagesize"))
					if err != nil {
						board.MaxFilesize = 1024 * 4
					}

					board.MaxPages, err = strconv.Atoi(request.FormValue("maxpages"))
					if err != nil {
						board.MaxPages = 11
					}

					board.DefaultStyle = strings.Trim(request.FormValue("defaultstyle"), "\n")
					board.Locked = (request.FormValue("locked") == "on")
					board.ForcedAnon = (request.FormValue("forcedanon") == "on")

					board.Anonymous = request.FormValue("anonymous")
					if board.Anonymous == "" {
						board.Anonymous = "Anonymous"
					}

					board.MaxAge, err = strconv.Atoi(request.FormValue("maxage"))
					if err != nil {
						board.MaxAge = 0
					}

					board.AutosageAfter, err = strconv.Atoi(request.FormValue("autosageafter"))
					if err != nil {
						board.AutosageAfter = 200
					}

					board.NoImagesAfter, err = strconv.Atoi(request.FormValue("noimagesafter"))
					if err != nil {
						board.NoImagesAfter = 0
					}

					board.MaxMessageLength, err = strconv.Atoi(request.FormValue("maxmessagelength"))
					if err != nil {
						board.MaxMessageLength = 1024 * 8
					}

					board.EmbedsAllowed = (request.FormValue("embedsallowed") == "on")
					board.RedirectToThread = (request.FormValue("redirecttothread") == "on")
					board.RequireFile = (request.FormValue("require_file") == "on")
					board.EnableCatalog = (request.FormValue("enablecatalog") == "on")

					//actually start generating stuff

					if err = os.Mkdir(path.Join(systemCritical.DocumentRoot, board.Dir), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/ already exists.",
							systemCritical.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(systemCritical.DocumentRoot, board.Dir, "res"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/res/ already exists.",
							systemCritical.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(systemCritical.DocumentRoot, board.Dir, "src"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/src/ already exists.",
							systemCritical.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(systemCritical.DocumentRoot, board.Dir, "thumb"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/thumb/ already exists.",
							systemCritical.DocumentRoot, board.Dir)
						break
					}

					if err = gcsql.CreateBoard(board); err != nil {
						do = ""
						boardCreationStatus = gclog.Print(gclog.LErrorLog, "Error creating board: ", err.Error())
						break
					}
					boardCreationStatus = "Board created successfully"
					building.BuildBoards(false)
					gcsql.ResetBoardSectionArrays()
					gclog.Print(gclog.LStaffLog, "Boards rebuilt successfully")
					done = true
				case do == "del":
					// resetBoardSectionArrays()
				case do == "edit":
					// resetBoardSectionArrays()
				default:
					boardConfig := config.GetBoardConfig("")
					// put the default column values in the text boxes
					board.Section = 1
					board.MaxFilesize = 4718592
					board.MaxPages = 11
					board.DefaultStyle = "pipes.css"
					board.Anonymous = "Anonymous"
					board.AutosageAfter = 200
					board.MaxMessageLength = 8192
					board.EmbedsAllowed = true
					board.EnableCatalog = true
					board.Worksafe = true
					board.ThreadsPerPage = boardConfig.ThreadsPerPage
				}

				output = `<h1 class="manage-header">Manage boards</h1><form action="/manage?action=boards" method="POST"><input type="hidden" name="do" value="existing" /><select name="boardselect"><option>Select board...</option>`
				var boards []string
				boards, err = gcsql.GetBoardUris()
				if err != nil {
					err = errors.New(gclog.Print(gclog.LErrorLog,
						"Error getting board list: ", err.Error()))
					return "", err
				}
				for _, boardDir := range boards {
					outputStr += "<option>" + boardDir + "</option>"
				}

				outputStr += `</select><input type="submit" value="Edit" /><input type="submit" value="Delete" /></form><hr />` +
					`<h2 class="manage-header">Create new board</h2><span id="board-creation-message">` + boardCreationStatus + `</span><br />`

				manageBoardsBuffer := bytes.NewBufferString("")
				gcsql.AllSections, _ = gcsql.GetAllSectionsOrCreateDefault()

				boardConfig := config.GetBoardConfig("")

				if err = serverutil.MinifyTemplate(gctemplates.ManageStaff,
					map[string]interface{}{
						"boardConfig": boardConfig,
						"board":       board,
						"section_arr": gcsql.AllSections,
					},
					manageBoardsBuffer, "text/html"); err != nil {
					return "", errors.New(gclog.Print(gclog.LErrorLog,
						"Error executing board management page template: ", err.Error()))
				}
				return outputStr + manageBoardsBuffer.String(), nil
			}
			gcsql.ResetBoardSectionArrays()
			return
		}},
	"actions": {
		Title:       "Staff actions",
		Permissions: JanitorPerms,
		isJSON:      true,
	},
	"rebuildfront": {
		Title:       "Rebuild front page",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			if err = gctemplates.InitTemplates(); err != nil {
				return "", err
			}
			return "Built front page successfully", building.BuildFrontPage()
		}},
	"rebuildall": {
		Title:       "Rebuild everything",
		Permissions: AdminPerms,
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
			buildStr := ""
			for _, msg := range buildMap {
				buildStr += fmt.Sprintln(msg, "<hr />")
			}
			return buildStr, nil
		}},
	"rebuildboard": {
		Title:       "Rebuild board",
		Permissions: AdminPerms,
		isJSON:      true,
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
	"rebuildboards": {
		Title:       "Rebuild boards",
		Permissions: AdminPerms,
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
			return "Boards built successfully", building.BuildBoards(false)
		}},
	"reparsehtml": {
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
	"recentposts": {
		Title:       "Recent posts",
		Permissions: JanitorPerms,
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
	"postinfo": {
		Title:       "Post info",
		Permissions: ModPerms,
		isJSON:      true,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			var post gcsql.Post
			post, err = gcsql.GetSpecificPost(gcutil.HackyStringToInt(request.FormValue("postid")), false)
			if err != nil {
				output, _ = gcutil.MarshalJSON(err, false)
				return output, nil
			}
			jsonStr, _ := gcutil.MarshalJSON(post, false)
			return jsonStr, nil
		}},
	"staff": {
		Title:       "Staff",
		Permissions: AdminPerms,
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
	"tempposts": {
		Title:       "Temporary posts lists",
		Permissions: AdminPerms,
		Callback: func(writer http.ResponseWriter, request *http.Request, wantsJSON bool) (output interface{}, err error) {
			outputStr := `<h1 class="manage-header">Temporary posts</h1>`
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
