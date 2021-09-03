import { getCookie, setCookie } from "./cookies";
import { getBoard } from "./postutil";

let watching = false;

export function getWatchedThreads() {
	if(!watching) {
		clearInterval(getWatchedThreads);
		return;
	}
	let threadJsonURL = `${webroot}/res/$`
	fetch("/test/res/1.json")
	.then(response => {
		if(!response.ok)
			throw new Error(response.statusText);
		return response.json();
	})
	.then(data => {
		console.log(data);
	})
	.catch(err => {
		console.log(`Error getting watched threads: ${err}`);
		clearInterval(getWatchedThreads);
		watching = false;
	})
}

export function watchThread(threadID, board) {
	watchedCookie = getCookie("watched", {type: "json", default: {}});

	if(!(watchedCookie[board] instanceof Array))
		watchedCookie[board] = [];
	for(const id of watchedCookie[board]) {
		if(id == threadID) return; // thread is already in the watched list
	}
	watchedCookie[board].push(threadID);
	setCookie("watched", JSON.stringify(watchedCookie));
}

export function initWatcher() {
	let watched = {}
	let localWatched = localStorage.getItem("watched");
	if(localWatched) {
		try {
			watched = JSON.parse(localWatched);
		} catch(e) {
			console.log(`Error parsing watched thread setting: ${e}`);
			localStorage.setItem("watched", "{}");
		}
	} else {
		localStorage.setItem("watched", "{}");
	}

	// watchedCookie = getCookie("watched", {type: "json", default: {}});
	let board = getBoard();
	watching = watched[board] instanceof Array;

	if(watching) {
		getWatchedThreads();
		// setInterval(getWatchedThreads, 1000);
	}

}