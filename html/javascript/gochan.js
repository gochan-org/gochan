var $jq = jQuery.noConflict();

var down_arrow_symbol = "&#9660;";
var up_arrow_symbol = "&#9650;";
var board;
var topbar;
var settings_menu;
var staff_btn;
var watched_threads_btn;
var settings_arr = [];
var current_staff;
var lightbox_css_added = false;

var TopBarButton = function(title,callback_open, callback_close) {
	this.title = title;
	$jq("div#topbar").append("<ul><a href=\"javascript:void(0)\" class=\"dropdown-button\" id=\""+title.toLowerCase()+"\"><li>"+title+down_arrow_symbol+"</li></a></ul>");
	var button_open = false;

	$jq("div#topbar a#"+title.toLowerCase()).click(function(event) {
		if(!button_open) {
			callback_open();
			if(callback_close != null) {
				$jq(document).bind("click", function() {
					callback_close();
				});
				button_open = true;
			}
		} else {
			if(callback_close != null) {
				callback_close();
			}
			button_open	= false;
		}
		return false;
	});
}

var DropDownMenu = function(title,menu_html) {
	this.title = title;
	this.menuHTML = menu_html;

	this.button = new TopBarButton(title, function() {
		topbar.after("<div id=\""+title.toLowerCase()+"\" class=\"dropdown-menu\">"+menu_html+"</div>");
		$jq("a#"+title.toLowerCase()).children(0).html(title+up_arrow_symbol);
		$jq("div#"+title.toLowerCase()).css({
			top:topbar.height()
		});
	}, function() {
		$jq("div#"+title.toLowerCase() + ".dropdown-menu").remove();
		$jq("a#"+title.toLowerCase()).children(0).html(title+down_arrow_symbol);
	});
}

function showLightBox(title,innerHTML) {
	if(!lightbox_css_added) {
		$jq(document).find("head").append("\t<link rel=\"stylesheet\" href=\"/css/lightbox.css\" />");
		lightbox_css_added = true;
	}
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

/* function showLightBox(innerHTML) {
	if(!lightbox_css_added) {
		$ja(document).find("head").append("\t<link rel=\"stylesheet\" href=\"/css/lightbox.css\" />");
		lightbox_css_added = true;
	}
} */

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
	window.location = webroot + "util?action=delete&posts="+id+"&board="+board+"&password"
}

function deleteCheckedPosts() {
	if(confirm('Are you sure you want to delete these posts?') == true) {
		form = $jq("form#main-form");
		form.append("<input type=\"hidden\" name=\"action\" value=\"delete\" ");
		form.get(0).submit();
		return true;
	}
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

function hidePost(id) {
	var posttext = $jq("div#"+id+".post .posttext");
	if(posttext.length > 0) posttext.remove();
	var fileinfo = $jq("div#"+id+".post .file-info")
	if(fileinfo.length > 0) fileinfo.remove();
	var postimg = $jq("div#"+id+".post img")
	if(postimg.length > 0) postimg.remove();
}

function initCookies() {
	var name_field = $jq("input#postname");
	var email_field = $jq("input#postemail");
	var password_field = $jq("input#postpassword");
	name_field.val(getCookie("name"));
	email_field.val(getCookie("email"));
	password_field.val(getCookie("password"));
}

function isFrontPage() {
	var page = window.location.pathname;
	return page == "/" || page == "/index.html" || page == "/template.html";
}

function setCookie(name,value) {
	document.cookie = name + "=" + escape(value)
}

function getCookie(name) {
	var cookie_arr = document.cookie.split("; ");
	for(var i = 0; i < cookie_arr.length; i++) {
		pair = cookie_arr[i].split("=");
		if(pair[0] == name) {
			//var val = decodeURIComponent(pair[1]);
			val = pair[1].replace("+", " ");
			val = val.replace("%2B", "+");
			val = decodeURIComponent(val);
			alert(pair[1] + ", " + decodeURIComponent(pair[1]) + ", " +val)
			return val;
		}
	}
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
	board = location.pathname.substring(1,location.pathname.indexOf("/",1))
	current_staff = getStaff()

	topbar = $jq("div#topbar");
	var settings_html = "<table width=\"100%\"><colgroup><col span=\"1\" width=\"50%\"><col span=\"1\" width=\"50%\"></colgroup><tr><td><b>Style:</b></td><td><select name=\"style\" style=\"min-width:50%\">"
	for(var i = 0; i < styles.length; i++) {
		settings_html += "<option value=\""+styles[i]+"\">"+styles[i][0].toUpperCase()+styles[i].substring(1,styles[i].length);
	}
	settings_html+="</select></td><tr><tr><td><b>Pin top bar:</b></td><td><input type=\"checkbox\" /></td></tr><tr><td><b>Enable post previews on hover</b></td><td><input type=\"checkbox\" /></td></tr></table><div class=\"lightbox-footer\"><hr /><button id=\"save-settings-button\">Save Settings</button></div>"

 	settings_menu = new TopBarButton("Settings",function(){
 		showLightBox("Settings",settings_html,null)
 	});
 	watched_threads_btn = new TopBarButton("WT",function() {});

 	if(current_staff.rank > 0) {
 		staff_btn = new DropDownMenu("Staff",getStaffMenuHTML())
 		$jq("a#staff.dropdown-button").click(function() {
 			$jq("a.staffmenu-item").click(function() {
	 			var url = $jq(this).attr("id");
				openStaffLightBox(url)
	 		});
 		});
 		addStaffButtons();
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
	initCookies();
});