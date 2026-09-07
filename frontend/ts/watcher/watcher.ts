import $ from "jquery";

import { getThreadJSON } from "../api/threads";
import { currentThread, getPageThread } from "../postinfo";
import { getJsonStorageVal, getNumberStorageVal, setStorageVal } from "../storage";
import "./menu";
import { addPostDropdown } from "../dom/postdropdown";

const subjectCuttoff = 24;
export const defaultWatcherSeconds = 30;

let secondsLeft = -1;
let watcherInterval = -1;
let addedPosts = 0;
let currentThreadError = false;

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
						setStorageVal("watched", watched);
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
					setStorageVal("watched", watched);
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
		setStorageVal("watched", watched);
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
			setStorageVal("watched", watched);
			$(document).trigger("unwatchThread", threadID);
			return;
		}
	}
}

export function stopThreadWatcher() {
	clearInterval(watcherInterval);
	watcherInterval = -1;
	secondsLeft = -1;
	addedPosts = 0;
}

function updateCurrentThread() {
	const pageThread = getPageThread();
	if(currentThreadError || pageThread.op < 1) return;

	const url = `${webroot ?? "/"}${pageThread.board}/res/${pageThread.op}.html`;
	fetch(url).then(async resp => {
		if(!resp.ok) {
			currentThreadError = true;
			throw new Error(`Failed to fetch thread /${pageThread.board}/${pageThread.op}: ${resp.status} ${resp.statusText}`);
		}
		const respText = await resp.text();
		const $doc = $(respText);
		const $docPosts = $doc.find(".reply-container");
		const $posts = $(".reply-container");
		addedPosts = Math.max($docPosts.length - $posts.length, 0);
		for(const post of $docPosts) {
			const $post = $(post);
			if($posts.filter(`#${$post.attr("id")}`).length === 0) {
				addPostDropdown($post);
				$posts.last().parent().append($post);
			}
		}
	}).catch(e => {
		currentThreadError = true;
		console.error(e);
	});
}


function countdownToUpdate() {
	if(watcherInterval === -1) return;
	$("#mini-watcher-label").text(`+${addedPosts} -${Math.max(secondsLeft, 0)}`);
	if(--secondsLeft <= 0) {
		secondsLeft = getNumberStorageVal("watcherseconds", defaultWatcherSeconds);
		updateWatchedThreads();
		updateCurrentThread();
	}
}

export function resetThreadWatcherInterval() {
	stopThreadWatcher();
	watcherInterval = setInterval(countdownToUpdate, 1000) as unknown as number;
}

function initCurrentThreadUpdater() {
	const pageThread = getPageThread();
	if(pageThread.op < 1) return;

	const $watcherContents = $("<div/>").append(
		$("<label/>").append(
			"Auto-update current thread",
			$<HTMLInputElement>("<input/>").prop({
				type: "checkbox",
				checked: true
			}).on("change", (ev: JQuery.ChangeEvent) => {
				if(ev.target.checked) {
					resetThreadWatcherInterval();
				} else {
					stopThreadWatcher();
				}
			})
		),
		$("<label/>").append(
			"Auto-scroll on new posts",
			$<HTMLInputElement>("<input/>").prop({
				type: "checkbox",
				checked: false
			}).on("change", (ev: JQuery.ChangeEvent) => {
				console.log("Auto-scroll:", ev.target.checked);
			})
		),
		$("<div/>").append(
			"Update interval: ",
			$<HTMLInputElement>("<input/>").attr({
				type: "number",
				min: 5,
				max: 3600,
				value: getNumberStorageVal("watcherseconds", defaultWatcherSeconds)
			}).on("change", (ev: JQuery.ChangeEvent) => {
				const val = Math.min(Math.max(parseInt(ev.target.value), 5), 3600);
				setStorageVal("watcherseconds", val);
				secondsLeft = val;
			})
		),
		$("<input/>").attr({
			type: "button",
			value: "Update now"
		}).on("click", (ev:JQuery.Event) => {
			ev.preventDefault();
			secondsLeft = 0;
		})
	).hide();

	const $miniWatcher = $("<div/>").attr("id", "mini-watcher")
		.append(`<span id="mini-watcher-label">+0 -0</span>`, $watcherContents)
		.on("mouseover", () => {
			$miniWatcher.addClass("expanded");
			$watcherContents.show();
		}).on("mouseout", () => {
			$miniWatcher.removeClass("expanded");
			$watcherContents.hide();
		}).appendTo("body");
}

export function initWatcher() {
	updateWatchedThreads();
	resetThreadWatcherInterval();
	initCurrentThreadUpdater();
}

$(initWatcher);