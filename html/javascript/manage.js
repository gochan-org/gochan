var Staff = function(name,rank,boards) {
	this.name = name;
	this.rank = rank;
	this.boards = boards;
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
	var s = "<ul class=\"boardmenu\">";
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
			showLightBox("Manage","Something went wrong :(");
		}
	});
}

