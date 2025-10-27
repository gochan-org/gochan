import $ from "jquery";

import { isThreadWatched, watchThread, unwatchThread } from "../watcher/watcher";
import { isPostVisible, setPostVisibility, setThreadVisibility } from "./posthiding";
import { currentBoard } from "../postinfo";
import { getCookie } from "../cookies";
import { alertLightbox, promptLightbox } from "./lightbox";
import { getPostInfo } from "../management/manage";
import { updateThreadLock } from "../api/management";

const idRe = /^((reply)|(op))(\d+)/;

function editPost(id: number, _board: string) {
	const cookiePass = getCookie("password");
	promptLightbox(cookiePass, true, (_jq, inputData) => {
		$("input[type=checkbox]").prop("checked", false);
		$(`input#check${id}`).prop("checked", true);
		$("input#delete-password").val(inputData);
		$("input[name=edit_btn]").trigger("click");
	}, "Edit post");
}

function moveThread(id: number, _board: string) {
	const cookiePass = getCookie("password");
	promptLightbox(cookiePass, true, (_jq, inputData) => {
		$("input[type=checkbox]").prop("checked", false);
		$(`input#check${id}`).prop("checked", true);
		$("input#delete-password").val(inputData);
		$("input[name=move_btn]").trigger("click");
	}, "Move thread");
}

function reportPost(id: number, board: string) {
	promptLightbox("", false, async (_$lb, reason) => {
		if(reason === "" || reason === null) return;
		const searchParams = new URLSearchParams();
		searchParams.append("board", board);
		searchParams.append("report_btn", "Report");
		searchParams.append("reason", reason);
		searchParams.append(`check${id}`, "on");
		searchParams.append("json", "1");
		
		await fetch(`${webroot}util`, {
			method: "POST",
			body: searchParams,
			credentials: "same-origin"
		}).then(response => {
			if(!response.ok) {
				return Promise.reject(response.statusText);
			}
			return response.json();
		}).then(data => {
			if(data.error !== undefined && data.error !== null) {
				alertLightbox(`Report failed: ${data.error.Message}`, "Error");
			} else {
				alertLightbox("Report sent", "Success");
			}
		}).catch(errorText => {
			alertLightbox(`Report failed: ${errorText}`, "Error");
		});
	}, "Report post");
}

function deletePostFile(id: number) {
	const $elem = $(`div#op${id}.op-post, div#reply${id}.reply`);
	if($elem.length === 0) return;
	$elem.find(".file-info,.upload-container").remove();
	$("<div/>").prop({
		class: "file-deleted-box",
		style: "text-align: center;"
	}).text("File removed").insertBefore($elem.find("div.post-text"));
	alertLightbox("File deleted", "Success");
	$(document).trigger("deletePostFile", id);
}

function deletePostElement(id: number) {
	const $elem = $(`div#op${id}.op-post`);
	if($elem.length > 0) {
		$elem.parent().next().remove(); // also removes the <hr> element after
		$elem.parent().remove();
		$(document).trigger("deletePost", id);
	} else {
		$(`div#replycontainer${id}`).remove();
	}
}

function deletePost(id: number, board: string, fileOnly = false) {
	const cookiePass = getCookie("password");
	promptLightbox(cookiePass, true, async (_lb, password) => {
		const searchParams = new URLSearchParams();
		searchParams.append("board", board);
		searchParams.append("boardid", $("input[name=boardid]").val() as string);
		searchParams.append("delete_btn", "Delete");
		searchParams.append("password", password);
		searchParams.append("json", "1");
		searchParams.append(`check${id}`, "on");
		if(fileOnly) {
			searchParams.append("fileonly", "on");
		}
		await fetch(`${webroot}util`, {
			method: "POST",
			body: searchParams,
			credentials: "same-origin"
		}).then(response => {
			if(!response.ok) {
				return Promise.reject(response.statusText);
			}
			return response.json();
		}).then(data => {
			if(data.error === undefined || data === "") {
				if(fileOnly) {
					deletePostFile(id);
				} else if(location.href.indexOf(`/${board}/res/${id}.html`) > -1) {
					alertLightbox("Thread deleted", "Success");
				} else {
					deletePostElement(id);
				}
			} else {
				if(data.error !== undefined) {
					alertLightbox(data.error, `Error deleting post #${id}`);
				} else if(data !== "") {
					alertLightbox(`Error deleting post #${id}`, "Error");
				}
			}
		}).catch(errorText => {
			alertLightbox(`Delete failed: ${errorText}`, "Error");
		});
	}, "Password");
}

function handleActions(action: string, postIDStr: string) {
	const idArr = idRe.exec(postIDStr);
	if(!idArr) return;
	const postID = Number.parseInt(idArr[4]);
	const board = currentBoard();
	switch(action) {
	case "Watch thread":
		watchThread(postID, board);
		break;
	case "Unwatch thread":
		unwatchThread(postID, board);
		break;
	case "Show thread":
		setThreadVisibility(postID, true);
		break;
	case "Hide thread":
		setThreadVisibility(postID, false);
		break;
	case "Move thread":
		moveThread(postID, board);
		break;
	case "Show post":
		setPostVisibility(postID, true);
		break;
	case "Hide post":
		setPostVisibility(postID, false);
		break;
	case "Edit post":
		editPost(postID, board);
		break;
	case "Report post":
		reportPost(postID, board);
		break;
	case "Delete file":
		deletePost(postID, board, true);
		break;
	case "Delete thread":
	case "Delete post":
		deletePost(postID, board, false);
		break;
	// manage stuff
	case "Lock thread":
		console.log(`Locking /${board}/${postID}`);
		updateThreadLock(board, postID, true);
		break;
	case "Unlock thread":
		console.log(`Unlocking /${board}/${postID}`);
		updateThreadLock(board, postID, false);
		break;
	case "Posts from this IP":
		getPostInfo(postID).then((info: any) => {
			window.open(`${webroot}manage/ipsearch?limit=100&ip=${info.post.IP}`);
		}).catch((reason: JQuery.jqXHR) => {
			alertLightbox(`Failed getting post IP: ${reason.statusText}`, "Error");
		});
		break;
	case "Ban IP address":
		window.open(`${webroot}manage/bans?dir=${board}&postid=${postID}`);
		break;
	case "Filter similar posts":
		window.open(`${webroot}manage/filters?srcpost=${postID}`);
		break;
	default:
		// this shouldn't happen under normal circumstances
		alertLightbox("Unrecognized post dropdown option");
		break;
	}
}

export function addPostDropdown($post: JQuery<HTMLElement>) {
	if($post.find("select.post-actions").length > 0)
		return $post;
	const $postInfo = $post.find("label.post-info");
	const isOP = $post.prop("class").split(" ").indexOf("op-post") > -1;
	const hasUpload = $postInfo.siblings("div.file-info").length > 0;
	const postID = $postInfo.parent().attr("id");
	const threadPost = isOP?"thread":"post";
	const $ddownMenu = $("<select />", {
		class: "post-actions",
		id: postID
	}).append("<option disabled selected>Actions</option>");
	const idNum = Number.parseInt(idRe.exec(postID)[4]);
	if(isOP) {
		if(isThreadWatched(idNum, currentBoard())) {
			$ddownMenu.append("<option>Unwatch thread</option>");
		} else {
			$ddownMenu.append("<option>Watch thread</option>");
		}
		$ddownMenu.append("<option>Move thread</option>");
	}
	const showHide = isPostVisible(idNum)?"Hide":"Show";
	$ddownMenu.append(
		`<option>${showHide} ${threadPost}</option>`,
		"<option>Edit post</option>",
		"<option>Report post</option>",
		`<option>Delete ${threadPost}</option>`,
	).insertAfter($postInfo)
		.on("change", _e => {
			handleActions($ddownMenu.val() as string, postID);
			$ddownMenu.val("Actions");
		});
	if(hasUpload)
		$ddownMenu.append("<option>Delete file</option>");
	$post.trigger("postDropdownAdded", {
		post: $post,
		dropdown: $ddownMenu
	});
	return $post;
}

$(() => {
	$(document).on("watchThread", (_e, thread) => {
		$<HTMLOptionElement>(`div#op${thread.id} select.post-actions > option`).each((i, el) => {
			if(el.text === "Watch thread")
				el.text = "Unwatch thread";
		});
	}).on("unwatchThread", (_e, threadID) => {
		$<HTMLOptionElement>(`div#op${threadID} select.post-actions > option`).each((i, el) => {
			if(el.text === "Unwatch thread")
				el.text = "Watch thread";
		});
	});
});
