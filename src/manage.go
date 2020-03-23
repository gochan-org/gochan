package main

import (
	"bytes"
	"database/sql"
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

	"golang.org/x/crypto/bcrypt"
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

func getManageFunctionsJSON() string {
	var jsonStr string

	return jsonStr
}

func callManageFunction(writer http.ResponseWriter, request *http.Request) {
	var err error
	if err = request.ParseForm(); err != nil {
		serveErrorPage(writer,
			gclog.Print(lErrorLog, "Error parsing form data: ", err.Error()))
	}

	action := request.FormValue("action")
	staffRank := getStaffRank(request)
	var managePageBuffer bytes.Buffer
	if action == "" {
		action = "announcements"
	} else if action == "postinfo" {
		writer.Header().Add("Content-Type", "application/json")
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	}

	if action != "getstaffjquery" && action != "postinfo" {
		managePageBuffer.WriteString("<!DOCTYPE html><html><head>")
		if err = manageHeaderTmpl.Execute(&managePageBuffer, config); err != nil {
			serveErrorPage(writer, gclog.Print(lErrorLog|lStaffLog,
				"Error executing manage page header template: ", err.Error()))
			return
		}
	}

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
	if action != "getstaffjquery" && action != "postinfo" {
		managePageBuffer.Write([]byte("</body></html>"))
	}

	writer.Write(managePageBuffer.Bytes())
}

func getCurrentStaff(request *http.Request) (string, error) {
	sessionCookie, err := request.Cookie("sessiondata")
	if err != nil {
		return "", nil
	}
	key := sessionCookie.Value
	currentSession := new(LoginSession)
	if err := queryRowSQL("SELECT sessiondata FROM DBPREFIXsessions WHERE name = ?",
		[]interface{}{key},
		[]interface{}{&currentSession.Data},
	); err != nil {
		return "", err
	}
	return currentSession.Data, nil
}

func getStaff(name string) (*Staff, error) {
	staff := new(Staff)
	err := queryRowSQL("SELECT * FROM DBPREFIXstaff WHERE username = ?",
		[]interface{}{name},
		[]interface{}{&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.Boards, &staff.AddedOn, &staff.LastActive},
	)
	return staff, err
}

func getStaffRank(request *http.Request) int {
	staffname, err := getCurrentStaff(request)
	if err != nil {
		gclog.Print(lErrorLog, "Error getting current staff: ", err.Error())
		return 0
	}
	if staffname == "" {
		return 0
	}
	staff, err := getStaff(staffname)
	if err != nil {
		gclog.Print(lErrorLog, "Error getting current staff: ", err.Error())
		return 0
	}
	return staff.Rank
}

func newStaff(username string, password string, rank int) error {
	_, err := execSQL("INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES(?,?,?)",
		&username, bcryptSum(password), &rank)
	return err
}

func deleteStaff(username string) error {
	_, err := execSQL("DELETE FROM DBPREFIXstaff WHERE username = ?", username)
	return err
}

func createSession(key string, username string, password string, request *http.Request, writer http.ResponseWriter) int {
	//returns 0 for successful, 1 for password mismatch, and 2 for other
	domain := request.Host
	var err error
	domain = chopPortNumRegex.Split(domain, -1)[0]

	if !validReferrer(request) {
		gclog.Print(lStaffLog, "Rejected login from possible spambot @ "+request.RemoteAddr)
		return 2
	}
	staff, err := getStaff(username)
	if err != nil {
		gclog.Print(lErrorLog, err.Error())
		return 1
	}

	success := bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
	if success == bcrypt.ErrMismatchedHashAndPassword {
		// password mismatch
		gclog.Print(lStaffLog, "Failed login (password mismatch) from "+request.RemoteAddr+" at "+getSQLDateTime())
		return 1
	}

	// successful login, add cookie that expires in one month
	http.SetCookie(writer, &http.Cookie{
		Name:   "sessiondata",
		Value:  key,
		Path:   "/",
		Domain: domain,
		MaxAge: 60 * 60 * 24 * 7,
	})

	if _, err = execSQL("INSERT INTO DBPREFIXsessions (name,sessiondata,expires) VALUES(?,?,?)",
		key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
	); err != nil {
		gclog.Print(lErrorLog, "Error creating new staff session: ", err.Error())
		return 2
	}

	if _, err = execSQL("UPDATE DBPREFIXstaff SET last_active = ? WHERE username = ?",
		getSQLDateTime(), username,
	); err != nil {
		gclog.Print(lErrorLog, "Error creating new staff session: ", err.Error())
		return 2
	}
	return 0
}

var manageFunctions = map[string]ManageFunction{
	"cleanup": {
		Title:       "Cleanup",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = "<h2 class=\"manage-header\">Cleanup</h2><br />"
			var err error
			if request.FormValue("run") == "Run Cleanup" {
				html += "Removing deleted posts from the database.<hr />"
				if _, err = execSQL(
					"DELETE FROM DBPREFIXposts WHERE deleted_timestamp = ?", nilTimestamp,
				); err != nil {
					return html + "<tr><td>" +
						gclog.Print(lErrorLog, "Error removing deleted posts from database: ", err.Error()) +
						"</td></tr></table>"
				}
				// TODO: remove orphaned replies and uploads

				html += "Optimizing all tables in database.<hr />"
				tableRows, tablesErr := querySQL("SHOW TABLES")
				defer tableRows.Close()

				if tablesErr != nil && tablesErr != sql.ErrNoRows {
					return html + "<tr><td>" +
						gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
						"</td></tr></table>"
				}
				for tableRows.Next() {
					var table string
					tableRows.Scan(&table)
					if _, err := execSQL("OPTIMIZE TABLE " + table); err != nil {
						return html + "<tr><td>" +
							gclog.Print(lErrorLog, "Error optimizing SQL tables: ", tablesErr.Error()) +
							"</td></tr></table>"
					}
				}

				html += "Cleanup finished"
			} else {
				html += "<form action=\"/manage?action=cleanup\" method=\"post\">\n" +
					"	<input name=\"run\" id=\"run\" type=\"submit\" value=\"Run Cleanup\" />\n" +
					"</form>"
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
				configJSON, err := json.MarshalIndent(config, "", "\t")
				if err != nil {
					status += err.Error() + "<br />\n"
				} else if err = ioutil.WriteFile("gochan.json", configJSON, 0777); err != nil {
					status += "Error backing up old gochan.json, cancelling save: " + err.Error() + "\n"
				} else {
					config.Lockdown = (request.PostFormValue("Lockdown") == "on")
					config.LockdownMessage = request.PostFormValue("LockdownMessage")
					Sillytags_arr := strings.Split(request.PostFormValue("Sillytags"), "\n")
					var Sillytags []string
					for _, tag := range Sillytags_arr {
						Sillytags = append(Sillytags, strings.Trim(tag, " \n\r"))
					}
					config.Sillytags = Sillytags
					config.UseSillytags = (request.PostFormValue("UseSillytags") == "on")
					config.Modboard = request.PostFormValue("Modboard")
					config.SiteName = request.PostFormValue("SiteName")
					config.SiteSlogan = request.PostFormValue("SiteSlogan")
					config.SiteHeaderURL = request.PostFormValue("SiteHeaderURL")
					config.SiteWebfolder = request.PostFormValue("SiteWebfolder")
					// TODO: Change this to match the new Style type in gochan.json
					/* Styles_arr := strings.Split(request.PostFormValue("Styles"), "\n")
					var Styles []string
					for _, style := range Styles_arr {
						Styles = append(Styles, strings.Trim(style, " \n\r"))
					}
					config.Styles = Styles */
					config.DefaultStyle = request.PostFormValue("DefaultStyle")
					config.AllowDuplicateImages = (request.PostFormValue("AllowDuplicateImages") == "on")
					config.AllowVideoUploads = (request.PostFormValue("AllowVideoUploads") == "on")
					NewThreadDelay, err := strconv.Atoi(request.PostFormValue("NewThreadDelay"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.NewThreadDelay = NewThreadDelay
					}

					ReplyDelay, err := strconv.Atoi(request.PostFormValue("ReplyDelay"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ReplyDelay = ReplyDelay
					}

					MaxLineLength, err := strconv.Atoi(request.PostFormValue("MaxLineLength"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.MaxLineLength = MaxLineLength
					}

					ReservedTrips_arr := strings.Split(request.PostFormValue("ReservedTrips"), "\n")
					var ReservedTrips []string
					for _, trip := range ReservedTrips_arr {
						ReservedTrips = append(ReservedTrips, strings.Trim(trip, " \n\r"))

					}
					config.ReservedTrips = ReservedTrips

					ThumbWidth, err := strconv.Atoi(request.PostFormValue("ThumbWidth"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbWidth = ThumbWidth
					}

					ThumbHeight, err := strconv.Atoi(request.PostFormValue("ThumbHeight"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbHeight = ThumbHeight
					}

					ThumbWidth_reply, err := strconv.Atoi(request.PostFormValue("ThumbWidth_reply"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbWidth_reply = ThumbWidth_reply
					}

					ThumbHeight_reply, err := strconv.Atoi(request.PostFormValue("ThumbHeight_reply"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbHeight_reply = ThumbHeight_reply
					}

					ThumbWidth_catalog, err := strconv.Atoi(request.PostFormValue("ThumbWidth_catalog"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbWidth_catalog = ThumbWidth_catalog
					}

					ThumbHeight_catalog, err := strconv.Atoi(request.PostFormValue("ThumbHeight_catalog"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.ThumbHeight_catalog = ThumbHeight_catalog
					}

					PostsPerThreadPage, err := strconv.Atoi(request.PostFormValue("PostsPerThreadPage"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.PostsPerThreadPage = PostsPerThreadPage
					}

					RepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("RepliesOnBoardPage"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.RepliesOnBoardPage = RepliesOnBoardPage
					}

					StickyRepliesOnBoardPage, err := strconv.Atoi(request.PostFormValue("StickyRepliesOnBoardPage"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.StickyRepliesOnBoardPage = StickyRepliesOnBoardPage
					}

					BanColors_arr := strings.Split(request.PostFormValue("BanColors"), "\n")
					var BanColors []string
					for _, color := range BanColors_arr {
						BanColors = append(BanColors, strings.Trim(color, " \n\r"))

					}
					config.BanColors = BanColors

					config.BanMsg = request.PostFormValue("BanMsg")
					EmbedWidth, err := strconv.Atoi(request.PostFormValue("EmbedWidth"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.EmbedWidth = EmbedWidth
					}

					EmbedHeight, err := strconv.Atoi(request.PostFormValue("EmbedHeight"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.EmbedHeight = EmbedHeight
					}

					config.ExpandButton = (request.PostFormValue("ExpandButton") == "on")
					config.ImagesOpenNewTab = (request.PostFormValue("ImagesOpenNewTab") == "on")
					config.MakeURLsHyperlinked = (request.PostFormValue("MakeURLsHyperlinked") == "on")
					config.NewTabOnOutlinks = (request.PostFormValue("NewTabOnOutlinks") == "on")
					config.MinifyHTML = (request.PostFormValue("MinifyHTML") == "on")
					config.MinifyJS = (request.PostFormValue("MinifyJS") == "on")
					config.DateTimeFormat = request.PostFormValue("DateTimeFormat")
					AkismetAPIKey := request.PostFormValue("AkismetAPIKey")

					if err = checkAkismetAPIKey(AkismetAPIKey); err != nil {
						status += err.Error() + "<br />"
					} else {
						config.AkismetAPIKey = AkismetAPIKey
					}

					config.UseCaptcha = (request.PostFormValue("UseCaptcha") == "on")
					CaptchaWidth, err := strconv.Atoi(request.PostFormValue("CaptchaWidth"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.CaptchaWidth = CaptchaWidth
					}
					CaptchaHeight, err := strconv.Atoi(request.PostFormValue("CaptchaHeight"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.CaptchaHeight = CaptchaHeight
					}

					config.EnableGeoIP = (request.PostFormValue("EnableGeoIP") == "on")
					config.GeoIPDBlocation = request.PostFormValue("GeoIPDBlocation")

					MaxRecentPosts, err := strconv.Atoi(request.PostFormValue("MaxRecentPosts"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.MaxRecentPosts = MaxRecentPosts
					}

					config.EnableAppeals = (request.PostFormValue("EnableAppeals") == "on")
					MaxLogDays, err := strconv.Atoi(request.PostFormValue("MaxLogDays"))
					if err != nil {
						status += err.Error() + "<br />\n"
					} else {
						config.MaxLogDays = MaxLogDays
					}

					configJSON, err = json.MarshalIndent(config, "", "\t")
					if err != nil {
						status += err.Error() + "<br />\n"
					} else if err = ioutil.WriteFile("gochan.json", configJSON, 0777); err != nil {
						status = "Error writing gochan.json: %s\n" + err.Error()
					} else {
						status = "Wrote gochan.json successfully <br />"
						buildJS()
					}
				}
			}
			manageConfigBuffer := bytes.NewBufferString("")
			if err := manageConfigTmpl.Execute(manageConfigBuffer,
				map[string]interface{}{"config": config, "status": status},
			); err != nil {
				return html + gclog.Print(lErrorLog, "Error executing config management page: ", err.Error())
			}
			html += manageConfigBuffer.String()
			return
		}},
	"login": {
		Title:       "Login",
		Permissions: 0,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			if getStaffRank(request) > 0 {
				http.Redirect(writer, request, path.Join(config.SiteWebfolder, "manage"), http.StatusFound)
			}
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirect_action := request.FormValue("action")
			if redirect_action == "" {
				redirect_action = "announcements"
			}
			if username == "" || password == "" {
				//assume that they haven't logged in
				html = "\t<form method=\"POST\" action=\"" + config.SiteWebfolder + "manage?action=login\" id=\"login-box\" class=\"staff-form\">\n" +
					"\t\t<input type=\"hidden\" name=\"redirect\" value=\"" + redirect_action + "\" />\n" +
					"\t\t<input type=\"text\" name=\"username\" class=\"logindata\" /><br />\n" +
					"\t\t<input type=\"password\" name=\"password\" class=\"logindata\" /> <br />\n" +
					"\t\t<input type=\"submit\" value=\"Login\" />\n" +
					"\t</form>"
			} else {
				key := md5Sum(request.RemoteAddr + username + password + config.RandomSeed + randomString(3))[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(config.SiteWebfolder, "manage?action="+request.FormValue("redirect")), http.StatusFound)
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
			html = "<h1 class=\"manage-header\">Announcements</h1><br />"

			rows, err := querySQL("SELECT subject,message,poster,timestamp FROM DBPREFIXannouncements ORDER BY id DESC")
			defer rows.Close()
			if err != nil {
				return html + gclog.Print(lErrorLog, "Error getting announcements: ", err.Error())
			}
			iterations := 0
			for rows.Next() {
				announcement := new(Announcement)
				err = rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)
				if err != nil {
					html += gclog.Print(lErrorLog, "Error getting announcements: ", err.Error())
				} else {
					html += "<div class=\"section-block\">\n" +
						"<div class=\"section-title-block\"><b>" + announcement.Subject + "</b> by " + announcement.Poster + " at " + humanReadableTime(announcement.Timestamp) + "</div>\n" +
						"<div class=\"section-body\">" + announcement.Message + "\n</div></div>\n"
				}
				iterations++
			}

			if iterations == 0 {
				html += "No announcements"
			}
			return
		}},
	"bans": {
		Title:       "Bans",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (pageHTML string) {
			var post Post
			if request.FormValue("do") == "add" {
				ip := net.ParseIP(request.FormValue("ip"))
				name := request.FormValue("name")
				nameIsRegex := (request.FormValue("nameregex") == "on")
				checksum := request.FormValue("checksum")
				filename := request.FormValue("filename")
				durationForm := request.FormValue("duration")
				permaban := (durationForm == "" || durationForm == "0" || durationForm == "forever")
				duration, err := parseDurationString(durationForm)
				if err != nil {
					serveErrorPage(writer, err.Error())
				}
				expires := time.Now().Add(duration)
				var bantype int
				if request.FormValue("fullban") == "on" {
					bantype = 3
				} else {
					if request.FormValue("threadban") == "on" {
						bantype++
					}
					if request.FormValue("imageban") == "on" {
						bantype += 2
					}
				}
				if bantype == 0 {
					bantype = 3
				}

				boards := request.FormValue("boards")
				reason := html.EscapeString(request.FormValue("reason"))
				staffNote := html.EscapeString(request.FormValue("staffnote"))
				currentStaff, _ := getCurrentStaff(request)
				sqlStr := "INSERT INTO DBPREFIXbanlist (ip,name,name_is_regex,filename,file_checksum,boards,staff,expires,permaban,reason,type,staff_note) VALUES("
				for i := 1; i <= 12; i++ {
					sqlStr += "?"
					if i < 12 {
						sqlStr += ","
					}
				}
				sqlStr += ")"
				if _, err := execSQL(sqlStr,
					ip.String(), name, nameIsRegex, filename, checksum, boards, currentStaff, expires, permaban, reason, bantype, staffNote,
				); err != nil {
					pageHTML += err.Error()
				}
			}

			if request.FormValue("dir") != "" && request.FormValue("postid") != "" {
				boardDir := request.FormValue("dir")
				boards, err := getBoardArr(map[string]interface{}{
					"dir": boardDir,
				}, "")
				if err != nil {
					return pageHTML + gclog.Print(lErrorLog,
						"Error getting board list: ", err.Error())
				}
				if len(boards) < 1 {
					return pageHTML + gclog.Print(lStaffLog, "Board doesn't exist")
				}

				posts, err := getPostArr(map[string]interface{}{
					"id":      request.FormValue("postid"),
					"boardid": boards[0].ID,
				}, "")
				if err != nil {
					return pageHTML + gclog.Print(lErrorLog, "Error getting post list: ", err.Error())
				}
				if len(posts) < 1 {
					return pageHTML + gclog.Print(lStaffLog|lErrorLog, "Post doesn't exist")
				}
				post = posts[0]
			}
			rows, err := querySQL("SELECT ip,name,reason,boards,staff,timestamp,expires,permaban,can_appeal FROM DBPREFIXbanlist")
			defer rows.Close()
			if err != nil {
				return pageHTML + gclog.Print(lErrorLog, "Error getting ban list: ", err.Error())
			}

			var banlist []BanInfo
			for rows.Next() {
				var ban BanInfo
				rows.Scan(&ban.IP, &ban.Name, &ban.Reason, &ban.Boards, &ban.Staff, &ban.Timestamp, &ban.Expires, &ban.Permaban, &ban.CanAppeal)
				banlist = append(banlist, ban)
			}
			manageBansBuffer := bytes.NewBufferString("")

			if err := manageBansTmpl.Execute(manageBansBuffer,
				map[string]interface{}{"config": config, "banlist": banlist, "post": post},
			); err != nil {
				return pageHTML + gclog.Print(lErrorLog, "Error executing ban management page template: ", err.Error())
			}
			pageHTML += manageBansBuffer.String()
			return
		}},
	"getstaffjquery": {
		Permissions: 0,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			current_staff, err := getCurrentStaff(request)
			if err != nil {
				html = "nobody;0;"
				return
			}
			staff_rank := getStaffRank(request)
			if staff_rank == 0 {
				html = "nobody;0;"
				return
			}
			staff := new(Staff)
			if err := queryRowSQL("SELECT rank,boards FROM DBPREFIXstaff WHERE username = ?",
				[]interface{}{current_staff},
				[]interface{}{&staff.Rank, &staff.Boards},
			); err != nil {
				return html + gclog.Print(lErrorLog, "Error getting staff list: ", err.Error())
			}
			html = current_staff + ";" + strconv.Itoa(staff.Rank) + ";" + staff.Boards
			return
		}},
	"boards": {
		Title:       "Boards",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			var done bool
			board := new(Board)
			var boardCreationStatus string
			var err error
			var rows *sql.Rows
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
					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(lStaffLog|lErrorLog, "Directory %s/%s/ already exists.",
							config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "res"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(lStaffLog|lErrorLog, "Directory %s/%s/res/ already exists.",
							config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "src"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(lStaffLog|lErrorLog, "Directory %s/%s/src/ already exists.",
							config.DocumentRoot, board.Dir)
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "thumb"), 0666); err != nil {
						do = ""
						boardCreationStatus = gclog.Printf(lStaffLog|lErrorLog, "Directory %s/%s/thumb/ already exists.",
							config.DocumentRoot, board.Dir)
						break
					}

					boardCreationTimestamp := getSpecificSQLDateTime(board.CreatedOn)
					if _, err := execSQL(
						"INSERT INTO DBPREFIXboards (list_order,dir,type,upload_type,title,subtitle,"+
							"description,section,max_file_size,max_pages,default_style,locked,created_on,"+
							"anonymous,forced_anon,max_age,autosage_after,no_images_after,max_message_length,embeds_allowed,"+
							"redirect_to_thread,require_file,enable_catalog) "+
							"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
						board.ListOrder, board.Dir, board.Type, board.UploadType,
						board.Title, board.Subtitle, board.Description, board.Section,
						board.MaxFilesize, board.MaxPages, board.DefaultStyle,
						board.Locked, boardCreationTimestamp, board.Anonymous,
						board.ForcedAnon, board.MaxAge, board.AutosageAfter,
						board.NoImagesAfter, board.MaxMessageLength, board.EmbedsAllowed,
						board.RedirectToThread, board.RequireFile, board.EnableCatalog,
					); err != nil {
						do = ""
						boardCreationStatus = gclog.Print(lErrorLog, "Error creating board: ", err.Error())
						break
					} else {
						boardCreationStatus = "Board created successfully"
						buildBoards()
						resetBoardSectionArrays()
						gclog.Print(lStaffLog, "Boards rebuilt successfully")
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
					board.ThreadsPerPage = config.ThreadsPerPage
				}

				html = "<h1 class=\"manage-header\">Manage boards</h1>\n<form action=\"/manage?action=boards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"existing\" /><select name=\"boardselect\">\n<option>Select board...</option>\n"
				rows, err = querySQL("SELECT dir FROM DBPREFIXboards")
				defer rows.Close()
				if err != nil {
					return html + gclog.Print(lErrorLog, "Error getting board list: ", err.Error())
				}

				for rows.Next() {
					var boardDir string
					rows.Scan(&boardDir)
					html += "<option>" + boardDir + "</option>"
				}

				html += "</select> <input type=\"submit\" value=\"Edit\" /> <input type=\"submit\" value=\"Delete\" /></form><hr />" +
					"<h2 class=\"manage-header\">Create new board</h2>\n<span id=\"board-creation-message\">" + boardCreationStatus + "</span><br />"

				manageBoardsBuffer := bytes.NewBufferString("")
				allSections, _ = getSectionArr("")
				if len(allSections) == 0 {
					if _, err = execSQL(
						"INSERT INTO DBPREFIXsections (hidden,name,abbreviation) VALUES(0,'Main','main')",
					); err != nil {
						gclog.Print(lErrorLog, "Error creating new board section: ", err.Error())
					}
				}
				allSections, _ = getSectionArr("")

				if err := manageBoardsTmpl.Execute(manageBoardsBuffer, map[string]interface{}{
					"config":      config,
					"board":       board,
					"section_arr": allSections,
				}); err != nil {
					return html + gclog.Print(lErrorLog, "Error executing board management page template: ", err.Error())
				}
				html += manageBoardsBuffer.String()
				return
			}
			resetBoardSectionArrays()
			return
		}},
	"staffmenu": {
		Title:       "Staff menu",
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			rank := getStaffRank(request)

			html = "<a href=\"javascript:void(0)\" id=\"logout\" class=\"staffmenu-item\">Log out</a><br />\n" +
				"<a href=\"javascript:void(0)\" id=\"announcements\" class=\"staffmenu-item\">Announcements</a><br />\n"
			if rank == 3 {
				html += "<b>Admin stuff</b><br />\n<a href=\"javascript:void(0)\" id=\"staff\" class=\"staffmenu-item\">Manage staff</a><br />\n" +
					//"<a href=\"javascript:void(0)\" id=\"purgeeverything\" class=\"staffmenu-item\">Purge everything!</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"executesql\" class=\"staffmenu-item\">Execute SQL statement(s)</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"cleanup\" class=\"staffmenu-item\">Run cleanup</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildall\" class=\"staffmenu-item\">Rebuild all</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildfront\" class=\"staffmenu-item\">Rebuild front page</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildboards\" class=\"staffmenu-item\">Rebuild board pages</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"reparsehtml\" class=\"staffmenu-item\">Reparse all posts</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"boards\" class=\"staffmenu-item\">Add/edit/delete boards</a><br />\n"
			}
			if rank >= 2 {
				html += "<b>Mod stuff</b><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"bans\" class=\"staffmenu-item\">Ban User(s)</a><br />\n"
			}

			if rank >= 1 {
				html += "<a href=\"javascript:void(0)\" id=\"recentimages\" class=\"staffmenu-item\">Recently uploaded images</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"recentposts\" class=\"staffmenu-item\">Recent posts</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"searchip\" class=\"staffmenu-item\">Search posts by IP</a><br />\n"
			}
			return
		}},
	"rebuildfront": {
		Title:       "Rebuild front page",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			return buildFrontPage()
		}},
	"rebuildall": {
		Title:       "Rebuild everything",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			resetBoardSectionArrays()
			return buildFrontPage() + "<hr />\n" +
				buildBoardListJSON() + "<hr />\n" +
				buildBoards() + "<hr />\n" +
				buildJS() + "<hr />\n"
		}},
	"rebuildboards": {
		Title:       "Rebuild boards",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			return buildBoards()
		}},
	"reparsehtml": {
		Title:       "Reparse HTML",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			posts, err := getPostArr(map[string]interface{}{
				"deleted_timestamp": nilTimestamp,
			}, "")
			if err != nil {
				html += err.Error() + "<br />"
				return
			}

			for _, post := range posts {
				_, err = execSQL(
					"UPDATE DBPREFIXposts SET message = ? WHERE id = ? AND boardid = ?",
					formatMessage(post.MessageText), post.ID, post.BoardID,
				)
				if err != nil {
					return html + gclog.Printf(lErrorLog, "Error reparsing post (post ID: %d, board ID: %d)",
						post.ID, post.BoardID)
				}
			}
			html += "Done reparsing HTML<hr />" +
				buildFrontPage() + "<hr />" +
				buildBoardListJSON() + "<hr />" +
				buildBoards() + "<hr />"
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
			html = "<h1 class=\"manage-header\">Recent posts</h1>\nLimit by: <select id=\"limit\"><option>25</option><option>50</option><option>100</option><option>200</option></select>\n<br />\n<table width=\"100%%d\" border=\"1\">\n<colgroup><col width=\"25%%\" /><col width=\"50%%\" /><col width=\"17%%\" /></colgroup><tr><th></th><th>Message</th><th>Time</th></tr>"
			rows, err := querySQL("SELECT "+
				"DBPREFIXboards.dir AS boardname, "+
				"DBPREFIXposts.boardid AS boardid, "+
				"DBPREFIXposts.id AS id, "+
				"DBPREFIXposts.parentid AS parentid, "+
				"DBPREFIXposts.message AS message, "+
				"DBPREFIXposts.ip AS ip, "+
				"DBPREFIXposts.timestamp AS timestamp "+
				"FROM DBPREFIXposts, DBPREFIXboards "+
				"WHERE reviewed = 0 "+
				"AND DBPREFIXposts.deleted_timestamp = ? "+
				"AND boardid = DBPREFIXboards.id "+
				"ORDER BY timestamp DESC LIMIT ?",
				nilTimestamp, limit)
			defer rows.Close()
			if err != nil {
				return html + "<tr><td>" + gclog.Print(lErrorLog, "Error getting recent posts: ",
					err.Error()) + "</td></tr></table>"
			}

			for rows.Next() {
				recentpost := new(RecentPost)
				if err = rows.Scan(&recentpost.BoardName, &recentpost.BoardID,
					&recentpost.PostID, &recentpost.ParentID, &recentpost.Message,
					&recentpost.IP, &recentpost.Timestamp,
				); err != nil {
					return html + gclog.Print(lErrorLog, "Error getting recent posts: ",
						err.Error())
				}
				html += fmt.Sprintf(
					`<tr><td><b>Post:</b> <a href="%s/res/%d.html#%d">%s/%d</a><br /><b>IP:</b> %s</td><td>%s</td><td>%s</td></tr>`,
					path.Join(config.SiteWebfolder, recentpost.BoardName, "/res/", strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID)),
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
			boardDir := request.FormValue("dir")
			boards, err := getBoardArr(map[string]interface{}{
				"dir": boardDir,
			}, "")
			errMap := map[string]interface{}{
				"action":  "postInfo",
				"success": false,
			}
			if err != nil {
				errMap["message"] = err.Error()
				jsonErr, _ := marshalJSON(errMap, false)
				return jsonErr
			}
			if len(boards) < 1 {
				errMap["message"] = "Board doesn't exist"
				jsonErr, _ := marshalJSON(errMap, false)
				return jsonErr
			}

			posts, err := getPostArr(map[string]interface{}{
				"id":      request.FormValue("postid"),
				"boardid": boards[0].ID,
			}, "")
			if err != nil {
				errMap["message"] = err.Error()
				jsonErr, _ := marshalJSON(errMap, false)
				return jsonErr
			}
			if len(posts) < 1 {
				errMap["message"] = "Post doesn't exist"
				jsonErr, _ := marshalJSON(errMap, false)
				return jsonErr
			}
			jsonStr, _ := marshalJSON(posts[0], false)
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
			rows, err := querySQL("SELECT username,rank,boards,added_on FROM DBPREFIXstaff")
			defer rows.Close()
			if err != nil {
				return html + gclog.Print(lErrorLog, "Error getting staff list: ", err.Error())
			}

			iter := 1
			for rows.Next() {
				staff := new(Staff)
				if err = rows.Scan(&staff.Username, &staff.Rank, &staff.Boards, &staff.AddedOn); err != nil {
					return html + gclog.Print(lErrorLog, "Error getting staff list: ", err.Error())
				}
				username := request.FormValue("username")
				password := request.FormValue("password")
				rank := request.FormValue("rank")
				rankI, _ := strconv.Atoi(rank)
				if do == "add" {
					if err := newStaff(username, password, rankI); err != nil {
						serveErrorPage(writer, gclog.Print(lErrorLog,
							"Error creating new staff account %q: ", username, err.Error()))
						return
					}
				} else if do == "del" && username != "" {
					if err = deleteStaff(request.FormValue("username")); err != nil {
						serveErrorPage(writer, gclog.Printf(lErrorLog,
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
					staff.Username, rank, staff.Boards, humanReadableTime(staff.AddedOn), staff.Username)
				iter++
			}
			html += "</table>\n\n<hr />\n<h2 class=\"manage-header\">Add new staff</h2>\n\n" +
				"<form action=\"/manage?action=staff\" onsubmit=\"return makeNewStaff();\" method=\"POST\">\n" +
				"\t<input type=\"hidden\" name=\"do\" value=\"add\" />\n" +
				"\tUsername: <input id=\"username\" name=\"username\" type=\"text\" /><br />\n" +
				"\tPassword: <input id=\"password\" name=\"password\" type=\"password\" /><br />\n" +
				"\tRank: <select id=\"rank\" name=\"rank\">\n" +
				"\t\t<option value=\"3\">Admin</option>\n" +
				"\t\t<option value=\"2\">Moderator</option>\n" +
				"\t\t<option value=\"1\">Janitor</option>\n" +
				"\t\t</select><br />\n" +
				"\t\t<input id=\"submitnewstaff\" type=\"submit\" value=\"Add\" />\n" +
				"\t\t</form>"
			return
		}},
	"tempposts": {
		Title:       "Temporary posts lists",
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html += "<h1 class=\"manage-header\">Temporary posts</h1>"
			if len(tempPosts) == 0 {
				html += "No temporary posts<br />\n"
				return
			}
			for p, post := range tempPosts {
				html += fmt.Sprintf("Post[%d]: %#v<br />\n", p, post)
			}
			return
		}},
}
