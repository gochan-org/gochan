let watching = false;

export function getWatchedThreads() {
	if(!watching) {
		clearInterval(getWatchedThreads);
		return;
	}

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

export function initWatcher() {
	watching = true;
	getWatchedThreads();
	// setInterval(getWatchedThreads, 1000);
}