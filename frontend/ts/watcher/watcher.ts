import $ from "jquery";

import { getThreadJSON } from "../api/threads";
import { currentThread } from "../postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "../storage";
import "./menu";

const subjectCuttoff = 24;

let watcherInterval = -1; // eslint-disable-line prefer-const

export function updateWatchedThreads() {
	const watched = getJsonStorageVal<any>("watched", {});
	const boards = Object.keys(watched);
	const currentPage = currentThread();
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
					if(currentPage.board === board && currentPage.id === thread.id) {
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
				if(e.status === 404) {
					watched[board][t].err = e.statusText;
					setStorageVal("watched", watched, true);
				}
			});
		}
	}
}

export interface WatchedThreadsListJSON {
	[board: string]: WatchedThreadJSON[]
}

export interface WatchedThreadJSON {
	id: number;
	board?: string;
	posts?: number;
	op?: string;
	latest?: string;
	subject?: string;

	newNumPosts?: number;
	err?: string;
	newPosts?: any[];
}

export function isThreadWatched(threadID: number, board: string) {
	const watched = getJsonStorageVal<WatchedThreadsListJSON>("watched", {});
	const threads = watched[board];
	if(threads === undefined) return false;
	for(const thread of threads) {
		if(thread.id === threadID) return true;
	}
	return false;
}

export function watchThread(threadID: string|number, board: string) {
	const watched = getJsonStorageVal<WatchedThreadsListJSON>("watched", {});
	if(typeof threadID === "string") {
		threadID = parseInt(threadID);
	}
	if(!(watched[board] instanceof Array))
		watched[board] = [];

	for(const t in watched[board]) {
		let thread = watched[board][t];
		if(typeof thread === "number") {
			thread = watched[board][t] = {id: thread};
		}
		if(thread.id === threadID) return; // thread is already in the watched list
	}
	getThreadJSON(threadID, board).then(data => {
		const op = data.posts[0];
		const threadObj: WatchedThreadJSON = {
			id: threadID as number,
			board: board,
			posts: data.posts.length,
			op: op.name,
			latest: data.posts[data.posts.length-1].no
		};
		if(op.trip !== "") threadObj.op += "!" + op.trip;
		if(op.sub !== "") {
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

export function unwatchThread(threadID: number, board: string) {
	const watched = getJsonStorageVal<WatchedThreadsListJSON>("watched", {});
	if(!(watched[board] instanceof Array))
		return;
	for(const i in watched[board]) {
		if(watched[board][i].id === threadID) {
			watched[board].splice(i as any, 1);
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
	(watcherInterval as unknown as NodeJS.Timer) = setInterval(
		updateWatchedThreads,
		getNumberStorageVal("watcherseconds", 10) * 1000);
}

export function initWatcher() {
	updateWatchedThreads();
	resetThreadWatcherInterval();
}

$(initWatcher);