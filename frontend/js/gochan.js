/* global webroot, defaultStyle */

import "./vars";
import $ from "jquery";
import { handleKeydown } from "./boardevents";
import { initCookies } from "./cookies";
import { initStaff, createStaffMenu } from "./management/manage";
import { getPageThread } from "./postinfo";
import { prepareThumbnails, initPostPreviews } from "./postutil";
import { addPostDropdown } from "./dom/postdropdown";
import { initSettings } from "./settings";
import { initTopBar } from "./dom/topbar";
import { initQR } from "./dom/qr";
import { initWatcher } from "./watcher";
import { getBooleanStorageVal, getStorageVal } from "./storage";

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

$(() => {
	let pageThread = getPageThread();
	let style = getStorageVal("style", defaultStyle);
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

	if(pageThread.board != "") {
		prepareThumbnails();
		if(getBooleanStorageVal("useqr", true))
			initQR(pageThread);
		initPostPreviews();
	}

	$("div.post, div.reply").each((i, elem) => {
		addPostDropdown($(elem));
	});
	$(document).on("keydown", handleKeydown);
});
