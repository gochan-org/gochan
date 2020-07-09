import { showLightBox } from "./lightbox"

export class Staff {
	static makeNew() {
		let onManagePage = false; // true to submit, false for ajax;
		if(window.location.pathname == "/manage") {
			onManagePage = true;
		} else {
			let usernameTxt = $("input#username").val();
			let passwordTxt = $("input#password").val();
			let rankSel = $("select#rank").val();
			$.ajax({
				method: 'POST',
				url: `${webroot}manage?action=staff`,
				data: {
					do:"add",
					username: usernameTxt,
					password: passwordTxt,
					rank: rankSel,
					boards: "all"
				},
				cache: false,
				async:true,
				success: function(result) {
					let rankStr = "";
					switch(rankSel) {
						case "3":
							rankStr = "admin";
							break;
						case "2":
							rankStr = "mod";
							break;
						case "1":
							rankStr = "janitor";
							break;
					}
					$("table#stafftable tr:last").after(`<tr><td>${usernameTxt}</td><td>${rankStr}</td><td>all</td><td>now</td><td></td></tr>`);
				},
				error: function() {
					alert("Something went wrong...")
				}
			});
		}
		return onManagePage
	}

	static getStaff() {
		let s = null;
		$.ajax({
			method: 'GET',
			url:`${webroot}manage`,
			data: {
				action: 'getstaffjquery',
			},
			dataType:"text",
			cache: true,
			async:false,
			success: function(result) {
				let data = JSON.parse(result);
				s = new Staff(data.Username,data.Rank,"");
			},
			error: function() {
				s = new Staff("nobody","0","");
			}
		});
		return s;
	}

	constructor(name, rank, boards) {
		this.name = name;
		this.rank = rank;
		this.boards = boards;
	}
}

/* export function addStaffButtons() {
	$("input#delete-password").remove();
	$("input[value=Delete]").after("<input type=\"submit\" name=\"Ban\" value=\"Ban\" onclick=\"banSelectedPost(); return false;\"  />")
} */

export function getStaff() {
	return Staff.getStaff();
}

export function getManagePage() {

}

export function banSelectedPost() {
	let boardDirArr = location.pathname.split("/");
	if(boardDirArr.length < 2) return;
	let boardDir = boardDirArr[1];
	let checks = $("input[type=checkbox]");
	if(checks.length == 0) {
		alert("No posts selected");
		return false;
	}
	let postID = 0;
	for(let i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") == 0) {
			postID = checks[i].id.replace("check", "");
			break;
		}
	}
	window.location = `${webroot}manage?action=bans&dir=${boardDir}&postid=${postID}`
}

export function getStaffMenuHTML() {
	let s = "<ul class=\"staffmenu\">";
	$.ajax({
		method: 'GET',
		url: webroot + "manage",
		data: {
			action: 'staffmenu',
		},
		dataType: "text",
		cache: true,
		async: false,
		success: result => {
			let lines = result.substring(result.indexOf("body>")+5,result.indexOf("</body")).trim().split("\n");
			for(let l = 0; l < lines.length; l++) {
				if(lines[l] != "") {
					if(lines[l].indexOf("<a href=") > -1) {
						s += lines[l].substr(0,lines[l].indexOf("\">")+2)+"<li>"+$(lines[l]).text()+"</li></a>";
					} else {
						s += `<li>${lines[1]}</li>`;
					}
				}
			}
		},
		error: () => {
			s = "Something went wrong :/";
		}
	});
	return s+"</ul>";
}

export function openStaffLightBox(actionURL) {
	$.ajax({
		method: 'GET',
		url: `${webroot}manage`,
		data: {
			action: actionURL,
		},
		dataType:"html",
		async:false,

		success: result => {
			let body = `<div id="body-mock">${result.replace(/^[\s\S]*<body.*?>|<\/body>[\s\S]*$/ig, "")}</div>`;
			let $body = $(body);
			let header = $body.find("h1");
			let headerText = header.text();
			header.remove();
			if(headerText == "") headerText = "Manage";
			showLightBox(headerText,$body.html());
		},
		error: result => {
			let responseText = result.responseText;
			header = responseText.substring(responseText.indexOf("<h1>")+4,responseText.indexOf("</h1>"));

			responseText = responseText.substring(responseText.indexOf("</h1>") + 5, responseText.indexOf("</body>"));
			if(header == "") {
				showLightBox("Manage", responseText);
			} else {
				showLightBox(header, responseText);
			}
		}
	});
}