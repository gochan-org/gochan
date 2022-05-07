import { currentBoard } from "./postutil";
import { getJsonStorageVal, setStorageVal } from "./storage";
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
	let watched = getJsonStorageVal("watched", {});
	if(!(watched[board] instanceof Array))
		watched[board] = [];
	for(const id of watched[board]) {
		if(id == threadID) return; // thread is already in the watched list
	}
	watched[board].push(threadID);
	setStorageVal("watched", JSON.stringify(watched));
}

export function initWatcher() {
	let watched = getJsonStorageVal("watched", {});

	let board = currentBoard();
	watching = watched != null && watched[board] instanceof Array;

	if(watching) {
		getWatchedThreads();
		// setInterval(getWatchedThreads, 1000);
	}
}