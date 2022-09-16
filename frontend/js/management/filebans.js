import $ from "jquery";
import "jquery-ui/ui/widget";
import "jquery-ui/ui/unique-id";
import "jquery-ui/ui/keycode";
import "jquery-ui/ui/widgets/tabs";
$(() => {
	if(window.location.search.indexOf("?action=filebans") != 0)
		return;
	$("div#fileban-tabs").tabs();
});