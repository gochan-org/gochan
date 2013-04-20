// Drop down link menu

var down_arrow_symbol = "&#9660;";
var up_arrow_symbol = "&#9650;";

var $jq = jQuery.noConflict();

var LinkMenu = function(title) {
	this.linkURLs = [];
	this.linkTitles = [];
	this.buttonTitle = title;
	$jq("div#verytopbar").append("<a href=\"#\" style=\"float:right;\" class=\"dropdown-button\" id=\""+title.toLowerCase()+"\">"+title+"</a>");
	this.button_jq = $jq("a#"+title.);
	this.button_jq.click(function(){
		$jq(document.body).append("<div id="+title.toLowerCase())
	})
}

LinkMenu.prototype.open = function() {

}

LinkMenu.prototype.close = function() {

}

LinkMenu.prototype.isOpen = function() {

}

LinkMenu.prototype.addLink = function() {

}

LinkMenu.prototype.buildHTML = function() {

}