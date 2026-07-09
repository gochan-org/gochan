import $ from "jquery";

import { getThreadJSON } from "../api/threads";
import { currentThread } from "../postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "../storage";
import "./menu";

const subjectCuttoff = 24;

let watcherInterval = -1;

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
	newPosts?: ThreadPost[];
}

export function updateWatchedThreads() {
	const watched = getJsonStorageVal<WatchedThreadsListJSON>("watched", {});
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
				if(data.posts.length > (thread.posts ?? 0)) {
					// watched thread has new posts, trigger a menu update
					if(currentPage.board === board && currentPage.id === thread.id) {
						// we're currently in the thread, update the cookie
						watched[board][t].posts = data.posts.length;
						watched[board][t].latest = data.posts[data.posts.length - 1].no.toString();
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
			latest: data.posts[data.posts.length-1].no.toString()
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
	for(let i = 0; i < watched[board].length; i++) {
		if(watched[board][i].id === threadID) {
			watched[board].splice(i, 1);
			setStorageVal("watched", watched, true);
			$(document).trigger("unwatchThread", threadID);
			return;
		}
	}
}

export function stopThreadWatcher() {
	clearInterval(watcherInterval);
	watcherInterval = -1;
}

export function resetThreadWatcherInterval() {
	stopThreadWatcher();
	watcherInterval = setInterval(
		updateWatchedThreads,
		getNumberStorageVal("watcherseconds", 10) * 1000) as unknown as number;
}

export function initWatcher() {
	updateWatchedThreads();
	resetThreadWatcherInterval();

	const $watcherContents = $("<div/>").append(
		$("<label/>").append(
			"Auto-update threads",
			$<HTMLInputElement>("<input/>").attr({
				type: "checkbox"
			}).prop("checked", true).on("change", (ev) => {
				console.log("Auto-update:", ev.target.checked);
			})
		),
		$("<label/>").append(
			"Auto-scroll on new posts",
			$<HTMLInputElement>("<input/>").attr({
				type: "checkbox"
			}).prop("checked", false).on("change", (ev) => {
				console.log("Auto-scroll:", ev.target.checked);
			})
		),
		$("<div/>").append(
			"Update interval: ",
			$<HTMLInputElement>("<input/>").attr({
				type: "number",
				min: 1,
				max: 3600
			}).val(getNumberStorageVal("watcherseconds", 10)).on("change", ev => {
				const val = parseInt((ev.target as HTMLInputElement).value);
				console.log("Update interval:", val);
			})
		),
		$("<input/>").attr({
			type: "button",
			value: "Update now"
		}).on("click", ev => {
			ev.preventDefault();
			console.log("Updating watched threads now...");
		})
	).hide();


	const $miniWatcher = $("<div/>").attr({
		"id": "mini-watcher"
	}).text("+0 -0").append($watcherContents).on("mouseover", () => {
		$miniWatcher.addClass("expanded");
		$watcherContents.show();
	}).on("mouseout", () => {
		$miniWatcher.removeClass("expanded");
		$watcherContents.hide();
	}).appendTo("body");
}

$(initWatcher);