package main

import (
	"bytes"
	"code.google.com/p/go.crypto/bcrypt"
	"fmt"
	"io/ioutil"
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
	fmt.Fprintf(writer,manage_page_buffer.String())
}

func getCurrentStaff() string {
	session_cookie := getCookie("sessiondata")
	var key string
	if session_cookie == nil {
		return ""
	} else {
		key = session_cookie.Value
	}

	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"sessions` WHERE `key` = '"+key+"';")
	if err != nil {
		error_log.Write(err.Error())
		return ""
	}

	rows, err := results.GetRows()
    if err != nil {
		error_log.Write(err.Error())
		return ""
    }
	if len(rows) > 0 {
		for  _, row := range rows {
			return string(row[2].([]byte))
		}
	} else {
		//session key doesn't exist in db
		return ""
	}
	return ""
}

func getStaffRank() int {
	var key string
	var staffname string

	db.Start("USE `"+config.DBname+"`")
	session_cookie := getCookie("sessiondata")
	if session_cookie == nil {
		return 0
	} else {
		key = session_cookie.Value
	}

  	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"sessions` WHERE `key` = '"+key+"';")
	if err != nil {
		error_log.Write(err.Error())
		return 0
	}

	rows, err := results.GetRows()
    if err != nil {
		error_log.Write(err.Error())
		return 1
    }
	if len(rows) > 0 {
		for  _, row := range rows {
			staffname = string(row[2].([]byte))
			break
		}
	} else {
		//session key doesn't exist in db
		return 0
	}

  	results,err = db.Start("SELECT * FROM `"+config.DBprefix+"staff` WHERE `username` = '"+staffname+"';")
	if err != nil {
		error_log.Write(err.Error())
		return 0
	}

	rows, err = results.GetRows()
    if err != nil {
		error_log.Write(err.Error())
		return 1
    }
	if len(rows) > 0 {
		for  _, row := range rows {
			rank,rerr := strconv.Atoi(string(row[4].([]byte)))
			if rerr == nil {
				return rank
			} else {
				return 0
			}
		}
	}
	return 0
}

func createSession(key string,username string, password string, request *http.Request, writer *http.ResponseWriter) int {
	//returs 0 for successful, 1 for password mismatch, and 2 for other
	//db.Start("USE `"+config.DBname+"`;")
  	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"staff` WHERE `username` = '"+username+"';")

	if err != nil {
		error_log.Write(err.Error())
		return 2
	} else {
		rows, err := results.GetRows()
	    if err != nil {
			error_log.Write(err.Error())
			return 1
	    }
		if len(rows) > 0 {
			for _, row := range rows {
	    		success := bcrypt.CompareHashAndPassword(row[2].([]byte), []byte(password))
	    		if success == nil {
	    			// successful login
					cookie := &http.Cookie{Name: "sessiondata", Value: key, Path: "/", Domain:config.Domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2)))}
	    			http.SetCookie(*writer, cookie)
					_,err := db.Start("INSERT INTO `"+config.DBprefix+"sessions` (`key`, `data`, `expires`) VALUES('"+key+"','"+username+"', '"+getSpecificSQLDateTime(time.Now().Add(time.Duration(time.Hour*2)))+"');")
					if err != nil {
						error_log.Write(err.Error())
						return 2
					}
					_,err = db.Start("UPDATE `"+config.DBprefix+"staff` SET `last_active` ='"+getSQLDateTime()+"' WHERE `username` = '"+username+"';")
					if err != nil {
						error_log.Write(err.Error())
					}

					return 0
	    		} else if success == bcrypt.ErrMismatchedHashAndPassword {
	    			// password mismatch
	    			_,err := db.Start("INSERT `"+config.DBprefix+"loginattempts` (`ip`,`timestamp`) VALUES('"+request.RemoteAddr+"','"+getSQLDateTime()+"');")
	    			if err != nil {
	    				error_log.Write(err.Error())
	    			}
	    			return 1
	    		}
			}
		} else {
			//username doesn't exist
			return 1
		}
	}
	return 1
}

var manage_functions = map[string]ManageFunction{
	"initialsetup": {
		Permissions: 0,
		Callback: func() string {
			html,_ := ioutil.ReadFile(config.DocumentRoot+"/index.html")
			return string(html)
	}},
	"error": {
		Permissions: 0,
		Callback: func() (html string) {
			exitWithErrorPage(writer, "lel, internet")
			return
	}},
	"executesql": {
		Permissions: 3,
		Callback: func() (html string) {
			statement := request.FormValue("sql")
			html = "<h1>Execute SQL statement(s)</h1><form method = \"POST\" action=\"/manage?action=executesql\">\n<textarea name=\"sql\" id=\"sql-statement\">"+statement+"</textarea>\n<input type=\"submit\" />\n</form>"
		  	if statement != "" {
		  		html += "<hr />"
			  	_,sqlerr := db.Start(statement)
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
			username := request.FormValue("username")
			password := request.FormValue("password")
			redirect_action := request.FormValue("action")
			if redirect_action == ""  {
				redirect_action = "announcements"
			}
			fmt.Println(redirect_action)
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
				redirect(path.Join(config.SiteWebfolder,"/manage?action="+request.FormValue("redirect")))
			}
			return
	}},
	"logout": {
		Permissions: 1,
		Callback: func() (html string) {

			return
	}},
	"announcements": {
		Permissions: 1,
		Callback: func() (html string) {
			html = "<h1>Announcements</h1><br />"

		  	results,err := db.Start("SELECT `subject`,`message`,`poster`,`timestamp` FROM `"+config.DBprefix+"announcements` ORDER BY `id` DESC;")
			if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
			}

			rows, err := results.GetRows()
		    if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
		    }
			if len(rows) > 0 {
				for  _, row := range rows {
					html += "<div class=\"section-block\">\n<div class=\"section-title-block\"><b>"+string(row[0].([]byte))+"</b> by "+string(row[2].([]byte))+" at "+string(row[3].([]byte))+"</div>\n<div class=\"section-body\">"+string(row[1].([]byte))+"\n</div></div>\n"
				}
			} else {
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
			current_staff := getCurrentStaff()
			staff_rank := getStaffRank()
			if staff_rank == 0 {
				html = "nobody;0;"
				return
			}
			staff_boards := ""
		  	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"staff`;")
			if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
			}

			rows, err := results.GetRows()
		    if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
		    }
			if len(rows) > 0 {
				for  _, row := range rows {
					staff_boards = string(row[5].([]byte))
				}
			} else {
				// fuck you, I'm Spiderman.
			}
			html = current_staff+";"+strconv.Itoa(staff_rank)+";"+staff_boards
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
				dir = db.Escape(request.FormValue("dir"))
				order_str := db.Escape(request.FormValue("order"))
				order,err = strconv.Atoi(order_str)
				if err != nil {
					order = 0
				}
				title = db.Escape(request.FormValue("title"))
				subtitle = db.Escape(request.FormValue("subtitle"))
				description = db.Escape(request.FormValue("description"))
				section_str := db.Escape(request.FormValue("section"))
				section,err = strconv.Atoi(section_str)
				if err != nil {
					section = 0
				}
				maximagesize_str := db.Escape(request.FormValue("maximagesize"))
				maximagesize,err = strconv.Atoi(maximagesize_str)
				if err != nil {
					maximagesize = 1024*4
				}
				firstpost_str := db.Escape(request.FormValue("firstpost"))
				firstpost,err = strconv.Atoi(firstpost_str)
				if err != nil {
					firstpost = 1
				}

				maxpages_str := db.Escape(request.FormValue("maxpages"))
				maxpages,err = strconv.Atoi(maxpages_str)
				if err != nil {
					maxpages = 11
				}
				defaultstyle = db.Escape(request.FormValue("defaultstyle"))
				locked = (request.FormValue("locked") == "on")

				forcedanon = (request.FormValue("forcedanon") == "on")

				anonymous = db.Escape(request.FormValue("anonymous"))
				maxage_str := db.Escape(request.FormValue("maxage"))
				maxage,err = strconv.Atoi(maxage_str)
				if err != nil {
					maxage = 0
				}
				markpage_str := db.Escape(request.FormValue("markpage"))
				markpage,err = strconv.Atoi(markpage_str)
				if err != nil {
					markpage = 9
				}
				autosageafter_str := db.Escape(request.FormValue("autosageafter"))
				autosageafter,err = strconv.Atoi(autosageafter_str)
				if err != nil {
					autosageafter = 200
				}
				noimagesafter_str := db.Escape(request.FormValue("noimagesafter"))
				noimagesafter,err = strconv.Atoi(noimagesafter_str)
				if err != nil {
					noimagesafter = 0
				}
				maxmessagelength_str := db.Escape(request.FormValue("maxmessagelength"))
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
				_,err := db.Start("INSERT INTO `"+config.DBprefix+"boards` (`dir`,`title`,`subtitle`,`description`,`section`,`default_style`,`no_images_after`,`embeds_allowed`) VALUES('"+dir+"','"+title+"','"+subtitle+"','"+description+"',"+section_str+",'"+defaultstyle+"',"+noimagesafter_str+",0);")
				if err != nil {
					return err.Error();
				}
			}

			html = "<h1>Manage boards</h1>\n<form action=\"/manage?action=manageboards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"existing\" /><select name=\"boardselect\">\n<option>Select board...</option>\n"
			db.Start("USE `"+config.DBname+"`;")
		 	results,err := db.Start("SELECT `dir` FROM `"+config.DBprefix+"boards`;")
			if err != nil {
				html += err.Error()
				return
			}

			rows, err := results.GetRows()
		    if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
		    }
			if len(rows) > 0 {
				for  _, row := range rows {
    				html += "<option>"+string(row[0].([]byte))+"</option>\n"
				}
			}
			html += "</select> <input type=\"submit\" value=\"Edit\" /> <input type=\"submit\" value=\"Delete\" /></form><hr />"

			html += "<h2>Create new board</h2>\n<form action=\"manage?action=manageboards\" method=\"POST\">\n<input type=\"hidden\" name=\"do\" value=\"new\" />\n<table width=\"100%%\"><tr><td>Directory</td><td><input type=\"text\" name=\"dir\" value=\""+dir+"\"/></td></tr><tr><td>Order</td><td><input type=\"text\" name=\"order\" value=\""+strconv.Itoa(order)+"\"/></td></tr><tr><td>First post</td><td><input type=\"text\" name=\"firstpost\" value=\""+strconv.Itoa(firstpost)+"\" /></td></tr><tr><td>Title</td><td><input type=\"text\" name=\"title\" value=\""+title+"\" /></td></tr><tr><td>Subtitle</td><td><input type=\"text\" name=\"subtitle\" value=\""+subtitle+"\"/></td></tr><tr><td>Description</td><td><input type=\"text\" name=\"description\" value=\""+description+"\" /></td></tr><tr><td>Section</td><td><select name=\"section\" selected=\""+strconv.Itoa(section)+"\">\n<option value=\"none\">Select section...</option>\n"
		 	results,err = db.Start("SELECT `name` FROM `"+config.DBprefix+"sections` WHERE `hidden` = 0 ORDER BY `order`;")
			if err != nil {
				html += err.Error()
				return
			}

			rows, err = results.GetRows()
		    if err != nil {
				error_log.Write(err.Error())
				html += err.Error()
				return
		    }
			if len(rows) > 0 {
				for row_num, row := range rows {
					html += "<option value=\""+strconv.Itoa(row_num)+"\">"+string(row[0].([]byte))+"</option>\n"
				}
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
					  	"<a href=\"javascript:void(0)\" id=\"executesql\" class=\"staffmenu-item\">Execute SQL statement(s)</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildfront\" class=\"staffmenu-item\">Rebuild front page</a><br />\n" +
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
			var section_id int
			var section_order int
			var section_hidden bool
			var section_arr []interface{}

			// variables for board
			var board_dir string
			var board_title string
			var board_subtitle string
			var board_description string
			var board_section int
			var board_arr []interface{}

			// variables for frontpage table
			var front_page int
			var front_order int
			var front_subject string
			var front_message string
			var front_timestamp string
			var front_poster string
			var front_email string
			var front_arr []interface{}

			os.Remove("html/index.html")
			front_file,err := os.OpenFile("html/index.html",os.O_CREATE|os.O_RDWR,0777)
			defer func() {
				front_file.Close()
			}()
			if err != nil {
				return err.Error()
			}

			// get boards from db and push to variables to be put in an interface
		  	results,err := db.Start("SELECT `dir`,`title`,`subtitle`,`description`,`section` FROM `"+config.DBprefix+"boards` ORDER BY `order`;")
			if err != nil {
				error_log.Write(err.Error())
				return err.Error()
			}
			rows,err := results.GetRows()
			if err != nil {
				error_log.Write(err.Error())
				return err.Error()
			}

			for _,row := range rows {
				board_dir = string(row[0].([]byte))
				board_title = string(row[1].([]byte))
				board_subtitle = string(row[2].([]byte))
				board_description = string(row[3].([]byte))
				board_section,_ = strconv.Atoi(string(row[4].([]byte)))
			    board_arr = append(board_arr,BoardsTable{IName:"board", Dir:board_dir, Title:board_title, Subtitle:board_subtitle, Description:board_description, Section:board_section})
			}

			// get sections from db and push to variables to be put in an interface
		  	results,err = db.Start("SELECT `id`,`order`,`hidden` FROM `"+config.DBprefix+"sections` ORDER BY `order`;")
			if err != nil {
				error_log.Write(err.Error())
				return err.Error()
			}
			rows,err = results.GetRows()
			if err != nil {
				error_log.Write(err.Error())
				return err.Error()
			}

			for _,row := range rows {
				section_id,_ = strconv.Atoi(string(row[0].([]byte)))
				section_order,_ = strconv.Atoi(string(row[1].([]byte)))
				b := string(row[2].([]byte))
				if b == "1" {
					section_hidden = true
				} else {
					section_hidden = false
				}
			    section_arr = append(section_arr, BoardSectionsTable{IName: "section", ID: section_id, Order: section_order, Hidden: section_hidden})
			}

			// get front pages
			results,err = db.Start("SELECT * FROM `"+config.DBprefix+"frontpage`;")
			if err != nil {
				error_log.Write(err.Error())
				return err.Error()
			}

			rows, err = results.GetRows()
		    if err != nil {
				error_log.Write(err.Error())
				return err.Error()
		    }
			if len(rows) > 0 {
				for row_num, row := range rows {
	    			front_page,_ = strconv.Atoi(string(row[1].([]byte)))
	    			front_order,_ = strconv.Atoi(string(row[2].([]byte)))
	    			front_subject = string(row[3].([]byte))
	    			front_message = string(row[4].([]byte))
	    			front_timestamp = string(row[5].([]byte))
	    			front_poster = string(row[6].([]byte))
	    			front_email = string(row[7].([]byte))
					front_arr = append(front_arr,FrontTable{IName:"front page", ID:row_num, Page: front_page, Order: front_order, Subject: front_subject, Message: front_message, Timestamp: front_timestamp, Poster: front_poster, Email: front_email})
				}
			} else {
				// no front pages
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
			//html += manage_functions["rebuildfront"].Callback()+"\n<br />\n"
			return
	}},
	"rebuildthreads": {
		Permissions:3,
		Callback: func() (html string) {
			initTemplates()
			// variables for sections table
			op_posts := getPostArr("`deleted_timestamp` IS NULL AND `parentid` = 0")
			success := true
			for _,post := range op_posts {
				op_post := post.(PostTable)
				if buildThread(op_post) != nil {
					success = false
				}
			}
			if success {
				html = "Threads rebuilt successfully."
			} else {
				html = "Thread rebuilding failed somewhere (eventually we'll print out all the rebuilt threads."
			}
			return
	}},
	"recentposts": {
		Permissions:1,
		Callback: func() (html string) {
			html = "<h1>Recent posts</h1>\n<table style=\"border:2px solid;\">\n<tr><td>bleh</td><td>bleh bleh</td></tr>" +
			"</table>"
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
					"<table border=\"1\"><tr><td><b>Username</b></td><td><b>Rank</b></td><td><b>Boards</b></td><td><b>Added on</b></td><td><b>Action</b></td></tr>\n"
			db.Start("USE `"+config.DBname+"`;")
		 	results,err := db.Start("SELECT `username`,`rank`,`boards`,`added_on` FROM `"+config.DBprefix+"staff`;")
			if err != nil {
				html += "<tr><td>"+err.Error()+"</td></tr></table>"
				return
			}

			rows, err := results.GetRows()
	        if err != nil {
				html += "<tr><td>"+err.Error()+"</td></tr></table>"
				return
	        }
			for row_num, row := range rows {
	    		rank := string(row[1].([]byte))
	    		if rank == "3" {
	    			rank = "admin"
	    		} else if rank == "2" {
	    			rank = "mod"
	    		} else if rank == "1" {
	    			rank = "janitor"
	    		}
			    html  += "<tr><td>"+string(row[0].([]byte))+"</td><td>"+rank+"</td><td>"+string(row[2].([]byte))+"</td><td>"+string(row[3].([]byte))+"</td><td><a href=\"action=staff%%26do=del%%26index="+strconv.Itoa(row_num)+"\" style=\"float:right;color:red;\">X</a></td></tr>\n"
			}
			html += "</table>"
			return
	}},
}