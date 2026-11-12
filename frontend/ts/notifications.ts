import $ from "jquery";
import { webPath } from "./util";


const noteCloseTime = 4*1000; // 4 seconds
const noteIcon = webPath("favicon.png");

function canNotify() {
	return (location.protocol === "https:")
		&& (typeof Notification !== "undefined");
}

export function notify(title: string, body: string, icon = noteIcon) {
	const n = new Notification(title, {
		body: body,
		icon: icon
	});
	setTimeout(() => {
		n.close();
	}, noteCloseTime);
}

$(() => {
	if(!canNotify())
		return;

	Notification.requestPermission().then(granted => {
		if(granted !== "granted")
			return Promise.reject("denied");
	}).catch(err => {
		if(err !== "denied")
			console.log(`Error starting notifications: ${err}`);
	});
});