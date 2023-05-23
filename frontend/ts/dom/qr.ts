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
import { currentBoard, currentThread } from "../postinfo";
import { getReplyCooldown, getThreadCooldown } from "../api/cooldowns";
import { getUploadFilename, updateUploadImage } from "./uploaddata";
import { alertLightbox } from "./lightbox";

export let $qr: JQuery<HTMLElement> = null;
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
	if(currentThread().id < 1) {
		setSubmitButtonText("New Thread");
	} else {
		setSubmitButtonText("Reply");
	}
}

function setSubmitButtonText(text: string) {
	$qr.find("input[type=submit]").attr("value", text);
}

function setSubmitButtonEnabled(enabled = true) {
	const $submit = $qr.find("input[type=submit]");
	if(enabled) {
		$submit.removeAttr("disabled");
	} else {
		$submit.attr("disabled", "disabled");
	}
}

function unsetQrUpload() {
	$("#imagefile").val("");
	const $uploadContainer = $qr.find("div#upload-container");
	$uploadContainer.empty();
	$uploadContainer.css("display","none");
}

function qrUploadChange() {
	const $uploadContainer = $qr.find("div#upload-container");
	$uploadContainer.empty();
	const filename = getUploadFilename();
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
	let interval: NodeJS.Timer = null;
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

export function initQR() {
	if($qr !== null) {
		// QR box already initialized
		return;
	}

	if(!getBooleanStorageVal("useqr", true)) {
		return closeQR();
	}

	const onPostingPage = $("form input[name=boardid]").length > 0;
	// don't open the QR box if we aren't on a board or thread page
	if(!onPostingPage)
		return;

	const nameCookie = getCookie("name");
	const emailCookie = getCookie("email");
	const $oldForm = $("form#postform");

	const $qrbuttons = $("<div/>")
		.prop("id", "qrbuttons")
		.append(qrButtonHTML);
	const $postform = $("<form/>").prop({
		id: "qrpostform",
		name: "qrpostform",
		action: webroot + "post",
		method: "POST",
		enctype:"multipart/form-data"
	}).append(
		$("<input/>").prop({
			type: "hidden",
			name: "threadid",
			value: $oldForm.find("input[name='threadid']").val()
		}),
		$("<input/>").prop({
			type: "hidden",
			name: "json",
			value: 1
		}),
		$("<input/>").prop({
			type: "hidden",
			name: "boardid",
			value: $oldForm.find("input[name='boardid']").val()
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
	
	const pintopbar = getBooleanStorageVal("pintopbar", true);
	if(pintopbar)
		qrTop = $topbar.outerHeight() + 16;
	const qrPos = getJsonStorageVal("qrpos", {top: qrTop, left: 16});
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
	if(currentThread().id < 1) {
		$("form#qrpostform").on("submit", function(_e) {
			copyCaptchaResponse($(this));
		});
		return; 
	}
	$postform.on("submit", function(e) {
		const $form = $<HTMLFormElement>(this as HTMLFormElement);
		e.preventDefault();
		copyCaptchaResponse($form);
		const data = new FormData(this as HTMLFormElement);

		$.ajax({
			type: "POST",
			url: $form.attr("action"),
			enctype: "multipart/form-data",
			data: data, // $form.serialize(),
			processData: false,
			contentType: false,
			success: (data, _status, _jqXHR) => {
				if(data.error) {
					alertLightbox(data.error, "Error");
					return;
				}
				clearQR();
				const cooldown = (currentThread().id > 0)?replyCooldown:threadCooldown;
				setButtonTimeout("", cooldown);
				updateThread().then(clearQR).then(() => {
					const persist = getBooleanStorageVal("persistentqr", false);
					if(!persist) closeQR();
				});
				return false;
			},
			error: (_jqXHR, _status, error) => {
				alertLightbox(error, "Error");
			}
		});
		return false;
	});
}

function copyCaptchaResponse($copyToForm: JQuery<HTMLElement>) {
	const $captchaResp = $("textarea[name=h-captcha-response]");
	if($captchaResp.length > 0) {
		$("<textarea/>").prop({
			"name": "h-captcha-response"
		}).val($("textarea[name=h-captcha-response]").val()).css("display", "none")
			.appendTo($copyToForm);
	}
}

function clearQR() {
	if(!$qr) return;
	$qr.find("input[name=postsubject]").val("");
	$qr.find("textarea[name=postmsg]").val("");
	$qr.find("input[type=file]").val("");
	$qr.find("div#upload-container").empty();
}

export function openQR() {
	if($qr) {
		if($qr.parent().length == 0) {
			$qr.insertAfter("div#content");
		} else {
			$qr.show();
		}
	}
}
window.openQR = openQR;

export function closeQR() {
	if($qr) $qr.hide();
}
window.closeQR = closeQR;

$(() => {
	const board = currentBoard();
	if(board == "") return; // not on a board
	getThreadCooldown(board).then(cd => threadCooldown = cd);
	getReplyCooldown(board).then(cd => replyCooldown = cd);
});