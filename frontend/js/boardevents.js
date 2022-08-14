import $ from "jquery";

import { openQR } from "./dom/qr";

export function handleKeydown(e) {
	let ta = e.target;
	let isPostMsg = ta.nodeName == "TEXTAREA" && ta.name == "postmsg";
	let inForm = ta.form != undefined;
	if(!inForm && !e.ctrlKey) {
		openQR();
	} else if(isPostMsg && e.ctrlKey) {
		applyBBCode(e);
	}
}

export function applyBBCode(e) {
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
	let ta = e.target;
	let val = ta.value;
	let ss = ta.selectionStart;
	let se = ta.selectionEnd;
	let r = se + 2 + tag.length;
	ta.value = val.slice(0, ss) +
		`[${tag}]` +
		val.slice(ss, se) +
		`[/${tag}]` +
		val.slice(se);
	ta.setSelectionRange(r, r);
	$(ta).text(ta.value);
}
