/* global webroot */

import $ from "jquery";
import { getThreadJSON } from "./api/threads";

import { $topbar, TopBarButton } from "./dom/topbar";
import { currentBoard } from "./postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "./storage";

const subjectCuttoff = 24;

let watching = false;
let watcherInterval = -1;
let watcherBtn = null;
/** @type {JQuery<HTMLElement>} */
let $watcherMenu = null;

export function updateWatchedThreads() {
	if(!watching) {
		clearInterval(watcherInterval);
		return;
	}
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);
	for(const board of boards) {
		if(!(watched[board] instanceof Array)) {
			console.error(`Invalid data for board ${board}: expected Array object, deleting.`);
			delete watched[board];
			continue;
		}
		for(const t in watched[board]) {
			const thread = watched[board][t];
			if(thread.err !== undefined) continue;
			getThreadJSON(thread.id, board).then(data => {
				console.log(`Thread #${thread.id} has ${data.posts.length} posts, latest ID is ${data.posts.pop().no}`);
			}).catch(e => {
				if(e.status == 404) {
					watched[board][t].err = e.statusText;
					setStorageVal("watched", watched, true);
					updateWatcherMenu();
				}
			});
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

/**
 * @param {number|string} threadID
 * @param {string} board
 */
export function watchThread(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	threadID = parseInt(threadID);
	if(!(watched[board] instanceof Array))
		watched[board] = [];

	for(const t in watched[board]) {
		let thread = watched[board][t];
		if(typeof thread === "number") {
			thread = watched[board][t] = {id: thread};
		}
		if(thread.id == threadID) return; // thread is already in the watched list
	}
	getThreadJSON(threadID, board).then(data => {
		const op = data.posts[0];
		let threadObj = {
			id: threadID,
			posts: data.posts.length,
			op: op.name,
			latest: data.posts[data.posts.length-1].no
		};
		if(op.trip != "") threadObj.op += "!" + op.trip;
		if(op.sub != "") {
			if(op.sub.length > subjectCuttoff)
				threadObj.subject = op.sub.slice(0, subjectCuttoff) + "...";
			else
				threadObj.subject = op.sub;
		}
		watched[board].push(threadObj);
		setStorageVal("watched", watched, true);
		updateWatcherMenu();
	});
}

export function unwatchThread(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	if(!(watched[board] instanceof Array))
		return;
	for(const i in watched[board]) {
		if(watched[board][i].id == threadID) {
			// console.log(`unwatching thread /${board}/${threadID}`);
			watched[board].splice(i, 1);
			setStorageVal("watched", watched, true);
			updateWatcherMenu();
			return;
		}
	}
}

function updateWatcherMenu() {
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);

	if($watcherMenu === null) {
		$watcherMenu = $("<div/>").prop({
			id: "watchermenu",
			class: "dropdown-menu"
		}).append("<b>Watched threads</b><br/>");
	} else {
		$watcherMenu.find(".watcher-item,#no-threads").remove();
	}
	let numWatched = 0;
	for(const board of boards) {
		for(const thread of watched[board]) {
			let infoElem = ` &#8213; <b>OP:</b> ${thread.op}`;
			if(thread.subject !== undefined) {
				infoElem += `<br/><b>Subject: </b> ${thread.subject}`;
			}
			let $watcherItem = $("<div/>").prop({class: "watcher-item"}).append(
				$("<a/>").prop({
					href: `${webroot}${board}/res/${thread.id}.html`
				}).text(`/${board}/${thread.id}`),"  (",
				$("<a/>").prop({
					href: "javascript:;",
					title: `Unwatch thread #${thread.id}`
				}).on("click", () => {
					unwatchThread(thread.id, board);
				}).text("X"), ")  "
			);
			if(thread.err !== undefined)
				$watcherItem.append($("<span/>")
					.css({color: "red"})
					.text(`(${thread.err})`)
				);
			$watcherMenu.append(
				$watcherItem.append(infoElem)
			);
			numWatched++;
		}
	}
	if(numWatched == 0) {
		$watcherMenu.append($("<i/>")
			.prop({id: "no-threads"})
			.text("no watched threads")
		);
	}
	if(watcherBtn === null) {
		watcherBtn = new TopBarButton("Watcher", () => {
			$topbar.trigger("menuButtonClick", [$watcherMenu, $(document).find($watcherMenu).length == 0]);
		});
	}
}

export function initWatcher() {
	let watched = getJsonStorageVal("watched", {});
	updateWatchedThreads();
	updateWatcherMenu();
	if(watcherInterval > -1) {
		clearInterval(watcherInterval);
	}
	let board = currentBoard();
	watching = watched !== null && watched[board] instanceof Array;
	if(watching) {
		updateWatchedThreads();
		watcherInterval = setInterval(updateWatchedThreads, getNumberStorageVal("watcherseconds", 10) * 1000);
	}
}