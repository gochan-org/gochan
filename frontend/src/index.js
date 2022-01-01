// needed for Promise stuff
import "core-js/stable";
import "regenerator-runtime/runtime";

import { initCookies, getCookie } from "./cookies";
import { initStaff, createStaffMenu } from "./manage";
// import { notify } from './notifications';
import { currentBoard, prepareThumbnails, preparePostPreviews, deletePost, hidePost, reportPost } from "./postutil";
import { initSettings } from "./settings";
import { initTopBar, TopBarButton, DropDownMenu } from "./topbar";
import { initQR, openQR } from "./qr";
import { opRegex } from "./vars";
import { initWatcher, watchThread } from "./watcher";

let $watchedThreadsBtn = null;
let idRe = /^((reply)|(op))(\d)/;

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

function handleKeydown(e) {
	let ta = e.target;
	let isPostMsg = ta.nodeName == "TEXTAREA" && ta.name == "postmsg";
	let inForm = ta.form != undefined;
	if(!inForm && !e.ctrlKey) {
		openQR();
		return;
	} else if(isPostMsg && e.ctrlKey) {
		applyBBCode(e, ta);
	}
}

function applyBBCode(e, ta) {
	let tag = "";
	switch(e.keyCode) {
		case 10: // Enter key
		case 13: // Enter key in Chrome/IE
			document.getElementById("postform").submit();
		break;
		case 66: // B
			tag = "b"; // bold
		break;
		case 73: // I
			tag = "i"; // italics
		break;
		case 82: // R
			tag = "s"; // strikethrough
		break;
		case 83:
			tag = "?"; // spoiler (not yet implemented)
		break;
		case 85: // U
			tag = "u"; // underline
		break;
	}
	if(tag == "") return;

	e.preventDefault();
	let val = ta.value;
	let ss = ta.selectionStart;
	let se = ta.selectionEnd;
	let r = se + 2 + tag.length;
	ta.value = val.slice(0, ss) + 
		("[" + tag + "]") +
		val.slice(ss, se) +
		("[/" + tag + "]") +
		val.slice(se);
	ta.setSelectionRange(r, r);
}

function handleActions(action, postID) {
	// console.log(`Action for ${postID}: ${action}`);
	switch(action) {
		case "Watch thread":
			let idArr = idRe.exec(postID);
			if(!idArr) break;
			let threadID = idArr[4];
			let board = currentBoard();
			console.log(`Watching thread ${threadID} on board /${board}/`);
			watchThread(threadID, board);
			break;
		case "Show/hide thread":
		case "Show/hide post":
			console.log(`Showing/hiding ${postID}`);
			hidePost(postID);
			break;
		case "Report post":
			reportPost(postID);
			console.log(`Reporting ${postID}`);
			break;
		case "Delete thread":
		case "Delete post":
			console.log(`Deleting ${postID}`);
			deletePost(postID);
			break;
	}
}

$(() => {
	let pageThread = getPageThread();
	let style = getCookie("style", {default: defaultStyle});
	let themeElem = document.getElementById("theme");
	if(themeElem) themeElem.setAttribute("href", `${webroot}css/${style}`);
	initCookies();
	initTopBar();
	initSettings();
	initStaff().then(createStaffMenu);
	initWatcher();

	let passwordText = $("input#postpassword").val();
	$("input#delete-password").val(passwordText);

	$watchedThreadsBtn = new TopBarButton("WT", () => {
		alert("Watched threads yet implemented");
	});

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
			`<option>Delete ${threadPost}</option>`
		).insertAfter($elem)
		.on("click", event => {
			if(event.target.nodeName != "OPTION")
				return;
			handleActions($ddownMenu.val(), postID);
		});
	});
	$(document).on("keydown", handleKeydown);
});
