import $ from "jquery";

import { getThreadJSON } from "../api/threads";
import { currentThread } from "../postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "../storage";
import "./menu";

const subjectCuttoff = 24;

let watcherInterval = -1;

export function updateWatchedThreads() {
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);
	let currentPage = currentThread();
	for(const board of boards) {
		if(!(watched[board] instanceof Array)) {
			console.error(`Invalid data for board ${board}: expected Array object, deleting.`);
			delete watched[board];
			continue;
		}
		$(document).trigger("beginNewPostsCheck");
		for(const t in watched[board]) {
			const thread = watched[board][t];
			if(thread.err !== undefined) continue;
			getThreadJSON(thread.id, board).then(data => {
				if(data.posts.length > thread.posts) {
					// watched thread has new posts, trigger a menu update
					if(currentPage.board == board && currentPage.thread == thread.id) {
						// we're currently in the thread, update the cookie
						watched[board][t].posts = data.posts.length;
						watched[board][t].latest = data.posts[data.posts.length - 1].no;
						setStorageVal("watched", watched, true);
					}
					$(document).trigger("watcherNewPosts", {
						newPosts: data.posts.slice(thread.posts),
						newNumPosts: data.posts.length,
						op: thread.id,
						board: board
					});
				}
			}).catch(e => {
				if(e.status == 404) {
					watched[board][t].err = e.statusText;
					setStorageVal("watched", watched, true);
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
			board: board,
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
		$(document).trigger("watchThread", threadObj);
	});
}

export function unwatchThread(threadID, board) {
	let watched = getJsonStorageVal("watched", {});
	if(!(watched[board] instanceof Array))
		return;
	for(const i in watched[board]) {
		if(watched[board][i].id == threadID) {
			console.log(threadID);
			watched[board].splice(i, 1);
			setStorageVal("watched", watched, true);
			$(document).trigger("unwatchThread", threadID);
			return;
		}
	}
}

export function stopThreadWatcher() {
	clearInterval(watcherInterval);
}

export function resetThreadWatcherInterval() {
	stopThreadWatcher();
	watcherInterval = setInterval(updateWatchedThreads, getNumberStorageVal("watcherseconds", 10) * 1000);
}

export function initWatcher() {
	updateWatchedThreads();
	resetThreadWatcherInterval();
}

$(initWatcher);