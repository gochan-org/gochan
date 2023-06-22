import $ from "jquery";
import { getStorageVal, setStorageVal } from "../storage";

const noop = () => {
	return;
};

/**
 * isPostVisible returns true if the post exists and is visible, otherwise false
 * @param id the id of the post
 */
export function isPostVisible(id: number) {
	const $post = $(`div#op${id}.op-post,div#reply${id}.reply`);
	if($post.length === 0)
		return false;
	return $post.find(".post-text").is(":visible");
}

/**
 * setPostVisibility sets the visibility of the post with the given ID. It returns true if it finds
 * a post or thread with the given ID, otherwise false
 * @param id the id of the post to be toggled
 * @param visibility the visibility to be set
 * @param onComplete called after the visibility is set
 */
export function setPostVisibility(id: number|string, visibility: boolean, onComplete = noop) {
	const $post = $(`div#op${id}.op-post, div#reply${id}.reply`);

	if($post.length === 0)
		return false;
	const $toSet = $post.find(".file-info,.post-text,.upload,.file-deleted-box,br");
	const $backlink = $post.find("a.backlink-click");
	const hiddenStorage = getStorageVal("hiddenposts", "").split(",");
	if(visibility) {
		$toSet.show(0, onComplete);
		$post.find<HTMLOptionElement>("select.post-actions option").each((e, elem) => {
			elem.text = elem.text.replace("Show", "Hide");
		});
		$backlink.text(id);
		const newHidden = [];
		for(const sID of hiddenStorage) {
			if(sID !== id && newHidden.indexOf(sID) === -1) newHidden.push(sID);
		}
		setStorageVal("hiddenposts", newHidden.join(","));
	} else {
		$toSet.hide(0, onComplete);
		$post.find<HTMLOptionElement>("select.post-actions option").each((e, elem) => {
			elem.text = elem.text.replace("Hide", "Show");
		});
		$backlink.text(`${id} (hidden)`);
		if(hiddenStorage.indexOf(id as string) === -1) hiddenStorage.push(id as string);
		setStorageVal("hiddenposts", hiddenStorage.join(","));
	}

	return true;
}

/**
 * setThreadVisibility sets the visibility of the thread with the given ID, as well as its replies.
 * It returns true if it finds a thread with the given ID, otherwise false
 * @param id the id of the thread to be hidden
 * @param visibility the visibility to be set
 */
export function setThreadVisibility(opID: number|string, visibility: boolean) {
	const $thread = $(`div#op${opID}.op-post`).parent(".thread");
	if($thread.length === 0) return false;
	return setPostVisibility(opID, visibility, () => {
		const $toSet = $thread.find(".reply-container,b,br");
		if(visibility) {
			$toSet.show();
		} else {
			$toSet.hide();
		}
	});
}



$(() => {
	let hiddenPosts = getStorageVal("hiddenposts", "").split(",");
	if(typeof hiddenPosts === "number") hiddenPosts = [hiddenPosts];
	for(let i = 0; i < hiddenPosts.length; i++) {
		const id = hiddenPosts[i];
		setThreadVisibility(id, false);
		setPostVisibility(id, false);
	}
});