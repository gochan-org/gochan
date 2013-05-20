var $jq = jQuery.noConflict();

var down_arrow_symbol = "&#9660;";
var up_arrow_symbol = "&#9650;";
var topbar;
var settings_menu;
var staff_btn;
var watched_threads_btn;
var settings_arr = [];
var current_staff;

var SettingsMenuOption = {
	"name":"",
	"cookieName":"",
	"cookieValue":"",
	"cookieType":"",
}

var Staff = function(name,rank,boards) {
	this.name = name;
	this.rank = rank;
	this.boards = boards;
}

var TopBarButton = function(title,callback) {
	this.title = title;
	this.callback = callback;
	this.buttonTitle = title;
	$jq("div#topbar").append("<ul><a href=\"javascript:void(0)\" class=\"dropdown-button\" id=\""+title.toLowerCase()+"\"><li>"+title+down_arrow_symbol+"</li></a></ul>");
	this.button_jq = $jq("div#topbar a#"+title.toLowerCase());
	this.button_jq.click(this.callback);
}

var DropDownMenu = function(title,menu_html) {
	this.title = title;
	this.open = false;
	this.menuHTML = menu_html;

	this.button = new TopBarButton(title,function() {
		if(this.open) {
			$jq("a#"+title.toLowerCase()).children(0).html(title+down_arrow_symbol);
			$jq("div#"+title.toLowerCase()).remove();
			this.open = false;
		} else {
			$jq("a#"+title.toLowerCase()).children(0).html(title+up_arrow_symbol);
			topbar.after("<div id=\""+title.toLowerCase()+"\" class=\"dropdown-menu\">"+menu_html+"</div>")
			$jq("div#"+title.toLowerCase()).css({
				top:topbar.height()
			})
			this.open = true;
		}
	});
}

function showLightBox(title,innerHTML) {
	$jq(document.body).prepend("<div class=\"lightbox-bg\"></div><div class=\"lightbox\"><div class=\"lightbox-title\">"+title+"<a href=\"#\" class=\"lightbox-x\">X</a><hr /></div>"+innerHTML+"</div>");
	$jq("a.lightbox-x").click(function() {
		$jq(".lightbox").remove();
		$jq(".lightbox-bg").remove();
	});
	$jq(".lightbox-bg").click(function() {
		$jq(".lightbox").remove();
		$jq(".lightbox-bg").remove();
	});

}

function generateSettingsList() {

}

function changeFrontPage(page_name) {
	var tabs = $jq(".tab");
	var pages = $jq(".page");
	var current_page = getHashVal();
	pages.hide();
	if(current_page=="") {
		$jq(pages[0]).show();
	} else {
		for(var p = 0; p < pages.length; p++) {
			var page = $jq(pages[p]);
			if(page.attr("id").replace("-page","").replace("page","") == current_page) {
				page.show()
			}
		}
	}

	for(var i = 0; i < tabs.length; i++) {
		var child = $jq(tabs[i]).children(0)
		var tabname = child.text();
		if(tabname.toLowerCase() == current_page) {
			$jq("#current-tab").attr({"id":""});
			child.parent().attr({"id":"current-tab"});
		}
	}

	tabs.find("a").click(function(event) {
		current_page = getHashVal($jq(this).attr("href"));

		if(current_page == "") {
			$jq("#current-tab").attr({"id":""});
			$jq(tabs[0]).attr({"id":"current-tab"});
		} else {
			for(var i = 0; i < tabs.length; i++) {
				var child = $jq(tabs[i]).children(0)
				var tabname = child.text();
				if(tabname.toLowerCase() == current_page) {
					$jq("#current-tab").attr({"id":""});
					$jq(tabs[i]).attr({"id":"current-tab"});
				}
			}
		}

		pages.hide()
		if(current_page=="") {
			$jq(pages[0]).show();
		} else {
			for(var p = 0; p < pages.length; p++) {
				var page = $jq(pages[p]);
				if(page.attr("id").replace("-page","").replace("page","") == current_page) {
					page.show()
				}
			}
		}
	});
}

function deletePost(id) {
	var password = prompt("Password")
}

function getArg(name) {
	var href = window.location.href;
	var args = href.substr(href.indexOf("?")+1, href.length);
	args = args.split("&");

	for(var i = 0; i < args.length; i++) {
		temp_args = args[i];
		temp_args = temp_args.split("=");
		temp_name = temp_args[0];
		temp_value = temp_args[1];
		args[temp_name] = temp_value;
		args[i] = temp_args;
	}
	return args[name];
}

function getHashVal() {
	var href = window.location.href;
	if(arguments.length == 1) {
		href = arguments[0];
	}
	if(href.indexOf("#") == -1) {
		return "";
	} else {
		var hash = href.substring(href.indexOf("#"),href.length);
		if(hash == "#") return ""
		else return hash.substring(1,hash.length);
	}
}

function getManagePage() {
	
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

function hidePost(id) {
	var posttext = $jq("div#"+id+".post .posttext");
	if(posttext.length > 0) posttext.remove();
	var fileinfo = $jq("div#"+id+".post .file-info")
	if(fileinfo.length > 0) fileinfo.remove();
	var postimg = $jq("div#"+id+".post img")
	if(postimg.length > 0) postimg.remove();
}

function isFrontPage() {
	var page = window.location.pathname;
	return page == "/" || page == "/index.html" || page == "/template.html";
}

function isBoardPage() {

}

function isThreadPage() {

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

function preparePostPreviews(is_inline) {
	var m_type = "mousemove";
	if(!movable_postpreviews) m_type = "mouseover";
	if(expandable_postrefs) $("a.postref").attr("href","javascript:void(0);");
	var hvr_str = "a.postref";
	if(is_inline) hvr_str = "div.inlinepostprev "+hvr_str;
	$(hvr_str).hover(function(){
		$(document.body).append($("div#"+this.innerHTML.replace("&gt;&gt;","")).clone().attr("class","postprev"))
		$(document).bind(m_type, function(e){
		    $('.postprev').css({
		       left:  e.pageX + 8,
		       top:   e.pageY + 8
		    });
		})
	},
	function() {
		$(".postprev").remove();
	});

	if(expandable_postrefs) {
		var clk_str = "a.postref";
		if(is_inline) clk_str = "div.inlinepostprev "+clk_str;
		$(clk_str).click(function() {
			if($(this).next().attr("class") != "inlinepostprev") {
				$(".postprev").remove();
				$(this).after($("div#"+this.innerHTML.replace("&gt;&gt;","")).clone().attr({"class":"inlinepostprev","id":"i"+$(this).parent().attr("id")+"-"+($(this).parent().find("div#i"+$(this).parent().attr("id")).length+1)}));
				preparePostPreviews(true);
			} else {
				$(this).next().remove();
			}		
		});
	}
}

function reportPost(id) {
	var reason = prompt("Reason");
}

$jq(document).ready(function() {
	current_staff = getStaff()
	
	topbar = $jq("div#topbar");
	var settings_html = "<table width=\"100%\"><colgroup><col span=\"1\" width=\"50%\"><col span=\"1\" width=\"50%\"></colgroup><tr><td><b>Style:</b></td><td><select name=\"style\" style=\"min-width:50%\">"
	for(var i = 0; i < styles.length; i++) {
		settings_html += "<option value=\""+styles[i]+"\">"+styles[i][0].toUpperCase()+styles[i].substring(1,styles[i].length);
	}
	settings_html+="</select></td><tr><tr><td><b>Pin top bar:</b></td><td><input type=\"checkbox\" /></td></tr><tr><td><b>Enable post previews on hover</b></td><td><input type=\"checkbox\" /></td></tr></table><div class=\"lightbox-footer\"><hr /><button id=\"save-settings-button\">Save Settings</button></div>"

 	settings_menu = new TopBarButton("Settings",function(){
 		showLightBox("Settings",settings_html)
 	});
 	watched_threads_btn = new TopBarButton("WT",function() {});

 	if(current_staff.rank > 0) {
 		staff_btn = new DropDownMenu("Staff",getStaffMenuHTML())
 		$jq("a#staff.dropdown-button").click(function() {
 			$jq("a.staffmenu-item").click(function() {
	 			var url = $jq(this).attr("id");
				openStaffLightBox(url)
	 		});
 		})
 	}

	if(isFrontPage()) {
		changeFrontPage(getHashVal());
	}

	$jq(".plus").click(function() {
		var block = $jq(this).parent().next();
		if(block.css("display") == "none") {
			block.show();
			$jq(this).html("-");
		} else {
			block.hide();
			$jq(this).html("+");
		}
	});

	$jq(".thread-ddown a").click(function(e) {
		var post_id = $jq(this).parent().parent().parent().attr("id")
		var is_op = $jq(this).parent().parent().parent().attr("class") == "thread"
		
		if($jq(this).parent().find("div.thread-ddown-menu").length == 0) {
			$jq("div.thread-ddown-menu").remove();

			menu_html = "<div class=\"thread-ddown-menu\" id=\""+post_id+"\">";
			if(!is_op) menu_html += "<a href=\"javascript:hidePost("+post_id+");\" class=\"hide-post\">Show/Hide post</a><br />";
			menu_html +="<a href=\"javascript:deletePost("+post_id+");\" class=\"delete-post\">Delete post</a><br />" +
				"<a href=\"javascript:reportPost("+post_id+");\" class=\"report-post\">Report Post</a>" +
				"</div>";

			$jq(this).parent().append(menu_html);
		} else {
			$jq("div.thread-ddown-menu").remove();
		}
	});
});