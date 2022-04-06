import { currentBoard, deletePost, reportPost } from './postutil';
import { watchThread } from "./watcher";
import { openQR } from "./qr";

const idRe = /^((reply)|(op))(\d)/;

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

export function handleActions(action, postIDStr) {
	// console.log(`Action for ${postID}: ${action}`);
	let idArr = idRe.exec(postIDStr);
	if(!idArr) return;
	let postID = idArr[4]
	let board = currentBoard();
	switch(action) {
		case "Watch thread":
			console.log(`Watching thread ${postID} on board /${board}/`);
			watchThread(threadID, board);
			break;
		case "Show/hide thread":
		case "Show/hide post":
			console.log(`Showing/hiding ${postID}`);
			hidePost(postID);
			break;
		case "Report post":
			reportPost(postID, board);
			break;
		case "Delete thread":
		case "Delete post":
			console.log(`Deleting ${postID}`);
			deletePost(postID);
			break;
	}
}