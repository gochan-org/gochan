import { currentBoard, getThreadJSON } from "./postutil";
import { getJsonStorageVal, setStorageVal } from "./storage";
let watching = false;

export function getWatchedThreads() {
	if(!watching) {
		clearInterval(getWatchedThreads);
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
			console.log(thread);
		}
	}

	// let threadJsonURL = `${webroot}/res/$`;

	// fetch("/test/res/1.json")
	// .then(response => {
	// 	if(!response.ok)
	// 		throw new Error(response.statusText);
	// 	return response.json();
	// })
	// .then(data => {
	// 	console.log(data);
	// })
	// .catch(err => {
	// 	console.log(`Error getting watched threads: ${err}`);
	// 	clearInterval(getWatchedThreads);
	// 	watching = false;
	// })
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
	/* getThreadJSON(threadID, board).then(data => {

	}); */
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

export function initWatcher() {
	let watched = getJsonStorageVal("watched", {});

	let board = currentBoard();
	watching = watched != null && watched[board] instanceof Array;
	if(watching) {
		getWatchedThreads();
		setInterval(getWatchedThreads, 10000);
	}
}