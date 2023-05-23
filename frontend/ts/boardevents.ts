import $ from "jquery";

import { openQR } from "./dom/qr";

export function handleKeydown(e: JQuery.KeyDownEvent) {
	const ta = e.target;
	const isPostMsg = ta.nodeName == "TEXTAREA" && ta.name == "postmsg";
	const inForm = ta.form != undefined;
	if(!inForm && !e.ctrlKey && e.keyCode == 81) {
		openQR();
	} else if(isPostMsg && e.ctrlKey) {
		applyBBCode(e);
	}
}

export function applyBBCode(e: JQuery.KeyDownEvent) {
	let tag = "";
	switch(e.keyCode) {
	case 10: // Enter key
	case 13: // Enter key in Chrome/IE
		// trigger the form submit event, whether the QR post box or the static post box is currently
		$(e.target).parents("form#postform,form#qrpostform").trigger("submit");
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
	const ta = e.target;
	const val = ta.value;
	const ss = ta.selectionStart;
	const se = ta.selectionEnd;
	const r = se + 2 + tag.length;
	ta.value = val.slice(0, ss) +
		`[${tag}]` +
		val.slice(ss, se) +
		`[/${tag}]` +
		val.slice(se);
	ta.setSelectionRange(r, r);
	$(ta).text(ta.value);
}
