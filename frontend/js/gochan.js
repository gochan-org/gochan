/* global webroot, defaultStyle */

import $ from "jquery";

import "./vars";
import "./cookies";
import "./notifications";
import { setCustomCSS, setCustomJS } from "./settings";
import { handleKeydown } from "./boardevents";
import { initStaff, createStaffMenu } from "./management/manage";
import { getPageThread } from "./postinfo";
import { prepareThumbnails, initPostPreviews } from "./postutil";
import { addPostDropdown } from "./dom/postdropdown";
import { initQR } from "./dom/qr";
import { getBooleanStorageVal, getStorageVal } from "./storage";

export function toTop() {
	window.scrollTo(0,0);
}
window.toTop = toTop;

export function toBottom() {
	window.scrollTo(0,document.body.scrollHeight);
}
window.toBottom = toBottom;

$(() => {
	let pageThread = getPageThread();
	let style = getStorageVal("style", defaultStyle);
	let themeElem = document.getElementById("theme");
	if(themeElem) themeElem.setAttribute("href", `${webroot}css/${style}`);
	initStaff()
		.then(createStaffMenu)
	.catch(() => {
		// not logged in
	});

	let passwordText = $("input#postpassword").val();
	$("input#delete-password").val(passwordText);

	if(pageThread.board != "") {
		prepareThumbnails();
		if(getBooleanStorageVal("useqr", true))
			initQR();
		initPostPreviews();
	}
	$("div.post, div.reply").each((i, elem) => {
		addPostDropdown($(elem));
	});
	$(document).on("keydown", handleKeydown);
	setCustomCSS();
	setCustomJS();
});
