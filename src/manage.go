package main

import (
	"bytes"
	"code.google.com/p/go.crypto/bcrypt"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"strconv"
	"time"
)


type ManageFunction struct {
	Permissions int // 0 -> non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	Callback func() string //return string of html output
}

var (
	StaffNotFoundErr = errors.New("Username doesn't exist")
	PasswordMismatchErr = errors.New("Incorrect password")
	rebuildfront func() string
	rebuildboards func() string
	rebuildthreads func() string
)

func callManageFunction(w http.ResponseWriter, r *http.Request) {
	request = *r
	writer = w
	cookies = r.Cookies()
	request.ParseForm()
	action := request.FormValue("action")
	staff_rank := getStaffRank()
	var manage_page_buffer bytes.Buffer
	manage_page_html := ""

	if action == ""  {
		action = "announcements"
	}

	err := global_header_tmpl.Execute(&manage_page_buffer,config)
	if err != nil {
		fmt.Fprintf(writer,manage_page_html + err.Error() + "\n</body>\n</html>")
		return
	}

	err = manage_header_tmpl.Execute(&manage_page_buffer,config)
	if err != nil {
		fmt.Fprintf(writer,manage_page_html + err.Error() + "\n</body>\n</html>")
		return
	}


	if _,ok := manage_functions[action]; ok {
		if staff_rank >= manage_functions[action].Permissions {
			if action == "rebuildall" {
				rebuildfront = manage_functions["rebuildfront"].Callback
				rebuildboards = manage_functions["rebuildboards"].Callback
				rebuildthreads = manage_functions["rebuildthreads"].Callback
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
	if extension  == "" {
		//writer.Header().Add("Cache-Control", "max-age=5, must-revalidate")
	}
	fmt.Fprintf(writer,manage_page_buffer.String())
}

func getCurrentStaff() (string,error) {
	session_cookie := getCookie("sessiondata")
	var key string
	if session_cookie == nil {
		return "",nil
	} else {
		key = session_cookie.Value
	}

	row := db.QueryRow("SELECT `data` FROM `"+config.DBprefix+"sessions` WHERE `key` = '"+key+"';")
	current_session := new(SessionsTable)

	err := row.Scan(&current_session.Data)
	if err != nil {
		fmt.Println(err.Error())
		return "",err
	}
	return current_session.Data,nil
}

func getStaff(name string) (*StaffTable, error) {
	row := db.QueryRow("SELECT * FROM `"+config.DBprefix+"staff` WHERE `username` = '"+name+"';")
	staff_obj := new(StaffTable)
	err := row.Scan(&staff_obj.ID, &staff_obj.Username, &staff_obj.PasswordChecksum, &staff_obj.Salt, &staff_obj.Rank, &staff_obj.Boards, &staff_obj.AddedOn, &staff_obj.LastActive)
	return staff_obj,err
}

func getStaffRank() int {
	staffname,err := getCurrentStaff()
	if staffname == "" {
		return 0
	}
	if err != nil {
		return 0
	}

  	staff,err := getStaff(staffname)
  	if err != nil {
  		error_log.Print(err.Error())
  		return 0
  	}
  	return staff.Rank
}

func createSession(key string,username string, password string, request *http.Request, writer *http.ResponseWriter) int {
	//returs 0 for successful, 1 for password mismatch, and 2 for other

	if !validReferrer(*request) {
		mod_log.Print("Rejected login from possible spambot @ : "+request.RemoteAddr)
		return 2
	}
  	staff,err := getStaff(username)
  	if err != nil {
  		fmt.Println(err.Error())
  		error_log.Print(err.Error())
  		return 1
  	} else {
  		success := bcrypt.CompareHashAndPassword([]byte(staff.PasswordChecksum), []byte(password))
		if success == bcrypt.ErrMismatchedHashAndPassword {
			// password mismatch
			mod_log.Print("Failed login (password mismatch) from "+request.RemoteAddr+" at "+getSQLDateTime())
			return 1
  		} else {
			// successful login
			cookie := &http.Cookie{Name: "sessiondata", Value: key, Path: "/", Domain: config.SiteDomain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2))),MaxAge: 7200}
			http.SetCookie(*writer, cookie)
			_,err := db.Exec("INSERT INTO `"+config.DBprefix+"sessions` (`key`, `data`, `expires`) VALUES('"+key+"','"+username+"', '"+getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2)))+"');")
			if err != nil {
				error_log.Print(err.Error())
				return 2
			}
			_,err = db.Exec("UPDATE `"+config.DBprefix+"staff` SET `last_active` ='"+getSQLDateTime()+"' WHERE `username` = '"+username+"';")
			if err != nil {
				error_log.Print(err.Error())
			}
			return 0
  		}
  	}
  	return 2
}

var manage_functions = map[string]ManageFunction{
	"error": {
		Permissions: 0,
		Callback: func() (html string) {
			exitWithErrorPage(writer, "lel, internet")
			return
	}},
	"purgeeverything": {
		Permissions: 3,
		Callback: func() (html string) {
			html = "Purging everything ^_^<br />"
		 	rows,err := db.Query("SELECT `dir` FROM `"+config.DBprefix+"boards`;")
			if err != nil {
				html += err.Error()
				return
			}

			for rows.Next() {
				var board string
				err = rows.Scan(&board)
				if err != nil {
					html += err.Error()
					return
				}
    			err = deleteFolderContents(path.Join(config.DocumentRoot, board, "res"))
    			if err != nil {
    				html += err.Error()
    				return
    			}
				err = deleteFolderContents(path.Join(config.DocumentRoot, board, "src"))
    			if err != nil {
    				html += err.Error()
    				return
    			}
    			err = deleteFolderContents(path.Join(config.DocumentRoot, board, "thumb"))
    			if err != nil {
    				html += err.Error()
    				return
    			}

    			_,err = db.Exec("truncate `" + config.DBprefix + "posts`")
    			if err != nil {
    				html += err.Error() + "<br />"
    				return
    			}
    			html += "<br />Everything purged, rebuilding all<br />"
    			html += rebuildboards()+"<hr />\n"

			}
			return
	}},
	"executesql": {
		Permissions: 3,
		Callback: func() (html string) {
			statement := request.FormValue("sql")
			html = "<h1>Execute SQL statement(s)</h1><form method = \"POST\" action=\"/manage?action=executesql\">\n<textarea name=\"sql\" id=\"sql-statement\">"+statement+"</textarea>\n<input type=\"submit\" />\n</form>"
		  	if statement != "" {
		  		html += "<hr />"
				result,sqlerr := db.Exec(statement)
				fmt.Println(result)

				if sqlerr != nil {
					html += sqlerr.Error()
				} else {
					html += "Statement esecuted successfully."
				}
			}
			return
	}},
	"login":{
		Permissions: 0,
		Callback: func() (html string) {
			if getStaffRank() > 0 {
				http.Redirect(writer,&request,path.Join(config.SiteWebfolder,"manage"),http.StatusFound)
			}
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirect_action := request.FormValue("action")
			if redirect_action == ""  {
				redirect_action = "announcements"
			}
			if username == "" || password == "" {
				//assume that they haven't logged in
				html = "\t<form method=\"POST\" action=\"/manage?action=login\" class=\"loginbox\">\n" +
					"\t\t<input type=\"hidden\" name=\"redirect\" value=\""+redirect_action+"\" />\n" +
					"\t\t<input type=\"text\" name=\"username\" class=\"logindata\" /><br />\n" +
					"\t\t<input type=\"password\" name=\"password\" class=\"logindata\" /> <br />\n" +
					"\t\t<input type=\"submit\" value=\"Login\" />\n" +
					"\t</form>"
			} else {
				key := md5_sum(request.RemoteAddr+username+password+config.RandomSeed+generateSalt())[0:10]
				createSession(key,username,password,&request,&writer)
				http.Redirect(writer,&request,path.Join(config.SiteWebfolder,"/manage?action="+request.FormValue("redirect")),http.StatusFound)
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
				new_expire := time.Now().AddDate(0,0,-1)
				new_cookie := &http.Cookie{Name: "sessiondata",Value: cookie.Value,Path: "/",Domain: config.Domain,Expires: new_expire,RawExpires: new_expire.Format(time.UnixDate),MaxAge: -1,Secure: true,HttpOnly: true,Raw: "sessiondata="+key}
				http.SetCookie(writer, new_cookie)
				return "Logged out successfully"
			}
			return "wat"
	}},
	"announcements": {
		Permissions: 1,
		Callback: func() (html string) {
			html = "<h1>Announcements</h1><br />"

		  	rows,err := db.Query("SELECT `subject`,`message`,`poster`,`timestamp` FROM `"+config.DBprefix+"announcements` ORDER BY `id` DESC;")
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
					html += "<div class=\"section-block\">\n<div class=\"section-title-block\"><b>"+announcement.Subject+"</b> by "+announcement.Poster+" at "+humanReadableTime(announcement.Timestamp)+"</div>\n<div class=\"section-body\">"+announcement.Message+"\n</div></div>\n"
				}
				iterations += 1
			}

			if iterations == 0 {
				html += "No announcements"
			}
		return
	}},
	"manageserver": {
		Permissions: 3,
		Callback: func() (html string) {
			html = "<script type=\"text/javascript\">\n$jq = jQuery.noConflict();\n$jq(document).ready(function() {\n\tvar killserver_btn = $jq(\"button#killserver\");\n\n\t$jq(\"button#killserver\").click(function() {\n\t\t$jq.ajax({\n\t\t\tmethod:'GET',\n\t\t\turl:\"/manage\",\n\t\t\tdata: {\n\t\t\t\taction: 'killserver'\n\t\t\t},\n\n\t\t\tsuccess: function() {\n\t\t\t\t\n\t\t\t},\n\t\t\terror:function() {\n\t\t\t\t\n\t\t\t}\n\t\t});\n\t});\n});\n</script>" +
			"<button id=\"killserver\">Kill server</button><br />\n"

			return
	}},
	"cleanup": {
		Permissions:3,
		Callback: func() (html string) {

			return
	}},
	"getstaffjquery": {
		Permissions:0,
		Callback: func() (html string) {
			current_staff,err := getCurrentStaff()
			if err != nil {
				html = "nobody;0;"
				return
			}
			staff_rank := getStaffRank()
			if staff_rank == 0 {
				html = "nobody;0;"
				return
			}
		  	row := db.QueryRow("SELECT `rank`,`boards` FROM `"+config.DBprefix+"staff` WHERE `username` = '"+current_staff+"';")
			staff := new(StaffTable)
			err = row.Scan(&staff.Rank,&staff.Boards)
			if err != nil {
				error_log.Print(err.Error())
				html += err.Error()
				return
			}
			html = current_staff+";"+strconv.Itoa(staff.Rank)+";"+staff.Boards
			return
	}},
	"manageboards": {
		Permissions:3,
		Callback: func() (html string) {
			do := request.FormValue("do")
			var dir string
			var order int
			var title string
			var subtitle string
			var description string
			var section int
			var maximagesize int
			var firstpost int
			var maxpages int
			var defaultstyle string
			var locked bool
			var forcedanon bool
			var anonymous string
			var maxage int
			var markpage int
			var autosageafter int
			var noimagesafter int
			var maxmessagelength int
			var embedsallowed bool
			var redirecttothread bool
			var showid bool
			var compactlist bool
			var enablenofile bool
			var enablecatalog bool
			var err error

			if do != "" {
				dir = escapeString(request.FormValue("dir"))
				order_str := escapeString(request.FormValue("order"))
				order,err = strconv.Atoi(order_str)
				if err != nil {
					order = 0
				}
				title = escapeString(request.FormValue("title"))
				subtitle = escapeString(request.FormValue("subtitle"))
				description = escapeString(request.FormValue("description"))
				section_str := escapeString(request.FormValue("section"))
				section,err = strconv.Atoi(section_str)
				if err != nil {
					section = 0
				}
				maximagesize_str := escapeString(request.FormValue("maximagesize"))
				maximagesize,err = strconv.Atoi(maximagesize_str)
				if err != nil {
					maximagesize = 1024*4
				}
				firstpost_str := escapeString(request.FormValue("firstpost"))
				firstpost,err = strconv.Atoi(firstpost_str)
				if err != nil {
					firstpost = 1
				}

				maxpages_str := escapeString(request.FormValue("maxpages"))
				maxpages,err = strconv.Atoi(maxpages_str)
				if err != nil {
					maxpages = 11
				}
				defaultstyle = escapeString(request.FormValue("defaultstyle"))
				locked = (request.FormValue("locked") == "on")

				forcedanon = (request.FormValue("forcedanon") == "on")

				anonymous = escapeString(request.FormValue("anonymous"))
				maxage_str := escapeString(request.FormValue("maxage"))
				maxage,err = strconv.Atoi(maxage_str)
				if err != nil {
					maxage = 0
				}
				markpage_str := escapeString(request.FormValue("markpage"))
				markpage,err = strconv.Atoi(markpage_str)
				if err != nil {
					markpage = 9
				}
				autosageafter_str := escapeString(request.FormValue("autosageafter"))
				autosageafter,err = strconv.Atoi(autosageafter_str)
				if err != nil {
					autosageafter = 200
				}
				noimagesafter_str := escapeString(request.FormValue("noimagesafter"))
				noimagesafter,err = strconv.Atoi(noimagesafter_str)
				if err != nil {
					noimagesafter = 0
				}
				maxmessagelength_str := escapeString(request.FormValue("maxmessagelength"))
				maxmessagelength,err = strconv.Atoi(maxmessagelength_str)
				if err != nil {
					maxmessagelength = 1024*8
				}
				
				embedsallowed = (request.FormValue("embedsallowed") == "on")
				redirecttothread = (request.FormValue("redirecttothread") == "on")
				showid = (request.FormValue("showid") == "on")
				compactlist = (request.FormValue("compactlist") == "on")
				enablenofile = (request.FormValue("enablenofile") == "on")
				enablecatalog = (request.FormValue("enablecatalog") == "on")

				//actually start generating stuff
				err = os.Mkdir(path.Join(config.DocumentRoot,dir),0777)
				if err != nil {
					return err.Error()
				}
				
				err = os.Mkdir(path.Join(config.DocumentRoot,dir,"res"),0777)
				if err != nil {
					return err.Error()
				}

				err = os.Mkdir(path.Join(config.DocumentRoot,dir,"thumb"),0777)
				if err != nil {
					return err.Error()
				}
				
				err = os.Mkdir(path.Join(config.DocumentRoot,dir,"src"),0777)
				if err != nil {
					return err.Error()
				}
				_,err := db.Exec("INSERT INTO `"+config.DBprefix+"boards` (`dir`,`title`,`subtitle`,`description`,`section`,`default_style`,`no_images_after`,`embeds_allowed`) VALUES('"+dir+"','"+title+"','"+subtitle+"','"+description+"',"+section_str+",'"+defaultstyle+"',"+noimagesafter_str+",0);")
				if err != nil {
					return err.Error();
				}
			}

			html = "<h1>Manage boards</h1>\n<form action=\"/manage?action=manageboards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"existing\" /><select name=\"boardselect\">\n<option>Select board...</option>\n"
		 	rows,err := db.Query("SELECT `dir` FROM `"+config.DBprefix+"boards`;")
			if err != nil {
				html += err.Error()
				return
			}

			for rows.Next() {
				board := new(BoardsTable)
				err = rows.Scan(&board.Dir)
    			html += "<option>"+board.Dir+"</option>\n"
			}
			html += "</select> <input type=\"submit\" value=\"Edit\" /> <input type=\"submit\" value=\"Delete\" /></form><hr />"

			html += fmt.Sprintf("<h2>Create new board</h2>\n<form action=\"manage?action=manageboards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"new\" />\n<table width=\"100%%\"><tr><td>Directory</td><td><input type=\"text\" name=\"dir\" value=\"%s\"/></td></tr><tr><td>Order</td><td><input type=\"text\" name=\"order\" value=\"%d\"/></td></tr><tr><td>First post</td><td><input type=\"text\" name=\"firstpost\" value=\"%d\" /></td></tr><tr><td>Title</td><td><input type=\"text\" name=\"title\" value=\"%s\" /></td></tr><tr><td>Subtitle</td><td><input type=\"text\" name=\"subtitle\" value=\"%s\"/></td></tr><tr><td>Description</td><td><input type=\"text\" name=\"description\" value=\"%s\" /></td></tr><tr><td>Section</td><td><select name=\"section\" selected=\"%d\">\n<option value=\"none\">Select section...</option>\n",dir,order,firstpost,title,subtitle,description,section)
		 	rows,err = db.Query("SELECT `name` FROM `"+config.DBprefix+"sections` WHERE `hidden` = 0 ORDER BY `order`;")
			if err != nil {
				html += err.Error()
				return
			}

			iter := 0
			for rows.Next() {
				section := new(BoardSectionsTable)
				err = rows.Scan(&section.Name)
				html += "<option value=\""+strconv.Itoa(iter)+"\">"+section.Name+"</option>\n"
				iter += 1
			}
			html += "</select></td></tr><tr><td>Max image size</td><td><input type=\"text\" name=\"maximagesize\" value=\""+strconv.Itoa(maximagesize)+"\" /></td></tr><tr><td>Max pages</td><td><input type=\"text\" name=\"maxpages\" value=\""+strconv.Itoa(maxpages)+"\" /></td></tr><tr><td>Default style</td><td><select name=\"defaultstyle\" selected=\""+defaultstyle+"\">"
			for _, style := range config.Styles_img {
				html += "<option value=\""+style+"\">"+style+"</option>"
			}
			html += "</select></td></tr><tr><td>Locked</td><td>"
			if locked {
				html += "<input type=\"checkbox\" name=\"locked\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"locked\" />"
			}

			html += "</td></tr><tr><td>Forced anonymity</td><td>"

			if forcedanon {
				html += "<input type=\"checkbox\" name=\"forcedanon\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"forcedanon\" />"
			}

			html += "</td></tr><tr><td>Anonymous</td><td><input type=\"text\" name=\"anonymous\" value=\""+anonymous+"\" /></td></tr><tr><td>Max age</td><td><input type=\"text\" name=\"maxage\" value=\""+strconv.Itoa(maxage)+"\"/></td></tr><tr><td>Mark page</td><td><input type=\"text\" name=\"markpage\" value=\""+strconv.Itoa(markpage)+"\"/></td></tr><tr><td>Autosage after</td><td><input type=\"text\" name=\"autosageafter\" value=\""+strconv.Itoa(autosageafter)+"\"/></td></tr><tr><td>No images after</td><td><input type=\"text\" name=\"noimagesafter\" value=\""+strconv.Itoa(noimagesafter)+"\"/></td></tr><tr><td>Max message length</td><td><input type=\"text\" name=\"maxmessagelength\" value=\""+strconv.Itoa(maxmessagelength)+"\"/></td></tr><tr><td>Embeds allowed</td><td>"

			if embedsallowed {
				html += "<input type=\"checkbox\" name=\"embedsallowed\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"embedsallowed\" />"
			}

			html += "</td></tr><tr><td>Redirect to thread</td><td>"
			if redirecttothread {
				html += "<input type=\"text\" name=\"redirecttothread\" checked/>"
			} else {
				html += "<input type=\"text\" name=\"redirecttothread\" />"
			}

			html += "</td></tr><tr><td>Show ID</td><td>"

			if showid {
				html += "<input type=\"checkbox\" name=\"showid\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"showid\" />"
			}
				html += "</td></tr><tr><td>Compact list</td><td>"
			
			if compactlist {
				html += "<input type=\"checkbox\" name=\"compactlist\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"compactlist\" />"
			}

			html += "</td></tr><tr><td>Enable &quot;No file&quot; checkbox</td><td>"

			if enablenofile {
				html += "<input type=\"checkbox\" name=\"enablenofile\" checked/>"
			} else {
				html += "<input type=\"checkbox\" name=\"enablenofile\" />"
			}

			html += "</td></tr><tr><td>Enable catalog</td><td>"				
			
			if enablecatalog {
				html += "<input type=\"checkbox\" name=\"enablecatalog\" checked />"
			} else {
				html += "<input type=\"checkbox\" name=\"enablecatalog\" />"
			}

			html += "</td></tr></table><input type=\"submit\" /></form>"
			return
	}},
	"staffmenu": {
		Permissions:1,
		Callback: func() (html string) {
			rank := getStaffRank()

			html = "<a href=\"javascript:void(0)\" id=\"logout\" class=\"staffmenu-item\">Log out</a><br />\n" +
				   "<a href=\"javascript:void(0)\" id=\"announcements\" class=\"staffmenu-item\">Announcements</a><br />\n"
			if rank == 3 {
			  	html += "<b>Admin stuff</b><br />\n<a href=\"javascript:void(0)\" id=\"staff\" class=\"staffmenu-item\">Manage staff</a><br />\n" +
			  			"<a href=\"javascript:void(0)\" id=\"purgeeverything\" class=\"staffmenu-item\">Purge everything!</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"executesql\" class=\"staffmenu-item\">Execute SQL statement(s)</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildall\" class=\"staffmenu-item\">Rebuild all</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildfront\" class=\"staffmenu-item\">Rebuild front page</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildboards\" class=\"staffmenu-item\">Rebuild board pages</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildthreads\" class=\"staffmenu-item\">Rebuild threads</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"manageboards\" class=\"staffmenu-item\">Add/edit/delete boards</a><br />\n"
			}
			if rank >= 2 {
				html += "<b>Mod stuff</b><br />\n"
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
			initTemplates()
			// variables for sections table
			var section_arr []interface{}
			var board_arr []interface{}
			var front_arr []interface{}

			os.Remove(path.Join(config.DocumentRoot,"index.html"))
			front_file,err := os.OpenFile(path.Join(config.DocumentRoot,"index.html"),os.O_CREATE|os.O_RDWR,0777)
			defer func() {
				if front_file != nil {
					front_file.Close()
				}
			}()
			if err != nil {
				return err.Error()
			}

			// get boards from db and push to variables to be put in an interface
		  	rows,err := db.Query("SELECT `dir`,`title`,`subtitle`,`description`,`section` FROM `"+config.DBprefix+"boards` ORDER BY `order`;")
			if err != nil {
				error_log.Print(err.Error())
				return err.Error()
			}

			for rows.Next() {
				board := new(BoardsTable)
				board.IName = "board"
				err = rows.Scan(&board.Dir, &board.Title, &board.Subtitle, &board.Description, &board.Section)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}
			    board_arr = append(board_arr,board)
			}

			// get sections from db and push to variables to be put in an interface
		  	rows,err = db.Query("SELECT `id`,`order`,`hidden` FROM `"+config.DBprefix+"sections` ORDER BY `order`;")
			if err != nil {
				error_log.Print(err.Error())
				return err.Error()
			}
			for rows.Next() {
				section := new(BoardSectionsTable)
				section.IName = "section"
				err = rows.Scan(&section.ID, &section.Order, &section.Hidden)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}
			    section_arr = append(section_arr, section)
			}

			// get front pages
			rows,err = db.Query("SELECT * FROM `"+config.DBprefix+"frontpage`;")
			if err != nil {
				error_log.Print(err.Error())
				return err.Error()
			}

			for rows.Next() {
				frontpage := new(FrontTable)
				frontpage.IName = "front page"
				err = rows.Scan(&frontpage.ID, &frontpage.Page, &frontpage.Order, &frontpage.Subject, &frontpage.Message, &frontpage.Timestamp, &frontpage.Poster, &frontpage.Email)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}
				front_arr = append(front_arr,frontpage)
			}

		    page_data := &Wrapper{IName:"fronts", Data: front_arr}
		    board_data := &Wrapper{IName:"boards", Data: board_arr}
		    section_data := &Wrapper{IName:"sections", Data: section_arr}

		    var interfaces []interface{}
		    interfaces = append(interfaces, config)
		    interfaces = append(interfaces, page_data)
		    interfaces = append(interfaces, board_data)
		    interfaces = append(interfaces, section_data)

			wrapped := &Wrapper{IName: "frontpage",Data: interfaces}
			err = front_page_tmpl.Execute(front_file,wrapped)
			if err == nil {
				if err != nil {
					return err.Error()
				} else {
					return "Front page rebuilt successfully.<br />"
				}
			}
			return "Front page rebuilt successfully.<br />"
	}},
	"rebuildall": {
		Permissions:3,
		Callback: func() (html string) {
			html += rebuildfront()+"<hr />\n"
			html += rebuildboards()+"<hr />\n"
			html += rebuildthreads()+"\n"
			return
	}},
	"rebuildboards": {
		Permissions:3,
		Callback: func() (html string) {
			initTemplates()
			boards := getBoardArr("")
			sections := getSectionArr("")
			if boards != nil {
				if len(boards) == 0 {
					html = "No boards to build. Create a board first."
					return
				}

				for b,board := range boards {
					html += buildBoardPage(board.ID,boards,sections)
					if b < len(boards) -1 {
						html += "<br />"
					}
				}
			} else {
				html = "Failed building board pages."
			}
			return
	}},
	"rebuildthreads": {
		Permissions:3,
		Callback: func() (html string) {
			initTemplates()
			// variables for sections table
			op_posts,err := getPostArr("SELECT * FROM `ponychan_bunker_posts` WHERE `deleted_timestamp` = \""+nil_timestamp+"\" AND `parentid` = 0")
			if err != nil {
				exitWithErrorPage(writer,err.Error())
			}
			success := true
			for _,post := range op_posts {
				op_post := post.(PostTable)
				//template_friendly_op_post := TemplateFriendlyPostTable{op_post.IName, op_post.ID,op_post.BoardID,op_post.ParentID, op_post.Name, op_post.Tripcode, op_post.Email, op_post.Subject, op_post.Message, op_post.Password, op_post.Filename, op_post.FilenameOriginal, op_post.FileChecksum, op_post.Filesize, op_post.ImageW, op_post.ImageH, op_post.ThumbW, op_post.ThumbH, op_post.IP, op_post.Tag, humanReadableTime(op_post.Timestamp), op_post.Autosage, op_post.PosterAuthority, humanReadableTime(op_post.DeletedTimestamp), humanReadableTime(op_post.Bumped), op_post.Stickied, op_post.Locked, op_post.Reviewed, op_post.Sillytag}


				err := buildThread(op_post.ID,op_post.BoardID)
				if err != nil {
					success = false
					html += err.Error()+"<br />"
				} else {
					html += strconv.Itoa(op_post.ID)+" built successfully<br />"
				}
			}
			if success {
				html += "Threads rebuilt successfully."
			} else {
				html += "Thread rebuilding failed."
			}
			return
	}},
	"recentposts": {
		Permissions:1,
		Callback: func() (html string) {
			limit := request.FormValue("limit")
			if limit == "" {
				limit = "50"
			}
			html = "<h1>Recent posts</h1>\nLimit by: <select id=\"limit\"><option>25</option><option>50</option><option>100</option><option>200</option></select>\n<br />\n<table width=\"100%%\" border=\"1\">\n<colgroup><col width=\"25%%\" /><col width=\"50%%\" /><col width=\"17%%\" /></colgroup><tr><th></th><th>Message</th><th>Time</th></tr>"
		 	rows,err := db.Query("SELECT HIGH_PRIORITY `" + config.DBprefix + "boards`.`dir` AS `boardname`, `" + config.DBprefix + "posts`.`boardid` AS boardid, `" + config.DBprefix + "posts`.`id` AS id, `" + config.DBprefix + "posts`.`parentid` AS parentid, `" + config.DBprefix + "posts`.`message` AS message, `" + config.DBprefix + "posts`.`ip` AS ip, `" + config.DBprefix + "posts`.`timestamp` AS timestamp  FROM `" + config.DBprefix + "posts`, `" + config.DBprefix + "boards` WHERE `reviewed` = 0 AND `" + config.DBprefix + "posts`.`deleted_timestamp` = \""+nil_timestamp+"\"  AND `boardid` = `"+config.DBprefix+"boards`.`id` ORDER BY `timestamp` DESC LIMIT "+limit+";")
			if err != nil {
				html += "<tr><td>"+err.Error()+"</td></tr></table>"
				return
			}

			for rows.Next() {
				recentpost := new(RecentPost)
				err = rows.Scan(&recentpost.BoardName, &recentpost.BoardID, &recentpost.PostID, &recentpost.ParentID, &recentpost.Message, &recentpost.IP, &recentpost.Timestamp)
				if err != nil {
					error_log.Print(err.Error())
					return err.Error()
				}
				html += "<tr><td><b>Post:</b> <a href=\""+path.Join(config.SiteWebfolder,recentpost.BoardName,"/res/",strconv.Itoa(recentpost.ParentID)+".html#"+strconv.Itoa(recentpost.PostID))+"\">"+recentpost.BoardName+"/"+strconv.Itoa(recentpost.PostID)+"</a><br /><b>IP:</b> "+recentpost.IP+"</td><td>"+recentpost.Message+"</td><td>"+recentpost.Timestamp.Format("01/02/06, 15:04") + "</td></tr>"
			}
			html += "</table>"
			return
	}},
	"killserver": {
		Permissions:3,
		Callback: func() (html string) {
			os.Exit(0)
			return
	}},
	"staff": {
		Permissions:3,
		Callback: func() (html string) {
			//do := request.FormValue("do")
			html = "<h1>Staff</h1><br />\n" +
					"<table id=\"stafftable\" border=\"1\"><tr><td><b>Username</b></td><td><b>Rank</b></td><td><b>Boards</b></td><td><b>Added on</b></td><td><b>Action</b></td></tr>\n"
		 	rows,err := db.Query("SELECT `username`,`rank`,`boards`,`added_on` FROM `"+config.DBprefix+"staff`;")
			if err != nil {
				html += "<tr><td>"+err.Error()+"</td></tr></table>"
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
					_,err := db.Exec("INSERT INTO `"+config.DBprefix+"staff` (`username`, `password_checksum`, `rank`) VALUES('"+new_username+"','"+bcrypt_sum(new_password)+"', '"+new_rank+"');")
					if err != nil {
						exitWithErrorPage(writer,err.Error())
					}
				} else if request.FormValue("do") == "del" && request.FormValue("username") != "" {
					_,err := db.Exec("DELETE FROM `"+config.DBprefix+"staff` WHERE `username` = '"+request.FormValue("username")+"'")
					if err != nil {
						exitWithErrorPage(writer,err.Error())
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
			    html  += "<tr><td>"+staff.Username+"</td><td>"+rank+"</td><td>"+staff.Boards+"</td><td>"+humanReadableTime(staff.AddedOn)+"</td><td><a href=\"/manage?action=staff&do=del&username="+staff.Username+"\" style=\"float:right;color:red;\">X</a></td></tr>\n"
			    iter += 1
			}
			html += "</table>\n\n<hr />\n<h2>Add new staff</h2>\n\n<form action=\"manage?action=staff\" onsubmit=\"return makeNewStaff();\" method=\"POST\"><input type=\"hidden\" name=\"do\" value=\"add\" />Username: <input id=\"username\" name=\"username\" type=\"text\" /><br />\nPassword: <input id=\"password\" name=\"password\" type=\"password\" /><br />\nRank: <select id=\"rank\" name=\"rank\"><option value=\"3\">Admin</option>\n<option value=\"2\">Moderator</option>\n<option value=\"1\">Janitor</option>\n</select><br />\n<input id=\"submitnewstaff\" type=\"submit\" value=\"Add\" /></form>"

			return
	}},
}
