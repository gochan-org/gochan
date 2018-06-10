package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// ManageFunction represents the functions accessed by staff members at /manage?action=<functionname>.
// Eventually a plugin system might allow you to add more
type ManageFunction struct {
	Permissions int                                                            // 0 -> non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	Callback    func(writer http.ResponseWriter, request *http.Request) string //return string of html output
}

func callManageFunction(writer http.ResponseWriter, request *http.Request) {
	var err error
	if err = request.ParseForm(); err != nil {
		serveErrorPage(writer, err.Error())
		errorLog.Println(customError(err))
	}

	action := request.FormValue("action")
	staffRank := getStaffRank(request)
	var managePageBuffer bytes.Buffer
	mangePageHTML := ""

	if action == "" {
		action = "announcements"
	}

	if action != "getstaffjquery" {
		if err = global_header_tmpl.Execute(&managePageBuffer, config); err != nil {
			handleError(0, customError(err))
			fmt.Fprintf(writer, mangePageHTML+err.Error()+"\n</body>\n</html>")
			return
		}

		if err = manage_header_tmpl.Execute(&managePageBuffer, config); err != nil {
			handleError(0, customError(err))
			fmt.Fprintf(writer, mangePageHTML+err.Error()+"\n</body>\n</html>")
			return
		}
	}

	if _, ok := manage_functions[action]; ok {
		if staffRank >= manage_functions[action].Permissions {
			managePageBuffer.Write([]byte(manage_functions[action].Callback(writer, request)))
		} else if staffRank == 0 && manage_functions[action].Permissions == 0 {
			managePageBuffer.Write([]byte(manage_functions[action].Callback(writer, request)))
		} else if staffRank == 0 {
			managePageBuffer.Write([]byte(manage_functions["login"].Callback(writer, request)))
		} else {
			managePageBuffer.Write([]byte(action + " is undefined."))
		}
	} else {
		managePageBuffer.Write([]byte(action + " is undefined."))
	}
	if action != "getstaffjquery" {
		managePageBuffer.Write([]byte("\n</body>\n</html>"))
	}

	/* extension := getFileExtension(request.URL.Path)
	if extension == "" {
		writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	} */
	fmt.Fprintf(writer, managePageBuffer.String())
}

func getCurrentStaff(request *http.Request) (string, error) {
	sessionCookie, err := request.Cookie("sessiondata")
	if err != nil {
		return "", nil
	}
	key := sessionCookie.Value
	current_session := new(SessionsTable)
	if err := queryRowSQL(
		"SELECT `data` FROM `"+config.DBprefix+"sessions` WHERE `key` = ?",
		[]interface{}{key},
		[]interface{}{&current_session.Data},
	); err != nil {
		return "", err
	}
	return current_session.Data, nil
}

func getStaff(name string) (*StaffTable, error) {
	staff_obj := new(StaffTable)
	err := queryRowSQL(
		"SELECT * FROM `"+config.DBprefix+"staff` WHERE `username` = ?",
		[]interface{}{name},
		[]interface{}{&staff_obj.ID, &staff_obj.Username, &staff_obj.PasswordChecksum, &staff_obj.Salt, &staff_obj.Rank, &staff_obj.Boards, &staff_obj.AddedOn, &staff_obj.LastActive},
	)
	return staff_obj, err
}

func getStaffRank(request *http.Request) int {
	staffname, err := getCurrentStaff(request)
	if staffname == "" {
		return 0
	}
	if err != nil {
		handleError(1, customError(err))
		return 0
	}

	staff, err := getStaff(staffname)
	if err != nil {
		handleError(1, customError(err))
		return 0
	}
	return staff.Rank
}

func createSession(key string, username string, password string, request *http.Request, writer http.ResponseWriter) int {
	//returns 0 for successful, 1 for password mismatch, and 2 for other
	domain := request.Host
	var err error
	chopPortNumRegex := regexp.MustCompile("(.+|\\w+):(\\d+)$")
	domain = chopPortNumRegex.Split(domain, -1)[0]

	if !validReferrer(request) {
		modLog.Print("Rejected login from possible spambot @ : " + request.RemoteAddr)
		return 2
	}
	staff, err := getStaff(username)
	if err != nil {
		handleError(1, customError(err))
		return 1
	} else {
		success := bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
		if success == bcrypt.ErrMismatchedHashAndPassword {
			// password mismatch
			modLog.Print("Failed login (password mismatch) from " + request.RemoteAddr + " at " + getSQLDateTime())
			return 1
		} else {
			// successful login, add cookie that expires in one month
			cookie := &http.Cookie{Name: "sessiondata", Value: key, Path: "/", Domain: domain, Expires: time.Now().Add(time.Duration(time.Hour * 730))}
			http.SetCookie(writer, cookie)
			if _, err = execSQL(
				"INSERT INTO `"+config.DBprefix+"sessions` (`key`, `data`, `expires`) VALUES(?,?,?)",
				key, username, getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*730))),
			); err != nil {
				handleError(1, customError(err))
				return 2
			}

			if _, err = execSQL(
				"UPDATE `"+config.DBprefix+"staff` SET `last_active` = ? WHERE `username` = ?", getSQLDateTime(), username,
			); err != nil {
				handleError(1, customError(err))
			}
			return 0
		}
	}
}

var manage_functions = map[string]ManageFunction{
	"cleanup": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = "<h2>Cleanup</h2><br />"
			var err error
			if request.FormValue("run") == "Run Cleanup" {
				html += "Removing deleted posts from the database.<hr />"
				if _, err = execSQL(
					"DELETE FROM `"+config.DBprefix+"posts` WHERE `deleted_timestamp` = ?", nilTimestamp,
				); err != nil {
					html += "<tr><td>" + handleError(1, err.Error()) + "</td></tr></table>"
					return
				}
				// TODO: remove orphaned replies
				// TODO: remove orphaned uploads

				html += "Optimizing all tables in database.<hr />"
				tableRows, tablesErr := querySQL("SHOW TABLES")
				defer closeRows(tableRows)
				if tablesErr != nil {
					html += "<tr><td>" + tablesErr.Error() + "</td></tr></table>"
					return
				}

				for tableRows.Next() {
					var table string
					tableRows.Scan(&table)
					if _, err := execSQL("OPTIMIZE TABLE `" + table + "`"); err != nil {
						html += handleError(1, err.Error()) + "<br />"
						return
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
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			if do == "save" {
				// configJSON, err := json.Marshal(config)
				// if err != nil {
				// 	html += err.Error()
				// 	return
				// }

				// err = ioutil.WriteFile("gochan.json", configJSON, 0666)
				// if err != nil {
				// 	html += "Error writing \"gochan.json\": %s\n" + err.Error()
				// 	return
				// }
			}
			manageConfigBuffer := bytes.NewBufferString("")

			if err := manage_config_tmpl.Execute(manageConfigBuffer, nil); err != nil {
				html += handleError(1, err.Error())
				return
			}
			html += manageConfigBuffer.String()
			return
		}},
	"purgeeverything": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = "<img src=\"/css/purge.jpg\" />"
			rows, err := querySQL("SELECT `dir` FROM `" + config.DBprefix + "boards`")
			defer closeRows(rows)
			if err != nil {
				html += err.Error()
				handleError(1, customError(err))
				return
			}
			var board string
			for rows.Next() {
				if err = rows.Scan(&board); err != nil {
					html += err.Error()
					handleError(1, customError(err))
					return
				}
				if _, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board), ".html"); err != nil {
					html += err.Error()
					handleError(1, customError(err))
					return
				}
				if _, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "res"), ".*"); err != nil {
					html += err.Error()
					handleError(1, customError(err))
					return
				}
				if _, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "src"), ".*"); err != nil {
					html += err.Error()
					handleError(1, customError(err))
					return
				}
				if _, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "thumb"), ".*"); err != nil {
					html += err.Error()
					handleError(1, customError(err))
					return
				}
			}
			if _, err = execSQL("TRUNCATE `" + config.DBprefix + "posts`"); err != nil {
				html += err.Error() + "<br />"
				handleError(1, customError(err))
				return
			}

			if _, err = execSQL("ALTER TABLE `" + config.DBprefix + "posts` AUTO_INCREMENT = 1"); err != nil {
				html += err.Error() + "<br />"
				handleError(1, customError(err))
				return
			}
			html += "<br />Everything purged, rebuilding all<br />" +
				buildBoards(true, 0) + "<hr />\n" +
				buildFrontPage()
			return
		}},
	"executesql": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			statement := request.FormValue("sql")
			html = "<h1>Execute SQL statement(s)</h1><form method = \"POST\" action=\"/manage?action=executesql\">\n<textarea name=\"sql\" id=\"sql-statement\">" + statement + "</textarea>\n<input type=\"submit\" />\n</form>"
			if statement != "" {
				html += "<hr />"
				if _, sqlerr := execSQL(statement); sqlerr != nil {
					html += handleError(1, sqlerr.Error())
				} else {
					html += "Statement esecuted successfully."
				}
			}
			return
		}},
	"login": {
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
				html = "\t<form method=\"POST\" action=\"/manage?action=login\" id=\"login-box\" class=\"staff-form\">\n" +
					"\t\t<input type=\"hidden\" name=\"redirect\" value=\"" + redirect_action + "\" />\n" +
					"\t\t<input type=\"text\" name=\"username\" class=\"logindata\" /><br />\n" +
					"\t\t<input type=\"password\" name=\"password\" class=\"logindata\" /> <br />\n" +
					"\t\t<input type=\"submit\" value=\"Login\" />\n" +
					"\t</form>"
			} else {
				key := md5Sum(request.RemoteAddr + username + password + config.RandomSeed + generateSalt())[0:10]
				createSession(key, username, password, request, writer)
				http.Redirect(writer, request, path.Join(config.SiteWebfolder, "/manage?action="+request.FormValue("redirect")), http.StatusFound)
			}
			return
		}},
	"logout": {
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			cookie, err := request.Cookie("sessiondata")
			if err != nil {
				serveErrorPage(writer, err.Error())
			}
			var key string
			if cookie != nil {
				key = cookie.Value
				new_expire := time.Now().AddDate(0, 0, -1)
				new_cookie := &http.Cookie{
					Name:       "sessiondata",
					Value:      cookie.Value,
					Path:       "/",
					Domain:     config.SiteDomain,
					Expires:    new_expire,
					RawExpires: new_expire.Format(time.UnixDate),
					MaxAge:     -1,
					Secure:     true,
					HttpOnly:   true,
					Raw:        "sessiondata=" + key}
				// new_cookie := &http.Cookie{Name: "sessiondata",Value: cookie.Value,Path: "/",Domain: config.Domain,Expires: new_expire,RawExpires: new_expire.Format(time.UnixDate),MaxAge: -1,Secure: true,HttpOnly: true,Raw: "sessiondata="+key}
				http.SetCookie(writer, new_cookie)
				return "Logged out successfully"
			}
			return "wat"
		}},
	"announcements": {
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			html = "<h1>Announcements</h1><br />"

			rows, err := querySQL("SELECT `subject`,`message`,`poster`,`timestamp` FROM `" + config.DBprefix + "announcements` ORDER BY `id` DESC")
			defer closeRows(rows)
			if err != nil {
				html += handleError(1, err.Error())
				return
			}
			iterations := 0
			for rows.Next() {
				announcement := new(AnnouncementsTable)
				err = rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)
				if err != nil {
					html += handleError(1, err.Error())
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
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			var ban_which string // user, image, or both

			if request.PostFormValue("ban-user-button") == "Ban user" {
				ban_which = "user"
			} else if request.PostFormValue("ban-image-button") == "Ban image" {
				ban_which = "image"
			} else if request.PostFormValue("ban-both-button") == "Ban both" {
				ban_which = "both"
			}
			// if none of these are true, we can assume that the page was loaded without sending anything
			println(0, "ban_which"+ban_which)

			if ban_which == "user" {
				//var banned_tripcode string
				banned_ip := request.PostFormValue("ip")
				if banned_ip != "" {
					println(0, banned_ip)
				}
			}

			boards_list_html := "		<span style=\"font-weight: bold;\">Boards: </span><br />\n" +
				"		<label>All boards <input type=\"checkbox\" id=\"allboards\" /></label> overrides individual board selection<br />\n"

			rows, err := querySQL("SELECT `dir` FROM `" + config.DBprefix + "boards`")
			defer closeRows(rows)
			if err != nil {
				html += "<hr />" + handleError(1, err.Error())
				return
			}
			var board_dir string
			for rows.Next() {
				if err = rows.Scan(&board_dir); err != nil {
					html += "<hr />" + handleError(1, err.Error())
				}
				boards_list_html += "			<label>/" + board_dir + "/ <input type=\"checkbox\" id=\"" + board_dir + "\" class=\"board-check\"/></label>&nbsp;&nbsp;\n"
			}

			html = "<h1>Ban user(s)</h1>\n" +
				"<form method=\"POST\" action=\"/manage\">\n" +
				"<input type=\"hidden\" name=\"action\" value=\"bans\" />\n" +
				"<fieldset><legend>User(s)</legend>" +
				"	<div id=\"ip\" class=\"ban-type-div\" style=\"width:100%%; display: inline;\">\n" +
				"		<span style=\"font-weight: bold;\">IP address:</span> <input type=\"text\" name=\"ip\" /><br />\n" +
				"		\"192.168.1.36\" will ban posts from that IP address<br />\n" +
				"		\"192.168\" will block all IPs starting with 192.168<br /><hr />\n" +
				"	</div>\n" +
				"	<div id=\"name\" class=\"ban-type-div\" style=\"width:100%%;\">\n" +
				"		<span style=\"font-weight: bold;\">Name/tripcode:</span> <input type=\"text\" name=\"ip\" /><br />\n" +
				"		(format: \"Poster!tripcode\", \"!tripcode\", or \"Poster\")<br />\n" +
				"		<hr />\n" +
				"	</div>\n" +
				"		<span style=\"font-weight: bold;\">Duration: </span><br />\n" +
				"		<label>Permanent ban (overrides duration dropdowns if checked)<input type=\"checkbox\" name=\"forever\" value=\"forever\" /></label><br />\n" +
				"		<div class=\"duration-select\"></div>\n<hr />\n" +
				boards_list_html + "<hr />\n" +
				"	<div id=\"reason-staffnote\" style=\"text-align: right; float:left;\">\n" +
				"		<span style=\"font-weight: bold;\">Reason: </span><input type=\"text\" name=\"reason\" /><br />\n" +
				"		<span style=\"font-weight: bold;\">Staff note: </span><input type=\"text\" name=\"staff-note\" /><br />\n" +
				"	</div>\n<br /><br /><br /><input type=\"submit\" name=\"ban-user-button\" value=\"Ban user\"/>" +
				"</fieldset>\n<br />\n<hr />\n" +
				"<fieldset><legend>Image</legend>\n" +
				"	This will disallow an image with this hash from being posted, and will ban users who try to post it for the specified amount of time.<br /><br />\n" +
				"	<label style=\"font-weight: bold;\">Ban image hash: <input type=\"checkbox\" /></label><br />\n" +
				"		<span style=\"font-weight: bold;\">Duration: </span><br />\n" +
				"		<label>Permanent ban (overrides duration dropdowns if checked)<input type=\"checkbox\" name=\"forever\" value=\"forever\" /></label><br />\n" +
				"		<div class=\"duration-select\"></div>\n" +
				"		<hr />\n" +
				boards_list_html + "<hr />\n" +
				"	<div id=\"reason-staffnote\" style=\"text-align: right; float:left;\">\n" +
				"		<span style=\"font-weight: bold;\">Reason: </span><input type=\"text\" name=\"reason\" /><br />\n" +
				"		<span style=\"font-weight: bold;\">Staff note: </span><input type=\"text\" name=\"staff-note\" /><br />\n" +
				"	</div>\n<br /><br /><br /><input type=\"submit\" name=\"ban-image-button\" value=\"Ban image\"/>" +
				"</fieldset><br />\n" +
				"<input type=\"submit\" name=\"ban-both-button\" value=\"Ban both\" /></form>\n</br />" +
				"<h2>Banned IPs</h2>\n"

			rows, err = querySQL("SELECT * FROM `" + config.DBprefix + "banlist`")
			if err != nil {
				html += "</table><br />" + handleError(1, err.Error())
				return
			}
			var ban BanlistTable

			num_rows := 0
			for rows.Next() {
				if num_rows == 0 {
					html += "<table width=\"100%%\" border=\"1\">\n" +
						"<tr><th>IP</th><th>Name/Tripcode</th><th>Message</th><th>Date added</th><th>Added by</th><th>Reason</th><th>Expires/expired</th><th></th></tr>"
				}
				err = rows.Scan(&ban.ID, &ban.AllowRead, &ban.IP, &ban.Name, &ban.Tripcode, &ban.Message, &ban.SilentBan, &ban.Boards, &ban.BannedBy, &ban.Timestamp, &ban.Expires, &ban.Reason, &ban.StaffNote, &ban.AppealMessage, &ban.AppealAt)
				if err != nil {
					html += "</table><br />" + handleError(1, err.Error())
					return
				}
				ban_name := ""
				if ban.Name+ban.Tripcode != "" {
					ban_name = ban.Name + "!" + ban.Tripcode
				}

				html += "<tr><td>" + ban.IP + "</td><td>" + ban_name + "</td><td>" + ban.Message + "</td><td>" + humanReadableTime(ban.Timestamp) + "</td><td>" + ban.BannedBy + "</td><td>" + ban.Reason + "</td><td>" + humanReadableTime(ban.Expires) + "</td><td>Delete</td></tr>"
				num_rows++
			}
			if num_rows == 0 {
				html += "No banned IPs"
			} else {
				html += "</table>\n"
			}

			// html += "<tr><td>127.0.0.1</td><td>Banned message</td><td>12/25/1991</td><td>Luna</td><td>Spam</td><td>never</td><td>Delete</td></tr>" +

			html += "<br /><br /><br />" +
				"<script type=\"text/javascript\">banPage();</script>\n "
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
			staff := new(StaffTable)
			if err := queryRowSQL("SELECT `rank`,`boards` FROM `"+config.DBprefix+"staff` WHERE `username` = ?",
				[]interface{}{current_staff},
				[]interface{}{&staff.Rank, &staff.Boards},
			); err != nil {
				html += handleError(1, "Error getting staff list: "+err.Error())
				return
			}
			html = current_staff + ";" + strconv.Itoa(staff.Rank) + ";" + staff.Boards
			return
		}},
	"boards": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			var done bool
			board := new(BoardsTable)
			var board_creation_status string
			var err error
			var rows *sql.Rows
			for !done {
				switch {
				case do == "add":
					board.Dir = request.FormValue("dir")
					if board.Dir == "" {
						board_creation_status = "Error: \"Directory\" cannot be blank"
						do = ""
						continue
					}
					order_str := request.FormValue("order")
					board.Order, err = strconv.Atoi(order_str)
					if err != nil {
						board.Order = 0
					}
					board.Title = request.FormValue("title")
					if board.Title == "" {
						board_creation_status = "Error: \"Title\" cannot be blank"
						do = ""
						continue
					}
					board.Subtitle = request.FormValue("subtitle")
					board.Description = request.FormValue("description")
					section_str := request.FormValue("section")
					if section_str == "none" {
						section_str = "0"
					}

					board.CreatedOn = time.Now()
					board.Section, err = strconv.Atoi(section_str)
					if err != nil {
						board.Section = 0
					}
					board.MaxImageSize, err = strconv.Atoi(request.FormValue("maximagesize"))
					if err != nil {
						board.MaxImageSize = 1024 * 4
					}

					board.MaxPages, err = strconv.Atoi(request.FormValue("maxpages"))
					if err != nil {
						board.MaxPages = 11
					}
					board.DefaultStyle = request.FormValue("defaultstyle")
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
						board_creation_status = handleError(1, "ERROR: directory /"+config.DocumentRoot+"/"+board.Dir+"/ already exists!")
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "res"), 0666); err != nil {
						do = ""
						board_creation_status = handleError(1, "ERROR: directory /"+config.DocumentRoot+"/"+board.Dir+"/res/ already exists!")
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "thumb"), 0666); err != nil {
						do = ""
						board_creation_status = handleError(1, "ERROR: directory /"+config.DocumentRoot+"/"+board.Dir+"/thumb/ already exists!")
						break
					}

					if err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "src"), 0666); err != nil {
						do = ""
						board_creation_status = handleError(1, "ERROR: directory /"+config.DocumentRoot+"/"+board.Dir+"/src/ already exists!")
						break
					}
					boardCreationTimestamp := getSpecificSQLDateTime(board.CreatedOn)
					if _, err := execSQL(
						"INSERT INTO `"+config.DBprefix+"boards` (`order`,`dir`,`type`,`upload_type`,`title`,`subtitle`,"+
							"`description`,`section`,`max_image_size`,`max_pages`,`locale`,`default_style`,`locked`,`created_on`,"+
							"`anonymous`,`forced_anon`,`max_age`,`autosage_after`,`no_images_after`,`max_message_length`,`embeds_allowed`,"+
							"`redirect_to_thread`,`require_file`,`enable_catalog`) "+
							"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
						&board.Order, &board.Dir, &board.Type, &board.UploadType,
						&board.Title, &board.Subtitle, &board.Description, &board.Section,
						&board.MaxImageSize, &board.MaxPages, &board.Locale, &board.DefaultStyle,
						&board.Locked, &boardCreationTimestamp, &board.Anonymous,
						&board.ForcedAnon, &board.MaxAge, &board.AutosageAfter,
						&board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed,
						&board.RedirectToThread, &board.RequireFile, &board.EnableCatalog,
					); err != nil {
						do = ""
						board_creation_status = handleError(1, "Error creating board: "+customError(err))
						break
					} else {
						board_creation_status = "Board created successfully"
						println(2, board_creation_status)
						buildBoards(true, 0)
						resetBoardSectionArrays()
						println(2, "Boards rebuilt successfully")
						done = true
					}
					break
				case do == "del":
					// resetBoardSectionArrays()
				case do == "edit":
					// resetBoardSectionArrays()
				default:
					// put the default column values in the text boxes
					rows, err = querySQL("SELECT `column_name`,`column_default` FROM `information_schema`.`columns` WHERE `table_name` = '" + config.DBprefix + "boards'")
					if err != nil {
						html += handleError(1, "Error getting column names from boards table:"+err.Error())
						return
					}
					for rows.Next() {
						var columnName string
						var columnDefault string
						rows.Scan(&columnName, &columnDefault)
						columnDefaultInt, _ := strconv.Atoi(columnDefault)
						columnDefaultBool := (columnDefaultInt == 1)
						switch columnName {
						case "id":
							board.ID = columnDefaultInt
						case "order":
							board.Order = columnDefaultInt
						case "dir":
							board.Dir = columnDefault
						case "type":
							board.Type = columnDefaultInt
						case "upload_type":
							board.UploadType = columnDefaultInt
						case "title":
							board.Title = columnDefault
						case "subtitle":
							board.Subtitle = columnDefault
						case "description":
							board.Description = columnDefault
						case "section":
							board.Section = columnDefaultInt
						case "max_image_size":
							board.MaxImageSize = columnDefaultInt
						case "max_pages":
							board.MaxPages = columnDefaultInt
						case "locale":
							board.Locale = columnDefault
						case "default_style":
							board.DefaultStyle = columnDefault
						case "locked":
							board.Locked = columnDefaultBool
						case "anonymous":
							board.Anonymous = columnDefault
						case "forced_anon":
							board.ForcedAnon = columnDefaultBool
						case "max_age":
							board.MaxAge = columnDefaultInt
						case "autosage_after":
							board.AutosageAfter = columnDefaultInt
						case "no_images_after":
							board.NoImagesAfter = columnDefaultInt
						case "max_message_length":
							board.MaxMessageLength = columnDefaultInt
						case "embeds_allowed":
							board.EmbedsAllowed = columnDefaultBool
						case "redirect_to_thread":
							board.RedirectToThread = columnDefaultBool
						case "require_file":
							board.RequireFile = columnDefaultBool
						case "enable_catalog":
							board.EnableCatalog = columnDefaultBool
						}
					}
				}

				html = "<h1>Manage boards</h1>\n<form action=\"/manage?action=boards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"existing\" /><select name=\"boardselect\">\n<option>Select board...</option>\n"
				rows, err = querySQL("SELECT `dir` FROM `" + config.DBprefix + "boards`")
				defer closeRows(rows)
				if err != nil {
					html += handleError(1, err.Error())
					return
				}

				for rows.Next() {
					var boardDir string
					rows.Scan(&boardDir)
					html += "<option>" + boardDir + "</option>\n"
				}

				html += "</select> <input type=\"submit\" value=\"Edit\" /> <input type=\"submit\" value=\"Delete\" /></form><hr />" +
					"<h2>Create new board</h2>\n<span id=\"board-creation-message\">" + board_creation_status + "</span><br />"

				manageBoardsBuffer := bytes.NewBufferString("")
				allSections, _ = getSectionArr("")
				if len(allSections) == 0 {
					execSQL("INSERT INTO `" + config.DBprefix + "sections` (`hidden`,`name`,`abbreviation`) VALUES(0,'Main','main')")
				}
				allSections, _ = getSectionArr("")

				if err := manage_boards_tmpl.Execute(manageBoardsBuffer, map[string]interface{}{
					"board":       board,
					"section_arr": allSections,
				}); err != nil {
					html += handleError(1, err.Error())
					return
				}
				html += manageBoardsBuffer.String()
				return
			}
			resetBoardSectionArrays()
			return
		}},
	"staffmenu": {
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
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			return buildFrontPage()
		}},
	"rebuildall": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			return buildFrontPage() + "<hr />\n" +
				buildBoardListJSON() + "<hr />\n" +
				buildBoards(true, 0) + "<hr />\n"
		}},
	"rebuildboards": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			initTemplates()
			return buildBoards(true, 0)
		}},
	"reparsehtml": {
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
				_, err = execSQL("UPDATE `"+config.DBprefix+"posts` SET `message` = ? WHERE `id` = ? AND `boardid` = ?",
					formatMessage(post.MessageText), post.ID, post.BoardID,
				)
				if err != nil {
					html += handleError(1, err.Error()) + "<br />"
					return
				}
			}
			html += "Done reparsing HTML<hr />" +
				buildFrontPage() + "<hr />\n" +
				buildBoardListJSON() + "<hr />\n" +
				buildBoards(true, 0) + "<hr />\n"
			return
		}},
	"recentposts": {
		Permissions: 1,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			limit := request.FormValue("limit")
			if limit == "" {
				limit = "50"
			}
			html = "<h1>Recent posts</h1>\nLimit by: <select id=\"limit\"><option>25</option><option>50</option><option>100</option><option>200</option></select>\n<br />\n<table width=\"100%%d\" border=\"1\">\n<colgroup><col width=\"25%%\" /><col width=\"50%%\" /><col width=\"17%%\" /></colgroup><tr><th></th><th>Message</th><th>Time</th></tr>"
			rows, err := querySQL(
				"SELECT `"+config.DBprefix+"boards`.`dir` AS `boardname`, "+
					"`"+config.DBprefix+"posts`.`boardid` AS boardid, "+
					"`"+config.DBprefix+"posts`.`id` AS id, "+
					"`"+config.DBprefix+"posts`. "+
					"`parentid` AS parentid, "+
					"`"+config.DBprefix+"posts`. "+
					"`message` AS message, "+
					"`"+config.DBprefix+"posts`. "+
					"`ip` AS ip, "+
					"`"+config.DBprefix+"posts`. "+
					"`timestamp` AS timestamp  "+
					"FROM `"+config.DBprefix+"posts`, `"+config.DBprefix+"boards` "+
					"WHERE `reviewed` = 0 "+
					"AND `"+config.DBprefix+"posts`.`deleted_timestamp` = ? "+
					"AND `boardid` = `"+config.DBprefix+"boards`.`id` "+
					"ORDER BY `timestamp` DESC LIMIT ?",
				nilTimestamp, limit,
			)
			defer closeRows(rows)
			if err != nil {
				html += "<tr><td>" + handleError(1, err.Error()) + "</td></tr></table>"
				return
			}

			for rows.Next() {
				recentpost := new(RecentPost)
				if err = rows.Scan(&recentpost.BoardName, &recentpost.BoardID,
					&recentpost.PostID, &recentpost.ParentID, &recentpost.Message,
					&recentpost.IP, &recentpost.Timestamp,
				); err != nil {
					return handleError(1, err.Error())
				}
				html += "<tr><td><b>Post:</b> <a href=\"" + path.Join(config.SiteWebfolder, recentpost.BoardName, "/res/", strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID)) + "\">" + recentpost.BoardName + "/" + strconv.Itoa(recentpost.PostID) + "</a><br /><b>IP:</b> " + recentpost.IP + "</td><td>" + recentpost.Message + "</td><td>" + recentpost.Timestamp.Format("01/02/06, 15:04") + "</td></tr>"
			}
			html += "</table>"
			return
		}},
	"killserver": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			os.Exit(0)
			return
		}},
	"staff": {
		Permissions: 3,
		Callback: func(writer http.ResponseWriter, request *http.Request) (html string) {
			do := request.FormValue("do")
			html = "<h1>Staff</h1><br />\n" +
				"<table id=\"stafftable\" border=\"1\">\n" +
				"<tr><td><b>Username</b></td><td><b>Rank</b></td><td><b>Boards</b></td><td><b>Added on</b></td><td><b>Action</b></td></tr>\n"
			rows, err := querySQL("SELECT `username`,`rank`,`boards`,`added_on` FROM `" + config.DBprefix + "staff`")
			defer closeRows(rows)
			if err != nil {
				html += "<tr><td>" + handleError(1, err.Error()) + "</td></tr></table>"
				return
			}

			iter := 1
			for rows.Next() {
				staff := new(StaffTable)
				if err = rows.Scan(&staff.Username, &staff.Rank, &staff.Boards, &staff.AddedOn); err != nil {
					handleError(1, err.Error())
					return err.Error()
				}

				if do == "add" {
					newUsername := request.FormValue("username")
					newPassword := request.FormValue("password")
					newRank := request.FormValue("rank")
					if _, err := execSQL("INSERT INTO `"+config.DBprefix+"staff` (`username`, `password_checksum`, `rank`) VALUES(?,?,?)",
						&newUsername, bcryptSum(newPassword), &newRank,
					); err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
					}
				} else if do == "del" && request.FormValue("username") != "" {
					if _, err = execSQL("DELETE FROM `"+config.DBprefix+"staff` WHERE `username` = ?",
						request.FormValue("username"),
					); err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
					}
				}

				var rank string
				switch {
				case staff.Rank == 3:
					rank = "admin"
				case staff.Rank == 2:
					rank = "mod"
				case staff.Rank == 1:
					rank = "janitor"
				}
				html += "<tr><td>" + staff.Username + "</td><td>" + rank + "</td><td>" + staff.Boards + "</td><td>" + humanReadableTime(staff.AddedOn) + "</td><td><a href=\"/manage?action=staff&amp;do=del&amp;username=" + staff.Username + "\" style=\"float:right;color:red;\">X</a></td></tr>\n"
				iter++
			}
			html += "</table>\n\n<hr />\n<h2>Add new staff</h2>\n\n" +
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
}
