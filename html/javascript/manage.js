var Staff = function(name,rank,boards) {
	this.name = name;
	this.rank = rank;
	this.boards = boards;
}

function addStaffButtons() {
	$jq("input#delete-password").remove();
	$jq("input[value=Delete]").after("<input type=\"submit\" name=\"Ban\" value=\"Ban\" onclick=\"banSelectedPost(); return false;\"  />")
}

function getManagePage() {

}

function banSelectedPost() {
	var board_dir_arr = location.pathname.split("/");
	if(board_dir_arr.length < 2) return;
	var board_dir = board_dir_arr[1];
	var checks = $jq("input[type=checkbox]");
	if(checks.length == 0) {
		alert("No posts selected");
		return false;
	}
	var post_id = 0;
	for(var i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") == 0) {
			post_id = checks[i].id.replace("check", "");
			break;
		}
	}
	window.location = webroot + "manage?action=bans&dir=" + board_dir + "&postid=" + post_id;
}

function makeNewStaff() {
	var on_manage_page = false; // true to submit, false for ajax;
	if(window.location.pathname == "/manage") {
		on_manage_page = true;
	} else {
		var username_txt = $jq("input#username").val();
		var password_txt = $jq("input#password").val();
		var rank_sel = $jq("select#rank").val();
		$jq.ajax({
			method: 'POST',
			url: webroot+"manage?action=staff",
			data: {
				"do":"add",
				username: username_txt,
				password: password_txt,
				rank: rank_sel,
				boards: "all"
			},
			cache: false,
			async:true,
			success: function(result) {
				var rank_str;
				switch(rank_sel) {
					case "3":
						rank_str = "admin";
						break;
					case "2":
						rank_str = "mod";
						break;
					case "1":
						rank_str = "janitor";
						break;
				}
				$jq("table#stafftable tr:last").after("<tr><td>"+username_txt+"</td><td>"+rank_str+"</td><td>all</td><td>now</td><td></td></tr>")
			},
			error: function() {
				alert("Something went wrong...")
			}
		});
	}
	return on_manage_page;
}

function getStaff() {
	var s;
	$jq.ajax({
		method: 'GET',
		url: webroot+"manage",
		data: {
			action: 'getstaffjquery',
		},
		dataType:"text",
		cache: true,
		async:false,
		success: function(result) {
			var return_data = result.trim().split(";");
			s = new Staff(return_data[0],return_data[1],return_data[2].split(","));
		},
		error: function() {
			s = new Staff("nobody","0","");
		}
	});
	return s;
}

function getStaffMenuHTML() {
	var s = "<ul class=\"staffmenu\">";
	$jq.ajax({
		method: 'GET',
		url: webroot+"manage",
		data: {
			action: 'staffmenu',
		},
		dataType:"text",
		cache: true,
		async:false,
		success: function(result) {
			var lines = result.substring(result.indexOf("body>")+5,result.indexOf("</body")).trim().split("\n")
			var num_lines = lines.length;
			for(var l = 0; l < num_lines; l++) {
				if(lines[l] != "") {
					if(lines[l].indexOf("<a href=") > -1) {
						s += lines[l].substr(0,lines[l].indexOf("\">")+2)+"<li>"+$jq(lines[l]).text()+"</li></a>";
					} else {
						s += "<li>"+lines[l]+"</li>";
					}
				}
			}
		},
		error: function() {
			s = "Something went wrong :/";
		}
	});
	return s+"</ul>";
}

function openStaffLightBox(action_url) {
	$jq.ajax({
		method: 'GET',
		url: webroot+"manage",
		data: {
			action: action_url,
		},
		dataType:"html",
		async:false,

		success: function(result) {
			var body = '<div id="body-mock">' + result.replace(/^[\s\S]*<body.*?>|<\/body>[\s\S]*$/ig, '') + '</div>';
			var $body = $jq(body);
			var header = $body.find("h1");
			var header_text = header.text();
			header.remove();
			if(header_text == "") header_text = "Manage";
			showLightBox(header_text,$body.html());
		},
		error: function(result) {
			var responsetext = result.responseText;
			header = responsetext.substring(responsetext.indexOf("<h1>")+4,responsetext.indexOf("</h1>"))

			responsetext = responsetext.substring(responsetext.indexOf("</h1>") + 5, responsetext.indexOf("</body>"));
			if(header == "") {
				showLightBox("Manage",responsetext);
			} else {
				showLightBox(header,responsetext);
			}
		}
	});
}

/* $jq(document).ready(function() {
	
}); */
