import { opRegex } from "./vars";
import "jquery-ui-dist/jquery-ui";

import { handleActions, handleKeydown } from "./boardevents";
import { initCookies, getCookie } from "./cookies";
import { initStaff, createStaffMenu } from "./manage";
// import { notify } from './notifications';
import { currentBoard, prepareThumbnails, preparePostPreviews, deletePost, hidePost, reportPost } from "./postutil";
import { initSettings } from "./settings";
import { initTopBar, TopBarButton } from "./topbar";
import { initQR, openQR } from "./qr";
import { initWatcher, watchThread } from "./watcher";

let $watchedThreadsBtn = null;

export function toTop() {
	window.scrollTo(0,0);
}
window.toTop = toTop;

export function toBottom() {
	window.scrollTo(0,document.body.scrollHeight);
}
window.toBottom = toBottom;

export function changePage(sel) {
	let info = getPageThread();
	if(info.board == "" || info.op == -1) return;
	if(sel.value != "")
		window.location = webroot + info.board + "/res/" + info.op + "p" + sel.value + ".html";
}

export function getPageThread() {
	let arr = opRegex.exec(window.location.pathname);
	let info = {
		board: currentBoard(),
		boardID: -1,
		op: -1,
		page: 0
	};
	if(arr == null) return info;
	if(arr.length > 1) info.op = arr[1];
	if(arr.length > 3) info.page = arr[3];
	if(arr.board != "") info.boardID = $("form#postform input[name=boardid]").val() -1;
	return info;
}

$(() => {
	let pageThread = getPageThread();
	let style = getCookie("style", {default: defaultStyle});
	let themeElem = document.getElementById("theme");
	if(themeElem) themeElem.setAttribute("href", `${webroot}css/${style}`);
	initCookies();
	initTopBar();
	initSettings();
	initStaff()
		.then(createStaffMenu)
	.catch(() => {
		// not logged in
	});
	initWatcher();

	let passwordText = $("input#postpassword").val();
	$("input#delete-password").val(passwordText);

	// $watchedThreadsBtn = new TopBarButton("WT", () => {
	// 	alert("Watched threads yet implemented");
	// });

	if(pageThread.board != "") {
		prepareThumbnails();
		if(getCookie("useqr", {type: "bool"})) initQR(pageThread);
	}

	preparePostPreviews(false);
	$("plus").on("click", function() {
		let block = $(this).parent().next();
		if(block.css("display") == "none") {
			block.show();
			$(this).html("-");
		} else {
			block.hide();
			$(this).html("+");
		}
	});

	let $postInfo = $("label.post-info");
	$postInfo.each((i, elem) => {
		let $elem = $(elem);
		let isOP = $elem.parents("div.reply-container").length == 0;
		let postID = $elem.parent().attr("id");
		let threadPost = isOP?"thread":"post";

		let $ddownMenu = $("<select />", {
			class: "post-actions",
			id: postID
		}).append(
			"<option disabled selected>Actions</option>",
		);
		if(isOP) {
			$ddownMenu.append(
				"<option>Watch thread</option>"
			);
		}
		$ddownMenu.append(
			`<option>Show/hide ${threadPost}</option>`,
			`<option>Report post</option>`,
			`<option>Delete ${threadPost}</option>`,
			`<option>Delete file</option>`
		).insertAfter($elem)
		.on("click", event => {
			if(event.target.nodeName != "OPTION")
				return;
			handleActions($ddownMenu.val(), postID);
		});
	});
	$(document).on("keydown", handleKeydown);
});
