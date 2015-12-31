var Staff = function(name,rank,boards) {
	this.name = name;
	this.rank = rank;
	this.boards = boards;
}

function addStaffButtons() {
	$jq("input#delete-password").remove();
	$jq("input[value=Delete]").after("<input type=\"submit\" name=\"Ban\" value=\"Ban\" onclick=\"alert('Bans not yet implemented'); return false;\"  />")
}

function banPage() {
	switch(getArg("type")) {
		case "ip":
			$jq("div#.ban-type-div#ip").css({"display":"inline"})
			$jq("div#.ban-type-div#name").css({"display":"none"})
			$jq("input[type=hidden][name=type]").attr("value", "ip")
			break;
		case "name-tripcode":
			$jq("div#.ban-type-div#ip").css({"display":"none"})
			$jq("div#.ban-type-div#name").css({"display":"inline"})
			$jq("input[type=hidden][name=type]").attr("value", "name/tripcode")
			break;
	}

	$jq("select#ban-type").bind("change", function (e){
		var new_selection = this.value;
		switch(new_selection) {
			case "Single IP/IP range":
				$jq("div#ip.ban-type-div").css({"display":"inline"})
				$jq("div#name.ban-type-div").css({"display":"none"})
				$jq("input[type=hidden][name=type]").attr("value", "ip")
				break;
			case "Name/tripcode":
				$jq("div#ip.ban-type-div").css({"display":"none"})
				$jq("div#name.ban-type-div").css({"display":"inline"});
				$jq("input[type=hidden][name=type]").attr("value", "name-tripcode")
				break;
		}
	});
	$jq("input[type=checkbox]#allboards").bind("change", function() {
		var allboards_check = this;
		$jq("input[type=checkbox].board-check").each(function() {
			this.checked = allboards_check.checked;
		});
	});
	$jq("div.duration-select").html(
		"<select class=\"duration-months\">" +
			"<option>Months...</option>" +
		"</select>" +
		"<select class=\"duration-days\">" +
			"<option>Days...</option>" +
		"</select>" +
		"<select class=\"duration-hours\">" +
			"<option>Hours...</option>" +
		"</select>" +
		"<select class=\"duration-minutes\">" +
			"<option>Minutes...</option>" +
		"</select>"
	);
	var months_html = "";
	var i;
	for(i = 0; i < 49; i++) {
		months_html += "<option>" + i + "</option>";
	}

	var days_html = "";
	for(i = 0; i < 33; i++) {
		days_html += "<option>" + i + "</option>";
	}

	var hours_html = "";
	for(i = 0; i < 25; i++) {
		hours_html += "<option>" + i + "</option>";
	}

	var minutes_html = "";
	for(i = 0; i < 61; i++) {
		minutes_html += "<option>" + i + "</option>";
	}
	$jq("select.duration-months").append(months_html);
	$jq("select.duration-days").append(days_html);
	$jq("select.duration-hours").append(hours_html);
	$jq("select.duration-minutes").append(minutes_html);
	/*if(watermark) {
		$jq("input[type=text][name=ip]").watermark("IP address");
		$jq("input[type=text][name=ip]").prev().remove();
		$jq($jq("div#reason-staffnote input")[0]).prev().remove();
		$jq($jq("div#reason-staffnote input")[0]).watermark("Reason");
		$jq($jq("div#reason-staffnote input")[1]).watermark("Staff note");
		$jq($jq("div#reason-staffnote input")[1]).prev().remove();
	}*/
}

function getManagePage() {

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
			url: webroot+"manage?action=managestaff",
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
		dataType:"xml",
		cache: true,
		async:false,
		success: function(result) {
			var return_jq = $jq(result);
			var text = $jq($jq(return_jq.children()[0]).children()[1]).text();
			var return_data = text.trim().split(";");
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
		dataType:"xml",
		async:false,

		success: function(result) {
			var result_body = $jq(result).find("body");
			var header = $jq(result).find("h1");
			var header_text = header.text();
			header.remove()
			if(header_text == "") header_text = "Manage";
			showLightBox(header_text,result_body.html());
		},
		error: function(result) {
			var responsetext = result.responseText
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

$jq(document).ready(function() {
	/*if(location.pathname.indexOf("/manage" == location.pathname.length -7)) {
		if(getArg("action") == "banuser") {
			banPage();
		}
	}*/
});
