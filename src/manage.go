package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	_ "code.google.com/p/go.crypto/bcrypt"
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

	manage_page_html := ""

	if action == ""  {
		action = "announcements"
	}
	if staff_rank == 0 {
		action = "login"
	}

	global_header,err := getTemplateAsString(*global_header_tmpl)
	if err != nil {
		fmt.Fprintf(writer,err.Error())
	} else {
		fmt.Fprintf(writer,global_header)
	}

	manage_header,err := getTemplateAsString(*manage_header_tmpl)
	if err != nil {
		fmt.Fprintf(writer,err.Error())
	} else {
		fmt.Fprintf(writer,manage_header)
	}

	if _,ok := manage_functions[action]; ok {
		if staff_rank >= manage_functions[action].Permissions {
			manage_page_html += manage_functions[action].Callback()
			fmt.Fprintf(writer,manage_page_html)

		} else {
			manage_page_html = manage_page_html + action + " is undefined."
			fmt.Fprintf(writer,manage_page_html)
		}
	} else {
		manage_page_html = manage_page_html + action + " is undefined."
		fmt.Fprintf(writer,manage_page_html)
	}
	fmt.Fprintf(writer,"\n</body>\n</html>")
}

func getStaffRank() int {
	return 3
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

	for {
	    row, err := results.GetRow()
        if err != nil {
        	error_log.Write(err.Error())
        }

        if row == nil {
            break
        }

	    for col_num, col := range row {
			if col_num == 2 {
				staffname = string(col.([]byte))
			}
	    }
	}

  	results,err = db.Start("SELECT * FROM `"+config.DBprefix+"staff` WHERE `username` = '"+staffname+"';")
	if err != nil {
		error_log.Write(err.Error())
		return 0
	}

	for {
	    row, err := results.GetRow()
        if err != nil {
        	error_log.Write(err.Error())
        	return 0
        }

        if row == nil {
            break
        }

	    for col_num, col := range row {
			if col_num == 4 {
				rank,rerr := strconv.Atoi(string(col.([]byte)))
				if rerr == nil {
					return rank
				} else {
					return 0
				}
			}
	    }
	}
	return 0
}

func createSession(key string,username string, password string) bool {
	//sum := bcrypt_sum(password)
  	rows,_,err := db.Query("SELECT `password_checksum` FROM `"+config.DBprefix+"staff`")

	if err != nil {
		error_log.Write(err.Error())
		fmt.Println("nope 1")
		return false
	} else {


		if len(rows) > 0 {
			_,err := db.Start(" INSERT INTO `"+config.DBprefix+"sessions` (`key`, `data`, `expires`) VALUES('"+key+"','"+username+"', '2023-17-04 16:21:01');")
			if err != nil {
				fmt.Println("Initial setup failed.")
				error_log.Write(err.Error())
			}
		} else {
			fmt.Println("nope 2")
			return false
		}
	}
	fmt.Println("dafuq?")
	return false
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
			exitWithErrorPage("lel, internet")
			return
	}},
	"login":{
		Permissions: 0,
		Callback: func() (html string) {
			username := request.FormValue("username")
			password := request.FormValue("password")

			if username == "" || password == "" {
				//assume that they haven't logged in
				html = "\t<form method=\"POST\" action=\"/manage?action=login\" class=\"loginbox\">\n" +
					//"\t\t<input type=\"hidden\" name=\"action\" value=\"login\" />\n" +
					"\t\t<input type=\"text\" name=\"username\" class=\"logindata\" /><br />\n" +
					"\t\t<input type=\"password\" name=\"password\" class=\"logindata\" /> <br />\n" +
					"\t\t<input type=\"submit\" value=\"Login\" />\n" +
					"\t</form>"
			} else {
				key := md5_sum(request.RemoteAddr+username+password+config.RandomSeed+generateSalt())
				createSession(key,username,password)
				//check db for valid login
			  	/*
			  	password_bcrypt = bcrypt_encode(password)
			  	results,err := db.Query("SELECT `username`,`password`, FROM `"+config.DBprefix+"staff")
				if err != nil {
					error_log.Write(err.Error())
				}
				var entry StaffTable
				for results.Next() {
					err = results.Scan(&entry.username,&entry.password)
					if entry.username == username && entry.password == password_bcrypt {
						//authenticated

					}
					if err !=  nil { error_log.write(err.Error()) }
				}
				*/
			}
			return
	}},
	"announcements": {
		Permissions: 1,
		Callback: func() (html string) {
			html = "<h1>Announcements</h1><br />" +
				"Announcements will eventually go here."

		  	/*results,err := db.Query("SELECT * FROM `"+db_prefix+"announcements")
			if err != nil {
				error_log.Write(err.Error())
			}
			var entry ModPageAnnouncementsTable
			for results.Next() {
				err = results.Scan(&entry.id,&entry.parentid,&entry.subject,&entry.postedat,&entry.postedby,&entry.message)
				//if err !=  nil { panic(err) }
			}*/
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
			html = "Luna;3;test1,test2"
			return
	}},
	"staffmenu": {
		Permissions:1,
		Callback: func() (html string) {
			rank := getStaffRank()

			html = "<a href=\"javascript:void(0)\" id=\"logout\" class=\"staffmenu-item\">Log out</a><br />\n" +
				   "<a href=\"javascript:void(0)\" id=\"announcements\" class=\"staffmenu-item\">Announcements</a><br />\n"
			if rank == 3 {
			  	html += "<a href=\"javascript:void(0)\" id=\"staff\" class=\"staffmenu-item\">Manage staff</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"rebuildfront\" class=\"staffmenu-item\">Rebuild front page</a><br />\n" +
					  	"<a href=\"javascript:void(0)\" id=\"manageboards\" class=\"staffmenu-item\">Add/edit/delete boards</a><br />\n"
			}

			if rank > 0 {
				html += "<a href=\"javascript:void(0)\" id=\"recentimages\" class=\"staffmenu-item\">Recently uploaded images</a><br />\n" +
						"<a href=\"javascript:void(0)\" id=\"recentposts\" class=\"staffmenu-item\">Recent posts</a><br />\n" +
						"<a href=\"javascript:void(0)\" id=\"searchip\" class=\"staffmenu-item\">Search posts by IP</a><br />\n"
			}

			return
	}},
	"rebuildfront": {
		Permissions: 3,
		Callback: func() (html string) {
			f,err := os.OpenFile("html/index.html",os.O_RDWR|os.O_CREATE,0777)
			if err != nil {
				return err.Error()
			} else {
				err = front_page_tmpl.Execute(f,config)
				if err != nil {
					return err.Error()
				}
			}
			return "Front page rebuilt successfully.<br />"
	}},
	"rebuildall": {
		Permissions:3,
		Callback: func() (html string) {
			initTemplates()
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

			row_num := 0
			for {
			    row, err := results.GetRow()
		        if err != nil {
					html += "<tr><td>"+err.Error()+"</td></tr></table>"
					return
		        }

		        if row == nil {
		            break
		        }
		        html  += "<tr>"
			    for col_num, col := range row {
			    	if col_num == 1 {
			    		rank := string(col.([]byte))
			    		if rank == "3" {
			    			rank = "admin"
			    		} else if rank == "2" {
			    			rank = "mod"
			    		} else if rank == "1" {
			    			rank = "janitor"
			    		}
			    		html += "<td>"+rank+"</td>"	
			    	} else {
			    		html += "<td>"+string(col.([]byte))+"</td>"
			    	}
				}
				
				html += "<td><a href=\"action=staff%26do=del%26index="+strconv.Itoa(row_num)+"\" style=\"float:right;color:red;\">X</a></td></tr>\n"
			    
			}
			html += "</table>"
			return
	}},
}