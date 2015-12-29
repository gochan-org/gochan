package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)

type ManageFunction struct {
	Permissions int           // 0 -> non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	Callback    func() string //return string of html output
}

var (
	rebuildfront  func() string
	rebuildboards func() string
)

func callManageFunction(w http.ResponseWriter, r *http.Request, data interface{}) {
	request = *r
	writer = w
	cookies = r.Cookies()
	request.ParseForm()
	action := request.FormValue("action")
	staff_rank := getStaffRank()
	var manage_page_buffer bytes.Buffer
	manage_page_html := ""

	if action == "" {
		action = "announcements"
	}

	err := global_header_tmpl.Execute(&manage_page_buffer, config)
	if err != nil {
		fmt.Fprintf(writer, manage_page_html+err.Error()+"\n</body>\n</html>")
		return
	}

	err = manage_header_tmpl.Execute(&manage_page_buffer, config)
	if err != nil {
		fmt.Println(manage_page_html)
		fmt.Fprintf(writer, manage_page_html+err.Error()+"\n</body>\n</html>")
		return
	}

	if _, ok := manage_functions[action]; ok {
		if staff_rank >= manage_functions[action].Permissions {
			if action == "rebuildall" || action == "purgeeverything" {
				rebuildfront = manage_functions["rebuildfront"].Callback
				rebuildboards = manage_functions["rebuildboards"].Callback
			}
			manage_page_buffer.Write([]byte(manage_functions[action].Callback()))
		} else if staff_rank == 0 && manage_functions[action].Permissions == 0 {
			manage_page_buffer.Write([]byte(manage_functions[action].Callback()))
		} else if staff_rank == 0 {
			manage_page_buffer.Write([]byte(manage_functions["login"].Callback()))
		} else {
			manage_page_buffer.Write([]byte(action + " is undefined."))
		}
	} else {
		manage_page_buffer.Write([]byte(action + " is undefined."))
	}
	manage_page_buffer.Write([]byte("\n</body>\n</html>"))
	extension := getFileExtension(request.URL.Path)
	if extension == "" {
		//writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	}
	fmt.Fprintf(writer, manage_page_buffer.String())
}

func getCurrentStaff() (string, error) {
	session_cookie := getCookie("sessiondata")
	var key string
	if session_cookie == nil {
		return "", nil
	} else {
		key = session_cookie.Value
	}

	row := db.QueryRow("SELECT `data` FROM `" + config.DBprefix + "sessions` WHERE `key` = '" + key + "';")
	current_session := new(SessionsTable)

	err := row.Scan(&current_session.Data)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	return current_session.Data, nil
}

func getStaff(name string) (*StaffTable, error) {
	row := db.QueryRow("SELECT * FROM `" + config.DBprefix + "staff` WHERE `username` = '" + name + "';")
	staff_obj := new(StaffTable)
	err := row.Scan(&staff_obj.ID, &staff_obj.Username, &staff_obj.PasswordChecksum, &staff_obj.Salt, &staff_obj.Rank, &staff_obj.Boards, &staff_obj.AddedOn, &staff_obj.LastActive)
	return staff_obj, err
}

func getStaffRank() int {
	staffname, err := getCurrentStaff()
	if staffname == "" {
		return 0
	}
	if err != nil {
		return 0
	}

	staff, err := getStaff(staffname)
	if err != nil {
		error_log.Print(err.Error())
		return 0
	}
	return staff.Rank
}

func createSession(key string, username string, password string, request *http.Request, writer *http.ResponseWriter) int {
	//returs 0 for successful, 1 for password mismatch, and 2 for other

	if !validReferrer(*request) {
		mod_log.Print("Rejected login from possible spambot @ : " + request.RemoteAddr)
		return 2
	}
	staff, err := getStaff(username)
	if err != nil {
		fmt.Println(err.Error())
		error_log.Print(err.Error())
		return 1
	} else {
		success := bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
		if success == bcrypt.ErrMismatchedHashAndPassword {
			// password mismatch
			mod_log.Print("Failed login (password mismatch) from " + request.RemoteAddr + " at " + getSQLDateTime())
			return 1
		} else {
			// successful login
			cookie := &http.Cookie{Name: "sessiondata", Value: key, Path: "/", Domain: config.SiteDomain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour * 2))), MaxAge: 7200}
			// cookie := &http.Cookie{Name: "sessiondata", Value: key, Path: "/", Domain: config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2))),MaxAge: 7200}
			http.SetCookie(*writer, cookie)
			_, err := db.Exec("INSERT INTO `" + config.DBprefix + "sessions` (`key`, `data`, `expires`) VALUES('" + key + "','" + username + "', '" + getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2))) + "');")
			if err != nil {
				error_log.Print(err.Error())
				return 2
			}
			_, err = db.Exec("UPDATE `" + config.DBprefix + "staff` SET `last_active` ='" + getSQLDateTime() + "' WHERE `username` = '" + username + "';")
			if err != nil {
				error_log.Print(err.Error())
			}
			return 0
		}
	}
	return 2
}

var manage_functions = map[string]ManageFunction{
	/*"cleanup": {
		Permissions: 3,
		Callback: func() (html string) {
			html = "<h2>Cleanup</h2><br />"

			if (request.FormValue("run") == 1) {
				html += "<hr />Deleting non-deleted replies which belong to deleted threads.<hr />";
			 	boards_rows,err := db.Query("SELECT `id`,`dir` FROM `" + config.DBprefix + "boards`")
				if err != nil {
					html += "<tr><td>"+err.Error()+"</td></tr></table>"
					return
				}
				var id int
				var dir string
				for boards_rows.Next() {
					err = boards_rows.Scan(&id, &dir)
					html += "<b>Looking for orphans in /" + dir + "/</b><br />";

					parentid_rows, err := db.Query("SELECT `id`,`parentid` FROM `" + config.DBprefix + "posts` WHERE `boardid` = " + strconv.Itoa(id) + " AND `parentid` != '0' AND `is_deleted` = 0")
					if err != nil {
						html += err.Error()
						return
					}
					var id2 string
					var parentid string
					for parentid_rows.Next() {
						err = db.QueryRow("SELECT COUNT(*) FROM `" + config.DBprefix + "posts` WHERE `boardid` = " + id2 + " AND `id` = '" + parentid + "' AND `IS_DELETED` = 0")
						if err != nil {
							deletePost()
							$post_class = new Post($line['id'], $lineboard['name'], $lineboard['id']);
							$post_class->Delete;

							html +='Reply #%1$s\'s thread (#%2$s) does not exist! It has been deleted.'),$line['id'],$line['parentid']).'<br />';
						}


					}
				}

				$tpl_page .= '<hr />'. _gettext('Deleting unused images.') .'<hr />';
				$this->delunusedimages(true);
				$tpl_page .= '<hr />'. _gettext('Removing posts deleted more than one week ago from the database.') .'<hr />';
				$results = $tc_db->GetAll("SELECT `name`, `type`, `id` FROM `" . KU_DBPREFIX . "boards`");
				foreach ($results AS $line) {
					if ($line['type'] != 1) {
						$tc_db->Execute("DELETE FROM `" . KU_DBPREFIX . "posts` WHERE `boardid` = " . $line['id'] . " AND `IS_DELETED` = 1 AND `deleted_timestamp` < " . (time() - 604800) . "");
					}
				}
				$tpl_page .= _gettext('Optimizing all tables in database.') .'<hr />';
				if (KU_DBTYPE == 'mysql' || KU_DBTYPE == 'mysqli') {
					$results = $tc_db->GetAll("SHOW TABLES");
								foreach ($results AS $line) {
										$tc_db->Execute("OPTIMIZE TABLE `" . $line[0] . "`");
								}
				}
				if (KU_DBTYPE == 'postgres7' || KU_DBTYPE == 'postgres8' || KU_DBTYPE == 'postgres') {
									$results = $tc_db->GetAll("SELECT table_name FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'");
									foreach ($results AS $line) {
											$tc_db->Execute("VACUUM ANALYZE `" . $line[0] . "`");
									}
				}
				$tpl_page .= _gettext('Cleanup finished.');
				management_addlogentry(_gettext('Ran cleanup'), 2);
			} else {
				$tpl_page .= '<form action="manage_page.php?action=cleanup" method="post">'. "\n" .
							'	<input name="run" id="run" type="submit" value="'. _gettext('Run Cleanup') . '" />'. "\n" .
							'</form>';
			}
	}},*/
	"purgeeverything": {
		Permissions: 3,
		Callback: func() (html string) {
			html = "Purging everything ^_^<br />"
			rows, err := db.Query("SELECT `dir` FROM `" + config.DBprefix + "boards`;")
			if err != nil {
				html += err.Error()
				return
			}
			var board string
			for rows.Next() {
				err = rows.Scan(&board)
				if err != nil {
					html += err.Error()
					return
				}
				_, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "res"), "")
				if err != nil {
					html += err.Error()
					return
				}
				_, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "src"), "")
				if err != nil {
					html += err.Error()
					return
				}
				_, err = deleteMatchingFiles(path.Join(config.DocumentRoot, board, "thumb"), "")
				if err != nil {
					html += err.Error()
					return
				}
			}
			_, err = db.Exec("truncate `" + config.DBprefix + "posts`")
			if err != nil {
				html += err.Error() + "<br />"
				return
			}
			_, _ = db.Exec("ALTER TABLE `" + config.DBprefix + "posts` AUTO_INCREMENT = 1")
			html += "<br />Everything purged, rebuilding all<br />"
			html += rebuildboards() + "<hr />\n"
			return
		}},
	"executesql": {
		Permissions: 3,
		Callback: func() (html string) {
			statement := request.FormValue("sql")
			html = "<h1>Execute SQL statement(s)</h1><form method = \"POST\" action=\"/manage?action=executesql\">\n<textarea name=\"sql\" id=\"sql-statement\">" + statement + "</textarea>\n<input type=\"submit\" />\n</form>"
			if statement != "" {
				html += "<hr />"
				result, sqlerr := db.Exec(statement)
				fmt.Println(&result)

				if sqlerr != nil {
					html += sqlerr.Error()
				} else {
					html += "Statement esecuted successfully."
				}
			}
			return
		}},
	"login": {
		Permissions: 0,
		Callback: func() (html string) {
			if getStaffRank() > 0 {
				http.Redirect(writer, &request, path.Join(config.SiteWebfolder, "manage"), http.StatusFound)
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
				key := md5_sum(request.RemoteAddr + username + password + config.RandomSeed + generateSalt())[0:10]
				createSession(key, username, password, &request, &writer)
				http.Redirect(writer, &request, path.Join(config.SiteWebfolder, "/manage?action="+request.FormValue("redirect")), http.StatusFound)
			}
			return
		}},
	"logout": {
		Permissions: 1,
		Callback: func() (html string) {
			cookie := getCookie("sessiondata")
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
		Callback: func() (html string) {
			html = "<h1>Announcements</h1><br />"

			rows, err := db.Query("SELECT `subject`,`message`,`poster`,`timestamp` FROM `" + config.DBprefix + "announcements` ORDER BY `id` DESC;")
			if err != nil {
				error_log.Print(err.Error())
				html += err.Error()
				return
			}
			iterations := 0
			for rows.Next() {
				announcement := new(AnnouncementsTable)
				err = rows.Scan(&announcement.Subject, &announcement.Message, &announcement.Poster, &announcement.Timestamp)
				if err != nil {
					html += err.Error()
				} else {
					html += "<div class=\"section-block\">\n" +
						"<div class=\"section-title-block\"><b>" + announcement.Subject + "</b> by " + announcement.Poster + " at " + humanReadableTime(announcement.Timestamp) + "</div>\n" +
						"<div class=\"section-body\">" + announcement.Message + "\n</div></div>\n"
				}
				iterations += 1
			}

			if iterations == 0 {
				html += "No announcements"
			}
			return
		}},
	"managebans": {
		Permissions: 1,
		Callback: func() (html string) {
			var ban_which string // user, image, or both

			if request.PostFormValue("ban-user-button") == "Ban user" {
				ban_which = "user"
			} else if request.PostFormValue("ban-image-button") == "Ban image" {
				ban_which = "image"
			} else if request.PostFormValue("ban-both-button") == "Ban both" {
				ban_which = "both"
			}
			// if none of these are true, we can assume that the page was loaded without sending anything
			fmt.Println(ban_which)

			if ban_which == "user" {
				//var banned_tripcode string
				banned_ip := request.PostFormValue("ip")

				if banned_ip != "" {
					fmt.Println(banned_ip)
				}
			}

			boards_list_html := "		<span style=\"font-weight: bold;\">Boards: </span><br />\n" +
				"		<label>All boards <input type=\"checkbox\" id=\"allboards\" /></label> overrides individual board selection<br />\n"

			rows, err := db.Query("SELECT `dir` FROM `" + config.DBprefix + "boards`;")
			if err != nil {
				html += "<hr />" + err.Error()
				return
			}
			var board_dir string
			for rows.Next() {
				err = rows.Scan(&board_dir)
				if err != nil {
					html += "<hr />" + err.Error()
				}
				boards_list_html += "			<label>/" + board_dir + "/ <input type=\"checkbox\" id=\"" + board_dir + "\" class=\"board-check\"/></label>&nbsp;&nbsp;\n"
			}

			html = "<h1>Ban user(s)</h1>\n" +
				"<form method=\"POST\" action=\"/manage\">\n" +
				"<input type=\"hidden\" name=\"action\" value=\"managebans\" />\n" +
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

			rows, err = db.Query("SELECT * FROM `" + config.DBprefix + "banlist`")
			if err != nil {
				html += "</table><br />" + err.Error()
				error_log.Print(err.Error())
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
					html += "</table><br />" + err.Error()
					error_log.Print(err.Error())
					return
				}
				ban_name := ""
				if ban.Name+ban.Tripcode != "" {
					ban_name = ban.Name + "!" + ban.Tripcode
				}

				html += "<tr><td>" + ban.IP + "</td><td>" + ban_name + "</td><td>" + ban.Message + "</td><td>" + humanReadableTime(ban.Timestamp) + "</td><td>" + ban.BannedBy + "</td><td>" + ban.Reason + "</td><td>" + humanReadableTime(ban.Expires) + "</td><td>Delete</td></tr>"
				num_rows += 1
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
	"cleanup": {
		Permissions: 3,
		Callback: func() (html string) {

			return
		}},
	"getstaffjquery": {
		Permissions: 0,
		Callback: func() (html string) {
			current_staff, err := getCurrentStaff()
			if err != nil {
				html = "nobody;0;"
				return
			}
			staff_rank := getStaffRank()
			if staff_rank == 0 {
				html = "nobody;0;"
				return
			}
			row := db.QueryRow("SELECT `rank`,`boards` FROM `" + config.DBprefix + "staff` WHERE `username` = '" + current_staff + "';")
			staff := new(StaffTable)
			err = row.Scan(&staff.Rank, &staff.Boards)
			if err != nil {
				error_log.Print(err.Error())
				html += err.Error()
				return
			}
			html = current_staff + ";" + strconv.Itoa(staff.Rank) + ";" + staff.Boards
			return
		}},
	"manageboards": {
		Permissions: 3,
		Callback: func() (html string) {
			do := request.FormValue("do")
			var done bool
			board := new(BoardsTable)
			var board_creation_status string
			var err error
			var rows *sql.Rows
			for !done {
				switch {
				case do == "add":
					board.Dir = escapeString(request.FormValue("dir"))
					if board.Dir == "" {
						board_creation_status = "Error: \"Directory\" cannot be blank"
						do = ""
						continue
					}
					order_str := escapeString(request.FormValue("order"))
					board.Order, err = strconv.Atoi(order_str)
					if err != nil {
						board.Order = 0
					}
					board.Title = escapeString(request.FormValue("title"))
					if board.Title == "" {
						board_creation_status = "Error: \"Title\" cannot be blank"
						do = ""
						continue
					}
					board.Subtitle = escapeString(request.FormValue("subtitle"))
					board.Description = escapeString(request.FormValue("description"))
					section_str := escapeString(request.FormValue("section"))
					if section_str == "none" {
						section_str = "0"
					}

					board.Section, err = strconv.Atoi(section_str)
					if err != nil {
						board.Section = 0
					}
					maximagesize_str := escapeString(request.FormValue("maximagesize"))
					board.MaxImageSize, err = strconv.Atoi(maximagesize_str)
					if err != nil {
						board.MaxImageSize = 1024 * 4
					}

					maxpages_str := escapeString(request.FormValue("maxpages"))
					board.MaxPages, err = strconv.Atoi(maxpages_str)
					if err != nil {
						board.MaxPages = 11
					}
					board.DefaultStyle = escapeString(request.FormValue("defaultstyle"))
					board.Locked = (request.FormValue("locked") == "on")

					board.ForcedAnon = (request.FormValue("forcedanon") == "on")

					board.Anonymous = escapeString(request.FormValue("anonymous"))
					if board.Anonymous == "" {
						board.Anonymous = "Anonymous"
					}
					maxage_str := escapeString(request.FormValue("maxage"))
					board.MaxAge, err = strconv.Atoi(maxage_str)
					if err != nil {
						board.MaxAge = 0
					}
					autosageafter_str := escapeString(request.FormValue("autosageafter"))
					board.AutosageAfter, err = strconv.Atoi(autosageafter_str)
					if err != nil {
						board.AutosageAfter = 200
					}
					noimagesafter_str := escapeString(request.FormValue("noimagesafter"))
					board.NoImagesAfter, err = strconv.Atoi(noimagesafter_str)
					if err != nil {
						board.NoImagesAfter = 0
					}
					maxmessagelength_str := escapeString(request.FormValue("maxmessagelength"))
					board.MaxMessageLength, err = strconv.Atoi(maxmessagelength_str)
					if err != nil {
						board.MaxMessageLength = 1024 * 8
					}

					board.EmbedsAllowed = (request.FormValue("embedsallowed") == "on")
					board.RedirectToThread = (request.FormValue("redirecttothread") == "on")
					board.RequireFile = (request.FormValue("require_file") == "on")
					board.EnableCatalog = (request.FormValue("enablecatalog") == "on")

					//actually start generating stuff
					err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir), 0777)
					if err != nil {
						do = ""
						board_creation_status = err.Error()
						continue
					}

					err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "res"), 0777)
					if err != nil {
						do = ""
						board_creation_status = err.Error()
						continue
					}

					err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "thumb"), 0777)
					if err != nil {
						do = ""
						board_creation_status = err.Error()
						continue
					}

					err = os.Mkdir(path.Join(config.DocumentRoot, board.Dir, "src"), 0777)
					if err != nil {
						do = ""
						board_creation_status = err.Error()
						continue
					}
					_, err := db.Exec(
						"INSERT INTO `" + config.DBprefix + "boards` (" +
							"`order`, " +
							"`dir`, " +
							"`type`, " +
							"`upload_type`, " +
							"`title`, " +
							"`subtitle`, " +
							"`description`, " +
							"`section`, " +
							"`max_image_size`, " +
							"`max_pages`, " +
							"`locale`, " +
							"`default_style`, " +
							"`locked`, " +
							"`created_on`, " +
							"`anonymous`, " +
							"`forced_anon`, " +
							"`max_age`, " +
							"`autosage_after`, " +
							"`no_images_after`, " +
							"`max_message_length`, " +
							"`embeds_allowed`, " +
							"`redirect_to_thread`, " +
							"`require_file`, " +
							"`enable_catalog`" +
							") VALUES(" +
							strconv.Itoa(board.Order) + ", '" +
							board.Dir + "', " +
							strconv.Itoa(board.Type) + ", " +
							strconv.Itoa(board.UploadType) + ", '" +
							board.Title + "', '" +
							board.Subtitle + "', '" +
							board.Description + "', " +
							strconv.Itoa(board.Section) + ", " +
							strconv.Itoa(board.MaxImageSize) + ", " +
							strconv.Itoa(board.MaxPages) + ", '" +
							board.Locale + "', '" +
							board.DefaultStyle + "', " +
							Btoa(board.Locked) + ", '" +
							getSpecificSQLDateTime(board.CreatedOn) + "', '" +
							board.Anonymous + "', " +
							Btoa(board.ForcedAnon) + ", " +
							strconv.Itoa(board.MaxAge) + ", " +
							strconv.Itoa(board.AutosageAfter) + ", " +
							strconv.Itoa(board.NoImagesAfter) + ", " +
							strconv.Itoa(board.MaxMessageLength) + ", " +
							Btoa(board.EmbedsAllowed) + ", " +
							Btoa(board.RedirectToThread) + ", " +
							Btoa(board.RequireFile) + ", " +
							Btoa(board.EnableCatalog) + ")")
					if err != nil {
						do = ""
						board_creation_status = err.Error()
						continue
					} else {
						board_creation_status = "Board created successfully"
						done = true
					}
					resetBoardSectionArrays()
				case do == "del":
					// resetBoardSectionArrays()
				case do == "edit":
					// resetBoardSectionArrays()
				default:
					// put the default column values in the text boxes
					rows, err = db.Query("SELECT `column_name`,`column_default` FROM `information_schema`.`columns` WHERE `table_name` = '" + config.DBprefix + "boards'")
					if err != nil {
						html += err.Error()
						return
					}

					for rows.Next() {
						var column_name string
						var column_default string
						err = rows.Scan(&column_name, &column_default)
						column_default_int, _ := strconv.Atoi(column_default)
						column_default_bool := (column_default_int == 1)
						switch column_name {
						case "id":
							board.ID = column_default_int
						case "order":
							board.Order = column_default_int
						case "dir":
							board.Dir = column_default
						case "type":
							board.Type = column_default_int
						case "upload_type":
							board.UploadType = column_default_int
						case "title":
							board.Title = column_default
						case "subtitle":
							board.Subtitle = column_default
						case "description":
							board.Description = column_default
						case "section":
							board.Section = column_default_int
						case "max_image_size":
							board.MaxImageSize = column_default_int
						case "max_pages":
							board.MaxPages = column_default_int
						case "locale":
							board.Locale = column_default
						case "default_style":
							board.DefaultStyle = column_default
						case "locked":
							board.Locked = column_default_bool
						case "anonymous":
							board.Anonymous = column_default
						case "forced_anon":
							board.ForcedAnon = column_default_bool
						case "max_age":
							board.MaxAge = column_default_int
						case "autosage_after":
							board.AutosageAfter = column_default_int
						case "no_images_after":
							board.NoImagesAfter = column_default_int
						case "max_message_length":
							board.MaxMessageLength = column_default_int
						case "embeds_allowed":
							board.EmbedsAllowed = column_default_bool
						case "redirect_to_thread":
							board.RedirectToThread = column_default_bool
						case "require_file":
							board.RequireFile = column_default_bool
						case "enable_catalog":
							board.EnableCatalog = column_default_bool
						}
					}
				}

				html = "<h1>Manage boards</h1>\n<form action=\"/manage?action=manageboards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"existing\" /><select name=\"boardselect\">\n<option>Select board...</option>\n"
				rows, err = db.Query("SELECT `dir` FROM `" + config.DBprefix + "boards`;")
				if err != nil {
					html += err.Error()
					return
				}
				var board_dir string
				for rows.Next() {
					err = rows.Scan(&board_dir)
					html += "<option>" + board_dir + "</option>\n"
				}
				html += "</select> <input type=\"submit\" value=\"Edit\" /> <input type=\"submit\" value=\"Delete\" /></form><hr />"
				html += fmt.Sprintf("<h2>Create new board</h2>"+
					"<span id=\"board-creation-message\">%s</span><br />"+
					"<form action=\"/manage?action=manageboards\" method=\"POST\">"+
					"<input type=\"hidden\" name=\"do\" value=\"add\" />"+
					"Directory <input type=\"text\" name=\"dir\" value=\"%s\" /><br />"+
					"Order <input type=\"text\" name=\"order\" value=\"%d\" /><br />"+
					"Title <input type=\"text\" name=\"title\" value=\"%s\" /><br />"+
					"Subtitle <input type=\"text\" name=\"subtitle\" value=\"%s\" /><br />"+
					"Description <input type=\"text\" name=\"description\" value=\"%s\" /><br />"+
					"Section <select name=\"section\" selected=\"%d\">\n<option value=\"none\">Select section...</option>\n",
					board_creation_status, board.Dir, board.Order, board.Title, board.Subtitle, board.Description, board.Section)

				rows, err = db.Query("SELECT `name` FROM `" + config.DBprefix + "sections` WHERE `hidden` = 0 ORDER BY `order`;")
				if err != nil {
					html += err.Error()
					return
				}

				iter := 0
				var section_name string
				for rows.Next() {
					err = rows.Scan(&section_name)
					html += "<option value=\"" + strconv.Itoa(iter) + "\">" + section_name + "</option>\n"
					iter += 1
				}
				html += "</select><br />Max image size: <input type=\"text\" name=\"maximagesize\" value=\"" + strconv.Itoa(board.MaxImageSize) + "\" /><br />Max pages: <input type=\"text\" name=\"maxpages\" value=\"" + strconv.Itoa(board.MaxPages) + "\" /><br />Default style</td><td><select name=\"defaultstyle\" selected=\"" + board.DefaultStyle + "\">"
				for _, style := range config.Styles_img {
					html += "<option value=\"" + style + "\">" + style + "</option>"
				}

				html += "</select>Locked"
				if board.Locked {
					html += "<input type=\"checkbox\" name=\"locked\" checked/>"
				} else {
					html += "<input type=\"checkbox\" name=\"locked\" />"
				}

				html += "<br />Forced anonymity"

				if board.ForcedAnon {
					html += "<input type=\"checkbox\" name=\"forcedanon\" checked/>"
				} else {
					html += "<input type=\"checkbox\" name=\"forcedanon\" />"
				}

				html += "<br />Anonymous: <input type=\"text\" name=\"anonymous\" value=\"" + board.Anonymous + "\" /><br />" +
					"Max age: <input type=\"text\" name=\"maxage\" value=\"" + strconv.Itoa(board.MaxAge) + "\"/><br />" +
					"Bump limit: <input type=\"text\" name=\"autosageafter\" value=\"" + strconv.Itoa(board.AutosageAfter) + "\"/><br />" +
					"No images after <input type=\"text\" name=\"noimagesafter\" value=\"" + strconv.Itoa(board.NoImagesAfter) + "\"/>px<br />" +
					"Max message length</td><td><input type=\"text\" name=\"maxmessagelength\" value=\"" + strconv.Itoa(board.MaxMessageLength) + "\"/><br />" +
					"Embeds allowed "

				if board.EmbedsAllowed {
					html += "<input type=\"checkbox\" name=\"embedsallowed\" checked/>"
				} else {
					html += "<input type=\"checkbox\" name=\"embedsallowed\" />"
				}

				html += "<br />Redirect to thread</td><td>"
				if board.RedirectToThread {
					html += "<input type=\"checkbox\" name=\"redirecttothread\" checked/>"
				} else {
					html += "<input type=\"checkbox\" name=\"redirecttothread\" />"
				}

				html += "<br />Require an uploaded file"

				if board.RequireFile {
					html += "<input type=\"checkbox\" name=\"require_file\" checked/>"
				} else {
					html += "<input type=\"checkbox\" name=\"require_file\" />"
				}

				html += "<br />Enable catalog"

				if board.EnableCatalog {
					html += "<input type=\"checkbox\" name=\"enablecatalog\" checked />"
				} else {
					html += "<input type=\"checkbox\" name=\"enablecatalog\" />"
				}

				html += "<br /><input type=\"submit\" /></form>"
				return
			}
			resetBoardSectionArrays()
			return
		}},
	"staffmenu": {
		Permissions: 1,
		Callback: func() (html string) {
			rank := getStaffRank()

			html = "<a href=\"javascript:void(0)\" id=\"logout\" class=\"staffmenu-item\">Log out</a><br />\n" +
				"<a href=\"javascript:void(0)\" id=\"announcements\" class=\"staffmenu-item\">Announcements</a><br />\n"
			if rank == 3 {
				html += "<b>Admin stuff</b><br />\n<a href=\"javascript:void(0)\" id=\"managestaff\" class=\"staffmenu-item\">Manage staff</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"purgeeverything\" class=\"staffmenu-item\">Purge everything!</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"executesql\" class=\"staffmenu-item\">Execute SQL statement(s)</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildall\" class=\"staffmenu-item\">Rebuild all</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildfront\" class=\"staffmenu-item\">Rebuild front page</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"rebuildboards\" class=\"staffmenu-item\">Rebuild board pages</a><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"manageboards\" class=\"staffmenu-item\">Add/edit/delete boards</a><br />\n"
			}
			if rank >= 2 {
				html += "<b>Mod stuff</b><br />\n" +
					"<a href=\"javascript:void(0)\" id=\"managebans\" class=\"staffmenu-item\">Ban User(s)</a><br />\n"
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
		Callback: func() (html string) {
			return buildFrontPage()
		}},
	"rebuildall": {
		Permissions: 3,
		Callback: func() (html string) {
			html += rebuildfront() + "<hr />\n"
			html += rebuildboards() + "<hr />\n"
			return
		}},
	"rebuildboards": {
		Permissions: 3,
		Callback: func() (html string) {
			initTemplates()
			return buildBoards(true, 0)
		}},
	"recentposts": {
		Permissions: 1,
		Callback: func() (html string) {
			limit := request.FormValue("limit")
			if limit == "" {
				limit = "50"
			}
			html = "<h1>Recent posts</h1>\nLimit by: <select id=\"limit\"><option>25</option><option>50</option><option>100</option><option>200</option></select>\n<br />\n<table width=\"100%%d\" border=\"1\">\n<colgroup><col width=\"25%%\" /><col width=\"50%%\" /><col width=\"17%%\" /></colgroup><tr><th></th><th>Message</th><th>Time</th></tr>"
			rows, err := db.Query("SELECT  `" + config.DBprefix + "boards`.`dir` AS `boardname`, " +
				"`" + config.DBprefix + "posts`.`boardid` AS boardid, " +
				"`" + config.DBprefix + "posts`.`id` AS id, " +
				"`" + config.DBprefix + "posts`. " +
				"`parentid` AS parentid, " +
				"`" + config.DBprefix + "posts`. " +
				"`message` AS message, " +
				"`" + config.DBprefix + "posts`. " +
				"`ip` AS ip, " +
				"`" + config.DBprefix + "posts`. " +
				"`timestamp` AS timestamp  " +
				"FROM `" + config.DBprefix + "posts`, `" + config.DBprefix + "boards` " +
				"WHERE `reviewed` = 0 " +
				"AND `" + config.DBprefix + "posts`.`deleted_timestamp` = \"" + nil_timestamp + "\"  " +
				"AND `boardid` = `" + config.DBprefix + "boards`.`id` " +
				"ORDER BY `timestamp` DESC LIMIT " + limit + ";")
			if err != nil {
				html += "<tr><td>" + err.Error() + "</td></tr></table>"
				return
			}

			for rows.Next() {
				recentpost := new(RecentPost)
				err = rows.Scan(&recentpost.BoardName, &recentpost.BoardID, &recentpost.PostID, &recentpost.ParentID, &recentpost.Message, &recentpost.IP, &recentpost.Timestamp)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}
				html += "<tr><td><b>Post:</b> <a href=\"" + path.Join(config.SiteWebfolder, recentpost.BoardName, "/res/", strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID)) + "\">" + recentpost.BoardName + "/" + strconv.Itoa(recentpost.PostID) + "</a><br /><b>IP:</b> " + recentpost.IP + "</td><td>" + recentpost.Message + "</td><td>" + recentpost.Timestamp.Format("01/02/06, 15:04") + "</td></tr>"
			}
			html += "</table>"
			return
		}},
	"killserver": {
		Permissions: 3,
		Callback: func() (html string) {
			os.Exit(0)
			return
		}},
	"managestaff": {
		Permissions: 3,
		Callback: func() (html string) {
			//do := request.FormValue("do")
			html = "<h1>Staff</h1><br />\n" +
				"<table id=\"stafftable\" border=\"1\">\n" +
				"<tr><td><b>Username</b></td><td><b>Rank</b></td><td><b>Boards</b></td><td><b>Added on</b></td><td><b>Action</b></td></tr>\n"
			rows, err := db.Query("SELECT `username`,`rank`,`boards`,`added_on` FROM `" + config.DBprefix + "staff`;")
			if err != nil {
				html += "<tr><td>" + err.Error() + "</td></tr></table>"
				return
			}

			iter := 1
			for rows.Next() {
				staff := new(StaffTable)
				err = rows.Scan(&staff.Username, &staff.Rank, &staff.Boards, &staff.AddedOn)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}

				if request.FormValue("do") == "add" {
					new_username := request.FormValue("username")
					new_password := request.FormValue("password")
					new_rank := request.FormValue("rank")
					_, err := db.Exec("INSERT INTO `" + config.DBprefix + "staff` (`username`, `password_checksum`, `rank`) VALUES('" + new_username + "','" + bcrypt_sum(new_password) + "', '" + new_rank + "');")
					if err != nil {
						server.ServeErrorPage(writer, err.Error())
					}
				} else if request.FormValue("do") == "del" && request.FormValue("username") != "" {
					_, err := db.Exec("DELETE FROM `" + config.DBprefix + "staff` WHERE `username` = '" + request.FormValue("username") + "'")
					if err != nil {
						server.ServeErrorPage(writer, err.Error())
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
				html += "<tr><td>" + staff.Username + "</td><td>" + rank + "</td><td>" + staff.Boards + "</td><td>" + humanReadableTime(staff.AddedOn) + "</td><td><a href=\"/manage?action=staff&amp;o=del&amp;username=" + staff.Username + "\" style=\"float:right;color:red;\">X</a></td></tr>\n"
				iter += 1
			}
			html += "</table>\n\n<hr />\n<h2>Add new staff</h2>\n\n" +
				"<form action=\"manage?action=staff\" onsubmit=\"return makeNewStaff();\" method=\"POST\">\n" +
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
