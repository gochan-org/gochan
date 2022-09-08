/* global webroot */

import $ from "jquery";
import "jquery-ui/ui/version";
import "jquery-ui/ui/plugin";
import "jquery-ui/ui/safe-active-element";
import "jquery-ui/ui/widget";
import "jquery-ui/ui/scroll-parent";
import "jquery-ui/ui/widgets/mouse";
import "jquery-ui/ui/safe-blur";
import "jquery-ui/ui/widgets/draggable";

import { upArrow, downArrow } from "../vars";
import { getCookie } from "../cookies";
import { $topbar, topbarHeight } from "./topbar";
import { getBooleanStorageVal, getJsonStorageVal, setStorageVal } from "../storage";
import { updateThread } from "../postutil";
import { currentBoard, currentThread, getPageThread } from "../postinfo";
import { getReplyCooldown, getThreadCooldown } from "../api/cooldowns";
import { getUploadFilename, updateUploadImage } from "./uploaddata";

/**
 * @type {JQuery<HTMLElement>}
 */
export let $qr = null;
let threadCooldown = 0;
let replyCooldown = 0;

const qrButtonHTML = 
	`<input type="file" id="imagefile" name="imagefile" accept="image/jpeg,image/png,image/gif,video/webm,video/mp4"/>` +
	`<input type="submit" value="Post" style="float:right;min-width:50px"/>`;

const qrTitleBar =
	`<div id="qr-title">` +
	`<span id="qr-message"></span>` +
	`<span id="qr-buttons"><a href="javascript:toBottom();">${downArrow}</a>` +
	`<a href="javascript:toTop();">${upArrow}</a><a href="javascript:closeQR();">X</a></span></div>`;


function resetSubmitButtonText() {
	if(currentThread().thread < 1) {
		setSubmitButtonText("New Thread");
	} else {
		setSubmitButtonText("Reply");
	}
}

function setSubmitButtonText(text) {
	$qr.find("input[type=submit]").attr("value", text);
}

function setSubmitButtonEnabled(enabled = true) {
	let $submit = $qr.find("input[type=submit]");
	if(enabled) {
		$submit.removeAttr("disabled");
	} else {
		$submit.attr("disabled", "disabled");
	}
}

function unsetQrUpload() {
	$("#imagefile").val("");
	let $uploadContainer = $qr.find("div#upload-container");
	$uploadContainer.empty();
	$uploadContainer.css("display","none");
}

function qrUploadChange() {
	let $uploadContainer = $qr.find("div#upload-container");
	$uploadContainer.empty();
	let filename = getUploadFilename();
	$uploadContainer.append($(this).prop({
		"title": filename
	}).css({
		"max-width": "100%",
		"max-height": "inherit",
	}).on("click", e => {
		if(e.shiftKey) unsetQrUpload();
	}));
	$uploadContainer.css("display", "");
}

function setButtonTimeout(prefix = "", cooldown = 5) {
	let currentSeconds = cooldown;
	let interval = 0;
	const timeoutCB = () => {
		if(currentSeconds == 0) {
			setSubmitButtonEnabled(true);
			resetSubmitButtonText();
			clearInterval(interval);
		} else {
			setSubmitButtonEnabled(false);
			setSubmitButtonText(`${prefix}${currentSeconds--}`);
		}
	};
	interval = setInterval(timeoutCB, 1000);
	timeoutCB();
}

export function initQR(pageThread) {
	if($qr !== null) {
		// QR box already initialized
		return;
	}
	if(pageThread === undefined)
		pageThread = getPageThread();

	if(!getBooleanStorageVal("useqr", true)) {
		return closeQR();
	}


	let onPostingPage = $("form input[name=boardid]").length > 0;
	// don't open the QR box if we aren't on a board or thread page
	if(!onPostingPage)
		return;

	const nameCookie = getCookie("name");
	const emailCookie = getCookie("email");

	let $qrbuttons = $("<div/>")
		.prop("id", "qrbuttons")
		.append(qrButtonHTML);
	let $postform = $("<form/>").prop({
		id: "qrpostform",
		name: "qrpostform",
		action: webroot + "post",
		method: "POST",
		enctype:"multipart/form-data"
	}).append(
		$("<input/>").prop({
			type: "hidden",
			name: "threadid",
			value: pageThread.op
		}),
		$("<input/>").prop({
			type: "hidden",
			name: "json",
			value: 1
		}),
		$("<input/>").prop({
			type: "hidden",
			name: "boardid",
			value: 1
		}),
		$("<div/>").append($("<input/>").prop({
			id: "qrpostname",
			type: "text",
			name: "postname",
			value: nameCookie,
			placeholder: "Name"
		})),
		$("<div/>").append($("<input/>").prop({
			id: "qrpostemail",
			type: "text",
			name: "postemail",
			value: emailCookie,
			placeholder: "Email"
		})),
		$("<div/>").append($("<input/>").prop({
			id: "qrpostsubject",
			type: "text",
			name: "postsubject",
			placeholder: "Subject"
		})),
		$("<div/>").append($("<textarea/>").prop({
			id: "qrpostmsg",
			name: "postmsg",
			placeholder: "Message"
		})),
		$qrbuttons
	);

	let qrTop = 32;
	
	let pintopbar = getBooleanStorageVal("pintopbar", true);
	if(pintopbar)
		qrTop = $topbar.outerHeight() + 16;
	let qrPos = getJsonStorageVal("qrpos", {top: qrTop, left: 16});
	if(!(qrPos.top > -1))
		qrPos.top = qrTop;
	if(!(qrPos.left > -1))
		qrPos.left = 16;

	$qr = $("<div />").prop({
		id: "qr-box",
	}).css({
		top: qrPos.top,
		left: qrPos.left,
		position: "fixed"
	}).append(
		$(qrTitleBar),
		$postform,
		$("<div/>").prop({
			id: "upload-container"
		}).css({
			"display": "none"
		})
	).draggable({
		handle: "div#qr-title",
		scroll: false,
		containment: "window",
		drag: (event, ui) => {
			ui.position.top = Math.max(ui.position.top, topbarHeight);
			setStorageVal("qrpos", ui.position, true);
		}
	});
	openQR();
	updateUploadImage($qrbuttons.find("input#imagefile"), qrUploadChange);
	resetSubmitButtonText();
	if(currentThread().thread < 1) return; 

	$("form#qrpostform").on("submit", function(e) {
		let $form = $(this);
		e.preventDefault();
		let data = new FormData(this);

		$.ajax({
			type: "POST",
			url: $form.attr("action"),
			enctype: "multipart/form-data",
			data: data, // $form.serialize(),
			processData: false,
			contentType: false,
			success: () => {
				clearQR();
				let cooldown = (currentThread().thread > 0)?replyCooldown:threadCooldown;
				setButtonTimeout("", cooldown);
				updateThread().then(clearQR).then(() => {
					let persist = getBooleanStorageVal("persistentqr", false);
					if(!persist) closeQR();
				});
			}
		});
		return false;
	});
}

function clearQR() {
	if(!$qr) return;
	$qr.find("input[name=postsubject]").val("");
	$qr.find("textarea[name=postmsg]").val("");
	$qr.find("input[type=file]").val("");
	$qr.find("div#upload-container").empty();
}

export function openQR() {
	if($qr) $qr.insertAfter("div#content");
}
window.openQR = openQR;

export function closeQR() {
	if($qr) $qr.remove();
}
window.closeQR = closeQR;

$(() => {
	let board = currentBoard();
	if(board == "") return; // not on a board
	getThreadCooldown(board).then(cd => threadCooldown = cd);
	getReplyCooldown(board).then(cd => replyCooldown = cd);
});