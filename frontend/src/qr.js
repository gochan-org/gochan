import { upArrow, downArrow } from "./vars";
import { getCookie, setCookie } from "./cookies";
import { $topbar, topbarHeight } from "./topbar";

export let $qr = null;

const qrButtonHTML = 
	`<input type="file" id="imagefile" name="imagefile" style="display: none;" />` +
	`<input name="imagefilebtn" type="button" onclick="document.getElementById('imagefile').click();" value="Browse...">` +
	`<input type="submit" value="Post" style="float:right;"/>`;

const qrTitleBar =
	`<div id="qr-title">` +
	`<span id="qr-message"></span>` +
	`<span id="qr-buttons"><a href="javascript:toBottom();">${downArrow}</a>` +
	`<a href="javascript:toTop();">${upArrow}</a><a href="javascript:closeQR();">X</a></span></div>`;

export function initQR(pageThread) {
	if($qr != null) {
		console.log("QR box already initialized");
		return;
	}

	let onPostingPage = $("form input[name=boardid]").length > 0;
	// don't open the QR box if we aren't on a board or thread page
	if(!onPostingPage)
		return;

	const nameCookie = getCookie("name");
	const emailCookie = getCookie("email");

	let $qrbuttons = $("<div />")
		.prop("id", "qrbuttons")
		.append(qrButtonHTML);
	let $postform = $("<form />").prop({
		id: "qrpostform",
		name: "qrpostform",
		action: "/post",
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
			name: "boardid",
			value: 1
		}),
		$("<div id=\"qrpostname\"/>").append($("<input/>").prop({
			id: "qrpostname",
			type: "text",
			name: "postname",
			value: nameCookie,
			placeholder: "Name"
		})),
		$("<div id=\"qrpostemail\"/>").append($("<input/>").prop({
			id: "qrpostemail",
			type: "text",
			name: "postemail",
			value: emailCookie,
			placeholder: "Email"
		})),
		$("<div id=\"qrpostsubject\"/>").append($("<input/>").prop({
			id: "qrpostsubject",
			type: "text",
			name: "postsubject",
			placeholder: "Subject"
		})),
		$("<div id=\"qrpostmsg\"/>").append($("<textarea/>").prop({
			id: "qrpostmsg",
			name: "postmsg",
			placeholder: "Message"
		})),
		$qrbuttons
	);

	let qrTop = 32;
	if(!getCookie("pintopbar",{default: true, type: "bool"}))
		qrTop = $topbar.outerHeight() + 16;

	let qrPos = getCookie("qrpos", {
		type: "json",
		default: JSON.stringify({top: qrTop, left: 16})
	});
	$qr = $("<div />").prop({
		id: "qr-box",
	}).css({
		top: qrPos.top,
		left: qrPos.left,
		position: "fixed"
	}).append(
		$(qrTitleBar), $postform
	).draggable({
		handle: "div#qr-title",
		scroll: false,
		containment: "window",
		drag: (event, ui) => {
			ui.position.top = Math.max(ui.position.top, topbarHeight);
			setCookie("qrpos", JSON.stringify(ui.position),7);
		}
	});

	// Thread updating needs to be implemented for this to be useful
	/* $("form#qrpostform").submit(e => {
		let $form = $(this);
		e.preventDefault();
		$.ajax({
			type: "POST",
			url: $form.attr("action"),
			data: $form.serialize(),
			success: data => {
			}
		})
		return false;
	}); */
	openQR();
}

export function openQR() {
	if($qr) $qr.insertAfter("div#content");
}
window.openQR = openQR;

export function closeQR() {
	if($qr) $qr.remove();
}
window.closeQR = closeQR;
