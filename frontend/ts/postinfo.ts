import $ from "jquery";
import { WatchedThreadJSON } from "./watcher/watcher";

const opRE = /\/res\/(\d+)(p(\d)+)?.html$/;
const threadRE = /^\d+/;

export function currentBoard() {
	const board = $("form#main-form input[type=hidden][name=board]").val();
	if(typeof board === "string")
		return board;
	return "";
}

export function getPageThread() {
	let pathname = window.location.pathname;
	if(webroot !== "/") {
		pathname = pathname.slice(webroot.length);
		if(pathname === "" || pathname[0] !== "/") {
			pathname = "/" + pathname;
		}
	}
	const arr = opRE.exec(pathname);
	const info = {
		board: currentBoard(),
		boardID: -1,
		op: -1,
		page: 0
	};
	if(arr === null) return info;

	if(arr.length > 1) info.op = Number.parseInt(arr[1]);
	if(arr.length > 3) info.page = Number.parseInt(arr[3]);
	if(info.board !== "") info.boardID = Number.parseInt($("form#postform input[name=boardid]").val() as string) -1;
	return info;
}

export function currentThread(): WatchedThreadJSON {
	// returns the board and thread ID if we are viewing a thread
	const thread = {board: currentBoard(), id: 0};
	let pathname = location.pathname;
	if(typeof webroot === "string" && webroot !== "/") {
		pathname = pathname.slice(webroot.length);
		if(pathname === "" || pathname[0] !== "/") {
			pathname = "/" + pathname;
		}
	}
	const splits = pathname.split("/");
	if(splits.length !== 4)
		return thread;
	const reArr = threadRE.exec(splits[3]);
	if(reArr.length > 0)
		thread.id = Number.parseInt(reArr[0]);
	return thread;
}

export function insideOP(elem: any) {
	return $(elem).parents("div.op-post").length > 0;
}

/**
 * Return the appropriate thumbnail filename for the given upload filename (replacing gif/webm with jpg, etc)
 */
export function getThumbFilename(filename: string) {
	const nameParts = /([^.]+)\.([^.]+)$/.exec(filename);
	if(nameParts === null) return filename;
	const name = nameParts[1] + "t";
	let ext = nameParts[2];
	if(ext === "gif" || ext === "webm")
		ext = "jpg";

	return name + "." + ext;
}