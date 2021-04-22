import { initCookies, getCookie } from "./cookies";
import { addStaffButtons, getStaff, getStaffMenuHTML, openStaffLightBox } from "./manage";
import { prepareThumbnails, preparePostPreviews } from "./postutil";
import { initSettings } from "./settings";
import { initTopBar, TopBarButton, DropDownMenu } from "./topbar";
import { initQR } from "./qr";
import { opRegex } from "./vars";

let currentStaff = null;
let $watchedThreadsBtn = null;
let $staffBtn = null;

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

function getBoard() {
	let rootIndex = window.location.pathname.indexOf(webroot);
	let board = window.location.pathname.substring(rootIndex+webroot.length);
	if(board.length > 0 && board.indexOf("/") > -1) {
		board = board.split("/")[0];
	} else {
		board = "";
	}
	return board;
}

export function getPageThread() {
	let arr = opRegex.exec(window.location.pathname);
	let info = {
		board: getBoard(),
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
	let tag = "";
	if(!e.ctrlKey || e.target.nodeName != "TEXTAREA") return;
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
	let ta = e.target;
	let val = ta.value;
	let ss = ta.selectionStart;
	let se = ta.selectionEnd;
	let r = se + 2 + tag.length;
	ta.value = val.slice(0, ss) + ("[" + tag + "]") + val.slice(ss, se) + ("[/" + tag + "]") + val.slice(se);
	ta.setSelectionRange(r, r);
}

$(() => {
	let pageThread = getPageThread();
	let style = getCookie("style", defaultStyle);
	let themeElem = document.getElementById("theme");
	if(themeElem) themeElem.setAttribute("href", `${webroot}css/${style}`);
	currentStaff = getStaff();
	initCookies();
	initTopBar();
	initSettings();

	$watchedThreadsBtn = new TopBarButton("WT", () => {});

	if(currentStaff.rank > 0) {
		$staffBtn = new TopBarButton("Staff", () => {
			window.location = "/manage?action=dashboard"
		})
		/* $staffBtn = new DropDownMenu("Staff",getStaffMenuHTML())
		$("a#staff.dropdown-button").click(function() {
			$("a.staffmenu-item").click(function() {
				let url = $(this).attr("id");
				openStaffLightBox(url);
	 		});
		}); */
		// addStaffButtons();
	}

	if(pageThread.board != "") {
		prepareThumbnails();
		if(getCookie("useqr") == "true") initQR(pageThread);
	}

	preparePostPreviews(false);
	$(".plus").click(function() {
		let block = $(this).parent().next();
		if(block.css("display") == "none") {
			block.show();
			$(this).html("-");
		} else {
			block.hide();
			$(this).html("+");
		}
	});
	let threadMenuOpen = false;
	$(".thread-ddown a, body").click(function(e) {
		e.stopPropagation();
		let postID = $(this).parent().parent().parent().attr("id");
		let isOP = $(this).parent().parent().parent().attr("class") == "thread";

		if(postID == undefined) return;
		if($(this).parent().find("div.thread-ddown-menu").length == 0) {
			$("div.thread-ddown-menu").remove();

			let menuHTML = `<div class="thread-ddown-menu" id="${postID}">`;
			if(!isOP) menuHTML += `<ul><li><a href="javascript:hidePost(${postID});" class="hide-post">Show/Hide post</a></li>`;
			menuHTML +=
				`<li><a href="javascript:deletePost(${postID});" class="delete-post">Delete post</a></li>` +
				`<li><a href="javascript:reportPost(${postID});" class="report-post">Report Post</a></li></ul></div>`

			$(this).parent().append(menuHTML);
			threadMenuOpen = true;
		} else {
			$("div.thread-ddown-menu").remove();
			threadMenuOpen = false;
		}
	});
	$(document).keydown(handleKeydown);
});
