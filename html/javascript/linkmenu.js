// Drop down link menu

var down_arrow_symbol = "&#9660;";
var up_arrow_symbol = "&#9650;";

var $jq = jQuery.noConflict();

var DropDownMenu = function(title,html) {
	this.html = html;
	this.buttonTitle = title;
	$jq("div#topmenu").append("<a href=\"#\" style=\"float:right;\" class=\"dropdown-button\" id=\""+title.toLowerCase()+"\">"+title+"</a>");
	this.button = $jq("div#topmenu a#"+title.toLowerCase());

	this.button_jq = $jq("a#"+title.);
	this.button_jq.click(function(){
		$jq(document.body).append("<div id="+title.toLowerCase())
	})
}

DropDownMenu.prototype.open = function() {

}

DropDownMenu.prototype.close = function() {

}

DropDownMenu.prototype.isOpen = function() {

}

