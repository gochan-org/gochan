import { getJsonStorageVal } from "../storage";
import { alertLightbox } from "./lightbox";

interface MyPost {
	post: string; // board:post
	time: Date;
}

export async function postFormCB(this: HTMLFormElement, ev: JQuery.SubmitEvent) {
	ev.preventDefault();
	const $form = $(this);
	const formData = new FormData(this);
	const url = $form.attr("action") as string;
	formData.append("json", "1");
	await fetch(url, {
		method: "POST",
		body: formData,
		credentials: "same-origin"
	}).then(response => response.json()).then(async (data: PostSubmitResponse) => {
		if(data.error) {
			alertLightbox(data.error, "Error");
			return;
		}
		addMyPost(data);
		location.href = `${location.origin}${webroot ?? "//"}${data.thread}#${data.id}`;
		return false;
	}).catch(error => alertLightbox(error, "Error"));
}

export function addMyPost(data: PostSubmitResponse) {
	const idParts = data.thread.split("/");
	if(idParts.length < 3) {
		throw new Error(`Failed to parse thread path: ${data.thread}`);
	}
	const board = idParts[idParts.length - 3];

	const myPosts = getJsonStorageVal<MyPost[]>("myposts", []);
	myPosts.push({
		post: `${board}:${data.id}`,
		time: data.time
	});
	localStorage.setItem("myposts", JSON.stringify(myPosts));
}