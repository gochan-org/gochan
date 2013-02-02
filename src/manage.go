package main

import (
	"net/http"
	"fmt"
	"os"
	"strings"
)

type ManageFunction struct {
	Permissions int // 0 -> non-staff, 1 => janitor, 2 => moderator, 3 => administrator
	Callback func() string //return string of html output?
}

func callManageFunction(w http.ResponseWriter, request *http.Request) int {
	// check if we have sufficient permissions to run this function
	//return values: 0 if successful, 1 if insufficient privelages
	form := request.Form
	action := form.Get("action")
	if action == "" {
		action = "announcements"
	}
	if _,ok := manage_functions[action]; ok {
		var manage_page_html = ""
		if getStaffRank() >= manage_functions[action].Permissions {
			global_header,_ := readFileToString("templates/global_header.html")
			manage_header,_ := readFileToString("templates/manage_header.html")
			global_footer,_ := readFileToString("templates/global_footer.html")
			manage_page_html = global_header +"\n"+ manage_header
			manage_page_html = strings.Replace(manage_page_html,"{link css}",getStyleLinks("manage"),-1)

			manage_page_html = manage_page_html + manage_functions[action].Callback()+global_footer

			fmt.Fprintf(w,manage_page_html)
			return 0

		}
	} else {
		var manage_page_html = ""
		global_header,_ := readFileToString("templates/global_header.html")
		manage_header,_ := readFileToString("templates/manage_header.html")
		global_footer,_ := readFileToString("templates/global_footer.html")
		manage_page_html = global_header +"\n"+ manage_header
		manage_page_html = strings.Replace(manage_page_html,"{link css}",getStyleLinks("manage"),-1)

		manage_page_html = manage_page_html + action + " is undefined." + global_footer

		fmt.Fprintf(w,manage_page_html)
		return 0
	}
	return 1
}

func getStaffRank() int {
	return 3
}

var manage_functions = map[string]ManageFunction{
	"initialsetup": {
		Permissions: 0,
		Callback: func() (html string) {
			html,err = readFileToString(document_root+"/index.html")
			return
	}},
	"login":{
		Permissions: 0,
		Callback: func() (html string) {
			html = "<div id=\"loginbox\">" +
				"\t<form method=\"GET\" action=\"/manage\">\n" +
				"\t\t<input type=\"hidden\" name=\"action\" value=\"login\" />\n" +
				"\t\t<input type=\"text\" name=\"username\" /><br />\n" +
				"\t\t<input type=\"password\" name=\"password\" /> <br />\n" +
				"\t\t<input type=\"submit\" value=\"Login\" />\n" +
				"\t</form>" +
			"</div>"
			return
	}},
	"announcements": {
		Permissions: 1,
		Callback: func() (html string) {
			html = "Announcements will eventually go here."

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
		Permissions: 0,
		Callback: func() (html string) {
			html = "<script type=\"text/javascript\">\n$jq = jQuery.noConflict();\n$jq(document).ready(function() {\n\tvar killserver_btn = $jq(\"button#killserver\");\n\n\t$jq(\"button#killserver\").click(function() {\n\t\t$jq.ajax({\n\t\t\tmethod:'GET',\n\t\t\turl:\"/manage\",\n\t\t\tdata: {\n\t\t\t\taction: 'killserver'\n\t\t\t},\n\n\t\t\tsuccess: function() {\n\t\t\t\t\n\t\t\t},\n\t\t\terror:function() {\n\t\t\t\t\n\t\t\t}\n\t\t});\n\t});\n});\n</script>" +
			"<button id=\"killserver\">Kill server</button><br />\n"
			return
	}},
	"rebuildall": {
		Permissions:3,
		Callback: func() (html string) {

			return
	}},
	"recentposts": {
		Permissions:1,
		Callback: func() (html string) {
			html = "<h1>Recent posts</h1>\n<table>\n<tr></tr>   "

			return
	}},
	"killserver": {
		Permissions:3,
		Callback: func() (html string) {
			os.Exit(0)
			return
	}}}