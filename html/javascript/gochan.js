var $jq = jQuery.noConflict();

var down_arrow_symbol = "&#9660;";
var up_arrow_symbol = "&#9650;";
var topbar;
var setings_menu;
var settings_arr = [];

var SettingsMenuOption = {
	"name":"",
	"cookieName":"",
	"cookieValue":"",
	"cookieType":"",
}

var DropDownMenu = function(title,callback) {
	this.callback = callback;
	this.buttonTitle = title;
	$jq("div#topbar").append("<ul><a href=\"#\" class=\"dropdown-button\" id=\""+title.toLowerCase()+"\"><li>"+title+down_arrow_symbol+"</li></a></ul>");
	this.button_jq = $jq("div#topbar a#"+title.toLowerCase());
	this.button_jq.click(function(){
		//$jq(document.body).append("<div id="+title.toLowerCase())
		callback();
	})
}

DropDownMenu.prototype.open = function() {

}

DropDownMenu.prototype.close = function() {

}

DropDownMenu.prototype.isOpen = function() {

}

function showLightBox(title,innerHTML) {
	$jq(document.body).prepend("<div class=\"lightbox-bg\"></div><div class=\"lightbox\"><div class=\"lightbox-title\">"+title+"<a href=\"#\" class=\"lightbox-x\">X</a><hr /></div>"+innerHTML+"</div>");
	$jq("a.lightbox-x").click(function() {
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

function isFrontPage() {
	var page = window.location.pathname;
	return page == "/" || page == "/index.html";
}

function isBoardPage() {

}

function isThreadPage() {

}

$jq(document).ready(function() {
	topbar = $jq("div#topbar");
	var settings_html = "<table width=\"100%\"><colgroup><col span=\"1\" width=\"50%\"><col span=\"1\" width=\"50%\"></colgroup><tr><td><b>Style:</b></td><td><select name=\"style\" style=\"min-width:50%\">"
	for(var i = 0; i < styles.length; i++) {
		settings_html += "<option value=\""+styles[i]+"\">"+styles[i][0].toUpperCase()+styles[i].substring(1,styles[i].length);
	}
	settings_html+="</select></td><tr><tr><td><b>Pin top bar:</b></td><td><input type=\"checkbox\" /></td></tr></table><div class=\"lightbox-footer\"><hr /><button id=\"save-settings-button\">Save Settings</button></div>"

 	setings_menu = new DropDownMenu("Settings",function(){
 		showLightBox("Settings",settings_html)
 	});
 	b = new DropDownMenu("WT",function() {});
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
});