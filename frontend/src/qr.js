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
	const nameCookie = getCookie("name");
	const emailCookie = getCookie("email");
	const qrFormHTML =
		`<input type="hidden" name="threadid" value="${pageThread.op}" />` +
		`<input type="hidden" name="boardid" value="1" />` +
		`<div id="qrpostname"><input id="qrpostname" type="text" name="postname" value="${nameCookie}" placeholder="Name"/></div>` +
		`<div id="qrpostemail"><input id="qrpostemail" type="text" name="postemail" value="${emailCookie}" placeholder="Email"/></div>` +
		`<div id="qrpostsubject"><input id="qrpostsubject" type="text" name="postsubject" placeholder="Subject"/></div>` +
		`<div id="qrpostmsg"><textarea id="qrpostmsg" name="postmsg" id="postmsg" placeholder="Message"></textarea></div>`;

	let $qrbuttons = $("<div />")
		.prop("id", "qrbuttons")
		.append(qrButtonHTML);
	let $postform = $("<form />").prop({
		id: "qrpostform",
		name: "qrpostform",
		action: "/post",
		method: "POST",
		enctype:"multipart/form-data"
	}).append(qrFormHTML,$qrbuttons);
	let qrTop = 32;
	if(!getCookie("pintopbar",{default: true, type: "bool"}))
		qrTop = $topbar.outerHeight() + 16;

	let qrPos = getCookie("qrpos", {
		type: "json",
		default: JSON.stringify({top: qrTop, left: 16})
	});
	$qr = $("<div />").prop({
		id: "qr-box",
		style: `top: ${qrPos.top}px;left: ${qrPos.left}px; position:fixed;`,
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
	}).insertAfter("div#footer");

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
}

export function closeQR() {
	if($qr) $qr.remove();
}
window.closeQR = closeQR;
