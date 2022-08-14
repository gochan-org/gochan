import $ from "jquery";

const opRegex = /\/res\/(\d+)(p(\d)+)?.html$/;
const threadRE = /^\d+/;

export function currentBoard() {
	let board = $("form#main-form input[type=hidden][name=board]").val();
	if(typeof board == "string")
		return board;
	return "";
}

export function getPageThread() {
	let arr = opRegex.exec(window.location.pathname);
	let info = {
		board: currentBoard(),
		boardID: -1,
		op: -1,
		page: 0
	};
	if(arr === null) return info;
	if(arr.length > 1) info.op = arr[1];
	if(arr.length > 3) info.page = arr[3];
	if(arr.board != "") info.boardID = $("form#postform input[name=boardid]").val() -1;
	return info;
}

export function currentThread() {
	// returns the board and thread ID if we are viewing a thread
	let thread = {board: currentBoard(), thread: 0};
	let splits = location.pathname.split("/");
	if(splits.length != 4)
		return thread;
	let reArr = threadRE.exec(splits[3]);
	if(reArr.length > 0)
		thread.thread = reArr[0];
	return thread;
}

export function insideOP(elem) {
	return $(elem).parents("div.op-post").length > 0;
}

/**
 * Return the appropriate thumbnail filename for the given upload filename (replacing gif/webm with jpg, etc)
 * @param {string} filename
 */
export function getThumbFilename(filename) {
	let nameParts = /([^.]+)\.([^.]+)$/.exec(filename);
	if(nameParts === null) return filename;
	let name = nameParts[1] + "t";
	let ext = nameParts[2];
	if(ext == "gif" || ext == "webm")
		ext = "jpg";

	return name + "." + ext;
}