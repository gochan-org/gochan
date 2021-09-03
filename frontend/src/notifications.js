import $ from "jquery"

const noteCloseTime = 5*1000; // 4 seconds
const noteIcon = "/favicon.png";

function canNotify() {
	return (location.protocol == "https:")
		&& (typeof Notification !== "undefined");
}

export function notify(title, body, img) {
	let n = new Notification(title, {
		body: body,
		image: img,
		icon: img
	});
	setTimeout(() => {
		n.close();
	}, noteCloseTime);
}

$(() => {
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