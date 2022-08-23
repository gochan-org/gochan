/* global webroot */

import $ from "jquery";

import { $topbar, TopBarButton } from "./dom/topbar";
import { currentBoard } from "./postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "./storage";
let watching = false;
let watcherInterval = -1;
let watcherBtn = null;
/** @type {JQuery<HTMLElement>} */
let $watcherMenu = null;

export function getWatchedThreads() {
	if(!watching) {
		clearInterval(watcherInterval);
		return;
	}
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);
	for(const board of boards) {
		if(!(watched[board] instanceof Array)) {
			console.log(`Invalid data for board ${board}:`);
			delete watched[board];
			continue;
		}
		for(const thread of watched[board]) {
			// console.log(thread);
		}
	}
}

export function isThreadWatched(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	let threads = watched[board];
	if(threads == undefined) return false;
	for(const thread of threads) {
		if(thread.id == threadID) return true;
	}
	return false;
}

export function watchThread(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	if(!(watched[board] instanceof Array))
		watched[board] = [];
	for(const id of watched[board]) {
		if(id == threadID) return; // thread is already in the watched list
	}
	watched[board].push({
		id: threadID
	});
	setStorageVal("watched", JSON.stringify(watched));
}

export function unwatchThread(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	if(!(watched[board] instanceof Array))
		return;
	for(const i in watched[board]) {
		if(watched[board][i].id == threadID) {
			console.log(`unwatching thread /${board}/${threadID}`);
			watched[board].splice(i, 1);
			setStorageVal("watched", JSON.stringify(watched));
			return;
		}
	}
}

function createWatcherMenu() {
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);

	$watcherMenu = $("<div/>").prop({
		id: "watchermenu",
		class: "dropdown-menu"
	}).append("<b>Watched threads</b><br/>");

	let numWatched = 0;
	for(const board of boards) {
		for(const thread of watched[board]) {
			$watcherMenu.append(
				$("<a/>").prop({href: webroot + board + "/res/" + thread.id + ".html"}).text(`/${board}/${thread.id}`)
			);
			numWatched++;
		}
	}
	if(numWatched == 0) {
		$watcherMenu.append("<i/>").text("no watched threads");
	}
	if(watcherBtn === null) {
		watcherBtn = new TopBarButton("Watcher", () => {
			let exists = $(document).find($watcherMenu).length > 0;
			if(exists)
				$watcherMenu.remove();
			else
				$topbar.after($watcherMenu);
		});
	}
}

export function initWatcher() {
	let watched = getJsonStorageVal("watched", {});
	// createWatcherMenu();
	if(watcherInterval > -1) {
		clearInterval(watcherInterval);
	}
	let board = currentBoard();
	watching = watched !== null && watched[board] instanceof Array;
	if(watching) {
		getWatchedThreads();
		watcherInterval = setInterval(getWatchedThreads, getNumberStorageVal("watcherseconds", 10) * 1000);
	}
}