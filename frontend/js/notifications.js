/* global webroot */

import $ from "jquery";

const noteCloseTime = 4*1000; // 4 seconds
const noteIcon = webroot + "/favicon.png";

function canNotify() {
	return (location.protocol == "https:")
		&& (typeof Notification !== "undefined");
}

export function notify(title, body, img = noteIcon) {
	let n = new Notification(title, {
		body: body,
		image: img,
		icon: noteIcon
	});
	setTimeout(() => {
		n.close();
	}, noteCloseTime);
}

$(document).on("ready", () => {
	if(!canNotify())
		return;

	Notification.requestPermission().then(granted => {
		if(granted != "granted")
			return Promise.reject("denied");
	}).catch(err => {
		if(err != "denied")
			console.log(`Error starting notifications: ${err}`);
	});
});