package manage

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io/ioutil"
	"net"
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

var (
	chopPortNumRegex = regexp.MustCompile(`(.+|\w+):(\d+)$`)
)

// ManageFunction represents the functions accessed by staff members at /manage?action=<functionname>.
type ManageFunction struct {
	Title       string
	Permissions int                                                            // 0 -> non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	Callback    func(writer http.ResponseWriter, request *http.Request) string `json:"-"` //return string of html output
}

var manageFunctions = map[string]ManageFunction{
	"cleanup": {
		Title:       "Cleanup",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = `<h2 class="manage-header">Cleanup</h2><br />`
			var err error
			if request.FormValue("run") == "Run Cleanup" {
				html += "Removing deleted posts from the database.<hr />"
				if err = gcsql.PermanentlyRemoveDeletedPosts(); err != nil {
					return html + "<tr><td>" +
						gclog.Print(gclog.LErrorLog, "Error removing deleted posts from database: ", err.Error()) +
						"</td></tr></table>"
				}
				// TODO: remove orphaned replies and uploads

				html += "Optimizing all tables in database.<hr />"
				err = gcsql.OptimizeDatabase()
				if err != nil {
					return html + "<tr><td>" +
						gclog.Print(gclog.LErrorLog, "Error optimizing SQL tables: ", err.Error()) +
						"</td></tr></table>"
				}

				html += "Cleanup finished"
			} else {
				html += `<form action="/manage?action=cleanup" method="post">` +
					`<input name="run" id="run" type="submit" value="Run Cleanup" />` +
					`</form>`
			}
			return
		}},
	"config": {
		Title:       "Configuration",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			var status string
			if do == "save" {
				configJSON, err := json.MarshalIndent(config.Config, "", "\t")
				if err != nil {
					status += gclog.Println(gclog.LErrorLog, err.Error()) + "<br />"
				} else if err = ioutil.WriteFile("gochan.json", configJSON, 0777); err != nil {
					status += gclog.Println(gclog.LErrorLog,
						"Error backing up old gochan.json, cancelling save:", err.Error())
				} else {
					config.Config.Lockdown = (request.PostFormValue("Lockdown") == "on")
					config.Config.LockdownMessage = request.PostFormValue("LockdownMessage")
					SillytagsArr := strings.Split(request.PostFormValue("Sillytags"), "\n")
					var Sillytags []string
					for _, tag := range SillytagsArr {
						Sillytags = append(Sillytags, strings.Trim(tag, " \n\r"))
					}

					config.Config.Sillytags = Sillytags
					config.Config.UseSillytags = (request.PostFormValue("UseSillytags") == "on")
					config.Config.Modboard = request.PostFormValue("Modboard")
					config.Config.SiteName = request.PostFormValue("SiteName")
					config.Config.SiteSlogan = request.PostFormValue("SiteSlogan")
					config.Config.SiteHeaderURL = request.PostFormValue("SiteHeaderURL")
					config.Config.SiteWebfolder = request.PostFormValue("SiteWebfolder")
					// TODO: Change this to match the new Style type in gochan.json
					/* Styles_arr := strings.Split(request.PostFormValue("Styles"), "\n")
					var Styles []string
					for _, style := range Styles_arr {
						Styles = append(Styles, strings.Trim(style, " \n\r"))
					}
					config.Styles = Styles */
					config.Config.DefaultStyle = request.PostFormValue("DefaultStyle")
					config.Config.AllowDuplicateImages = (request.PostFormValue("AllowDuplicateImages") == "on")
					config.Config.AllowVideoUploads = (request.PostFormValue("AllowVideoUploads") == "on")
					NewThreadDelay, err := strconv.Atoi(request.PostFormValue("NewThreadDelay"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.NewThreadDelay = NewThreadDelay
					}

					ReplyDelay, err := strconv.Atoi(request.PostFormValue("ReplyDelay"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ReplyDelay = ReplyDelay
					}

					MaxLineLength, err := strconv.Atoi(request.PostFormValue("MaxLineLength"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.MaxLineLength = MaxLineLength
					}

					ReservedTripsArr := strings.Split(request.PostFormValue("ReservedTrips"), "\n")
					var ReservedTrips []string
					for _, trip := range ReservedTripsArr {
						ReservedTrips = append(ReservedTrips, strings.Trim(trip, " \n\r"))

					}
					config.Config.ReservedTrips = ReservedTrips

					ThumbWidth, err := strconv.Atoi(request.PostFormValue("ThumbWidth"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbWidth = ThumbWidth
					}

					ThumbHeight, err := strconv.Atoi(request.PostFormValue("ThumbHeight"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbHeight = ThumbHeight
					}

					ThumbWidthReply, err := strconv.Atoi(request.PostFormValue("ThumbWidthReply"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbWidthReply = ThumbWidthReply
					}

					ThumbHeightReply, err := strconv.Atoi(request.PostFormValue("ThumbHeightReply"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbHeightReply = ThumbHeightReply
					}

					ThumbWidthCatalog, err := strconv.Atoi(request.PostFormValue("ThumbWidthCatalog"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbWidthCatalog = ThumbWidthCatalog
					}

					ThumbHeightCatalog, err := strconv.Atoi(request.PostFormValue("ThumbHeightCatalog"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.ThumbHeightCatalog = ThumbHeightCatalog
					}

					RepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("RepliesOnBoardPage"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.RepliesOnBoardPage = RepliesOnBoardPage
					}

					StickyRepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("StickyRepliesOnBoardPage"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.StickyRepliesOnBoardPage = StickyRepliesOnBoardPage
					}

					BanColorsArr := strings.Split(request.PostFormValue("BanColors"), "\n")
					var BanColors []string
					for _, color := range BanColorsArr {
						BanColors = append(BanColors, strings.Trim(color, " \n\r"))

					}
					config.Config.BanColors = BanColors

					config.Config.BanMsg = request.PostFormValue("BanMsg")
					EmbedWidth, err := strconv.Atoi(request.PostFormValue("EmbedWidth"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.EmbedWidth = EmbedWidth
					}

					EmbedHeight, err := strconv.Atoi(request.PostFormValue("EmbedHeight"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.EmbedHeight = EmbedHeight
					}

					config.Config.ExpandButton = (request.PostFormValue("ExpandButton") == "on")
					config.Config.ImagesOpenNewTab = (request.PostFormValue("ImagesOpenNewTab") == "on")
					config.Config.MakeURLsHyperlinked = (request.PostFormValue("MakeURLsHyperlinked") == "on")
					config.Config.NewTabOnOutlinks = (request.PostFormValue("NewTabOnOutlinks") == "on")
					config.Config.MinifyHTML = (request.PostFormValue("MinifyHTML") == "on")
					config.Config.MinifyJS = (request.PostFormValue("MinifyJS") == "on")
					config.Config.DateTimeFormat = request.PostFormValue("DateTimeFormat")
					AkismetAPIKey := request.PostFormValue("AkismetAPIKey")

					if err = serverutil.CheckAkismetAPIKey(AkismetAPIKey); err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.AkismetAPIKey = AkismetAPIKey
					}

					config.Config.UseCaptcha = (request.PostFormValue("UseCaptcha") == "on")
					CaptchaWidth, err := strconv.Atoi(request.PostFormValue("CaptchaWidth"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.CaptchaWidth = CaptchaWidth
					}
					CaptchaHeight, err := strconv.Atoi(request.PostFormValue("CaptchaHeight"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.CaptchaHeight = CaptchaHeight
					}

					config.Config.EnableGeoIP = (request.PostFormValue("EnableGeoIP") == "on")
					config.Config.GeoIPDBlocation = request.PostFormValue("GeoIPDBlocation")

					MaxRecentPosts, err := strconv.Atoi(request.PostFormValue("MaxRecentPosts"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.MaxRecentPosts = MaxRecentPosts
					}

					config.Config.EnableAppeals = (request.PostFormValue("EnableAppeals") == "on")
					MaxLogDays, err := strconv.Atoi(request.PostFormValue("MaxLogDays"))
					if err != nil {
						status += err.Error() + "<br />"
					} else {
						config.Config.MaxLogDays = MaxLogDays
					}

					configJSON, err = json.MarshalIndent(config.Config, "", "\t")
					if err != nil {
						status += err.Error() + "<br />"
					} else if err = ioutil.WriteFile("gochan.json", configJSON, 0777); err != nil {
						status = gclog.Print(gclog.LErrorLog, "Error writing gochan.json: ", err.Error())
					} else {
						status = "Wrote gochan.json successfully<br />"
						building.BuildJS()
					}
				}
			}
			manageConfigBuffer := bytes.NewBufferString("")
			if err := gctemplates.ManageConfig.Execute(manageConfigBuffer,
				map[string]interface{}{"config": config.Config, "status": status},
			); err != nil {
				return html + gclog.Print(gclog.LErrorLog, "Error executing config management page: ", err.Error())
			}
			html += manageConfigBuffer.String()
			return
		}},
	"login": {
		Title:       "Login",
		Permissions: 0,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			if GetStaffRank(request) > 0 {
				http.Redirect(writer, request, path.Join(config.Config.SiteWebfolder, "manage"), http.StatusFound)
			}
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirectAction := request.FormValue("action")
			if redirectAction == "" {
				redirectAction = "announcements"
			}
			if username == "" || password == "" {
				//assume that they haven't logged in
				html = `<form method="POST" action="` + config.Config.SiteWebfolder + `manage?action=login" id="login-box" class="staff-form">` +
					`<input type="hidden" name="redirect" value="` + redirectAction + `" />` +
					`<input type="text" name="username" class="logindata" /><br />` +
					`<input type="password" name="password" class="logindata" /><br />` +
					`<input type="submit" value="Login" />` +
					`</form>`
			} else {
				key := gcutil.Md5Sum(request.RemoteAddr + username + password + config.Config.RandomSeed + gcutil.RandomString(3))[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(config.Config.SiteWebfolder, "manage?action="+request.FormValue("redirect")), http.StatusFound)
			}
			return
		}},
	"logout": {
		Title:       "Logout",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			cookie, _ := request.Cookie("sessiondata")
			cookie.MaxAge = 0
			cookie.Expires = time.Now().Add(-7 * 24 * time.Hour)
			http.SetCookie(writer, cookie)
			return "Logged out successfully"
		}},
	"announcements": {
		Title:       "Announcements",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = `<h1 class="manage-header">Announcements</h1><br />`

			//get all announcements to announcement list
			//loop to html if exist, no announcement if empty
			announcements, err := gcsql.GetAllAccouncements()
			if err != nil {
				return html + gclog.Print(gclog.LErrorLog, "Error getting announcements: ", err.Error())
			}
			if len(announcements) == 0 {
				html += "No announcements"
			} else {
				for _, announcement := range announcements {
					html += `<div class="section-block">` +
						`<div class="section-title-block"><b>` + announcement.Subject + `</b> by ` + announcement.Poster + ` at ` + announcement.Timestamp.Format(config.Config.DateTimeFormat) + `</div>` +
						`<div class="section-body">` + announcement.Message + `</div></div>`
				}
			}
			return html
		}},
	"bans": {
		Title:       "Bans",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (pageHTML string) { //TODO whatever this does idk man
			var post gcsql.Post
			if request.FormValue("do") == "add" {
				ip := net.ParseIP(request.FormValue("ip"))
				name := request.FormValue("name")
				nameIsRegex := (request.FormValue("nameregex") == "on")
				checksum := request.FormValue("checksum")
				filename := request.FormValue("filename")
				durationForm := request.FormValue("duration")
				permaban := (durationForm == "" || durationForm == "0" || durationForm == "forever")
				duration, err := gcutil.ParseDurationString(durationForm)
				if err != nil {
					serverutil.ServeErrorPage(writer, err.Error())
				}
				expires := time.Now().Add(duration)

				boards := request.FormValue("boards")
				reason := html.EscapeString(request.FormValue("reason"))
				staffNote := html.EscapeString(request.FormValue("staffnote"))
				currentStaff, _ := getCurrentStaff(request)

				err = nil
				if filename != "" {
					err = gcsql.FileNameBan(filename, nameIsRegex, currentStaff, expires, permaban, staffNote, boards)
				}
				if err != nil {
					pageHTML += err.Error()
					err = nil
				}
				if name != "" {
					err = gcsql.UserNameBan(name, nameIsRegex, currentStaff, expires, permaban, staffNote, boards)
				}
				if err != nil {
					pageHTML += err.Error()
					err = nil
				}

				if request.FormValue("fullban") == "on" {
					err = gcsql.UserBan(ip, false, currentStaff, boards, expires, permaban, staffNote, reason, true, time.Now())
					if err != nil {
						pageHTML += err.Error()
						err = nil
					}
				} else {
					if request.FormValue("threadban") == "on" {
						err = gcsql.UserBan(ip, true, currentStaff, boards, expires, permaban, staffNote, reason, true, time.Now())
						if err != nil {
							pageHTML += err.Error()
							err = nil
						}
					}
					if request.FormValue("imageban") == "on" {
						err = gcsql.FileBan(checksum, currentStaff, expires, permaban, staffNote, boards)
						if err != nil {
							pageHTML += err.Error()
							err = nil
						}
					}
				}
			}

			if request.FormValue("postid") != "" {
				var err error
				post, err = gcsql.GetSpecificPostByString(request.FormValue("postid"))
				if err != nil {
					return pageHTML + gclog.Print(gclog.LErrorLog, "Error getting post: ", err.Error())
				}
			}

			banlist, err := gcsql.GetAllBans()
			if err != nil {
				return pageHTML + gclog.Print(gclog.LErrorLog, "Error getting ban list: ", err.Error())
			}
			manageBansBuffer := bytes.NewBufferString("")

			if err := gctemplates.ManageBans.Execute(manageBansBuffer,
				map[string]interface{}{"config": config.Config, "banlist": banlist, "post": post},
			); err != nil {
				return pageHTML + gclog.Print(gclog.LErrorLog, "Error executing ban management page template: ", err.Error())
			}
			pageHTML += manageBansBuffer.String()
			return
		}},
	"getstaffjquery": {
		Permissions: 0,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			staff, err := getCurrentFullStaff(request)
			if err != nil {
				html = "nobody;0;"
				return
			}
			html = staff.Username + ";" + strconv.Itoa(staff.Rank) + ";" + staff.Boards
			return
		}},
	"boards": {
		Title:       "Boards",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			var done bool
			board := new(gcsql.Board)
			var boardCreationStatus string
			var err error
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
					board.ListOrder, err = strconv.Atoi(orderStr)
					if err != nil {
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
					if err = os.Mkdir(path.Join(config.Config.DocumentRoot, board.Dir), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/ already exists.",
							config.Config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.Config.DocumentRoot, board.Dir, "res"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/res/ already exists.",
							config.Config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.Config.DocumentRoot, board.Dir, "src"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/src/ already exists.",
							config.Config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.Config.DocumentRoot, board.Dir, "thumb"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(gclog.LStaffLog|gclog.LErrorLog, "Directory %s/%s/thumb/ already exists.",
							config.Config.DocumentRoot, board.Dir)
						break
					}

					if err := gcsql.CreateBoard(board); err != nil {
						do = ""
						boardCreationStatus = gclog.Print(gclog.LErrorLog, "Error creating board: ", err.Error())
						break
					} else {
						boardCreationStatus = "Board created successfully"
						building.BuildBoards()
						gcsql.ResetBoardSectionArrays()
						gclog.Print(gclog.LStaffLog, "Boards rebuilt successfully")
						done = true
					}
				case do == "del":
					// resetBoardSectionArrays()
				case do == "edit":
					// resetBoardSectionArrays()
				default:
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
					board.ThreadsPerPage = config.Config.ThreadsPerPage
				}

				html = `<h1 class="manage-header">Manage boards</h1><form action="/manage?action=boards" method="POST"><input type="hidden" name="do" value="existing" /><select name="boardselect"><option>Select board...</option>`
				boards, err := gcsql.GetBoardUris()
				if err != nil {
					return html + gclog.Print(gclog.LErrorLog, "Error getting board list: ", err.Error())
				}
				for _, boardDir := range boards {
					html += "<option>" + boardDir + "</option>"
				}

				html += `</select><input type="submit" value="Edit" /><input type="submit" value="Delete" /></form><hr />` +
					`<h2 class="manage-header">Create new board</h2><span id="board-creation-message">` + boardCreationStatus + `</span><br />`

				manageBoardsBuffer := bytes.NewBufferString("")
				gcsql.AllSections, _ = gcsql.GetAllSectionsOrCreateDefault()

				if err := gctemplates.ManageBoards.Execute(manageBoardsBuffer, map[string]interface{}{
					"config":      config.Config,
					"board":       board,
					"section_arr": gcsql.AllSections,
				}); err != nil {
					return html + gclog.Print(gclog.LErrorLog,
						"Error executing board management page template: ", err.Error())
				}
				html += manageBoardsBuffer.String()
				return
			}
			gcsql.ResetBoardSectionArrays()
			return
		}},
	"staffmenu": {
		Title:       "Staff menu",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			rank := GetStaffRank(request)

			html = `<a href="javascript:void(0)" id="logout" class="staffmenu-item">Log out</a><br />` +
				`<a href="javascript:void(0)" id="announcements" class="staffmenu-item">Announcements</a><br />`
			if rank == 3 {
				html += `<b>Admin stuff</b><br /><a href="javascript:void(0)" id="staff" class="staffmenu-item">Manage staff</a><br />` +
					//`<a href="javascript:void(0)" id="purgeeverything" class="staffmenu-item">Purge everything!</a><br />` +
					`<a href="javascript:void(0)" id="executesql" class="staffmenu-item">Execute SQL statement(s)</a><br />` +
					`<a href="javascript:void(0)" id="cleanup" class="staffmenu-item">Run cleanup</a><br />` +
					`<a href="javascript:void(0)" id="rebuildall" class="staffmenu-item">Rebuild all</a><br />` +
					`<a href="javascript:void(0)" id="rebuildfront" class="staffmenu-item">Rebuild front page</a><br />` +
					`<a href="javascript:void(0)" id="rebuildboards" class="staffmenu-item">Rebuild board pages</a><br />` +
					`<a href="javascript:void(0)" id="reparsehtml" class="staffmenu-item">Reparse all posts</a><br />` +
					`<a href="javascript:void(0)" id="boards" class="staffmenu-item">Add/edit/delete boards</a><br />`
			}
			if rank >= 2 {
				html += `<b>Mod stuff</b><br />` +
					`<a href="javascript:void(0)" id="bans" class="staffmenu-item">Ban User(s)</a><br />`
			}

			if rank >= 1 {
				html += `<a href="javascript:void(0)" id="recentimages" class="staffmenu-item">Recently uploaded images</a><br />` +
					`<a href="javascript:void(0)" id="recentposts" class="staffmenu-item">Recent posts</a><br />` +
					`<a href="javascript:void(0)" id="searchip" class="staffmenu-item">Search posts by IP</a><br />`
			}
			return
		}},
	"rebuildfront": {
		Title:       "Rebuild front page",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			gctemplates.InitTemplates()
			return building.BuildFrontPage()
		}},
	"rebuildall": {
		Title:       "Rebuild everything",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			gctemplates.InitTemplates()
			gcsql.ResetBoardSectionArrays()
			return building.BuildFrontPage() + "<hr />" +
				building.BuildBoardListJSON() + "<hr />" +
				building.BuildBoards() + "<hr />" +
				building.BuildJS() + "<hr />"
		}},
	"rebuildboards": {
		Title:       "Rebuild boards",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			gctemplates.InitTemplates()
			return building.BuildBoards()
		}},
	"reparsehtml": {
		Title:       "Reparse HTML",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			messages, err := gcsql.GetAllNondeletedMessageRaw()
			if err != nil {
				html += err.Error() + "<br />"
				return
			}

			for _, message := range messages {
				message.Message = posting.FormatMessage(message.MessageRaw)
			}
			err = gcsql.SetFormattedInDatabase(messages)

			if err != nil {
				return html + gclog.Printf(gclog.LErrorLog, err.Error())
			}
			html += "Done reparsing HTML<hr />" +
				building.BuildFrontPage() + "<hr />" +
				building.BuildBoardListJSON() + "<hr />" +
				building.BuildBoards() + "<hr />"
			return
		}},
	"recentposts": {
		Title:       "Recent posts",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			limit := request.FormValue("limit")
			if limit == "" {
				limit = "50"
			}
			html = `<h1 class="manage-header">Recent posts</h1>` +
				`Limit by: <select id="limit">` +
				`<option>25</option><option>50</option><option>100</option><option>200</option>` +
				`</select><br /><table width="100%%d" border="1">` +
				`<colgroup><col width="25%%" /><col width="50%%" /><col width="17%%" /></colgroup>` +
				`<tr><th></th><th>Message</th><th>Time</th></tr>`
			recentposts, err := gcsql.GetRecentPostsGlobal(gcutil.HackyStringToInt(limit), false) //only uses boardname, boardid, postid, parentid, message, ip and timestamp

			if err != nil {
				return html + "<tr><td>" + gclog.Print(gclog.LErrorLog, "Error getting recent posts: ",
					err.Error()) + "</td></tr></table>"
			}

			for _, recentpost := range recentposts {
				html += fmt.Sprintf(
					`<tr><td><b>Post:</b> <a href="%s">%s/%d</a><br /><b>IP:</b> %s</td><td>%s</td><td>%s</td></tr>`,
					path.Join(config.Config.SiteWebfolder, recentpost.BoardName, "/res/", strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID)),
					recentpost.BoardName, recentpost.PostID, recentpost.IP, recentpost.Message,
					recentpost.Timestamp.Format("01/02/06, 15:04"),
				)
			}
			html += "</table>"
			return
		}},
	"postinfo": {
		Title:       "Post info",
		Permissions: 2,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			errMap := map[string]interface{}{
				"action":  "postInfo",
				"success": false,
			}
			post, err := gcsql.GetSpecificPost(gcutil.HackyStringToInt(request.FormValue("postid")), false)
			if err != nil {
				errMap["message"] = err.Error()
				jsonErr, _ := gcutil.MarshalJSON(errMap, false)
				return jsonErr
			}
			jsonStr, _ := gcutil.MarshalJSON(post, false)
			return jsonStr
		}},
	"staff": {
		Title:       "Staff",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			html = `<h1 class="manage-header">Staff</h1><br />` +
				`<table id="stafftable" border="1">` +
				"<tr><td><b>Username</b></td><td><b>Rank</b></td><td><b>Boards</b></td><td><b>Added on</b></td><td><b>Action</b></td></tr>"
			allStaff, err := gcsql.GetAllStaffNopass()
			if err != nil {
				return html + gclog.Print(gclog.LErrorLog, "Error getting staff list: ", err.Error())
			}

			for _, staff := range allStaff {
				username := request.FormValue("username")
				password := request.FormValue("password")
				rank := request.FormValue("rank")
				rankI, _ := strconv.Atoi(rank)
				if do == "add" {
					if err := gcsql.NewStaff(username, password, rankI); err != nil {
						serverutil.ServeErrorPage(writer, gclog.Printf(gclog.LErrorLog,
							"Error creating new staff account %q: %s", username, err.Error()))
						return
					}
				} else if do == "del" && username != "" {
					if err = gcsql.DeleteStaff(request.FormValue("username")); err != nil {
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
				html += fmt.Sprintf(
					`<tr><td>%s</td><td>%s</td><td>%s</td><td>%s</td><td><a href="/manage?action=staff&amp;do=del&amp;username=%s" style="float:right;color:red;">X</a></td></tr>`,
					staff.Username, rank, staff.Boards, staff.AddedOn.Format(config.Config.DateTimeFormat), staff.Username)

			}
			html += `</table><hr /><h2 class="manage-header">Add new staff</h2>` +
				`<form action="/manage?action=staff" onsubmit="return makeNewStaff();" method="POST">` +
				`<input type="hidden" name="do" value="add" />` +
				`Username: <input id="username" name="username" type="text" /><br />` +
				`Password: <input id="password" name="password" type="password" /><br />` +
				`Rank: <select id="rank" name="rank">` +
				`<option value="3">Admin</option>` +
				`<option value="2">Moderator</option>` +
				`<option value="1">Janitor</option>` +
				`</select><br />` +
				`<input id="submitnewstaff" type="submit" value="Add" />` +
				`</form>`
			return
		}},
	"tempposts": {
		Title:       "Temporary posts lists",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html += `<h1 class="manage-header">Temporary posts</h1>`
			if len(gcsql.TempPosts) == 0 {
				html += "No temporary posts<br />"
				return
			}
			for p, post := range gcsql.TempPosts {
				html += fmt.Sprintf("Post[%d]: %#v<br />", p, post)
			}
			return
		}},
}
