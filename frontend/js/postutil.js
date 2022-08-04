/* global webroot */
/**
 * @typedef { import("./types/gochan").BoardThread } BoardThread
 * @typedef { import("./types/gochan").ThreadPost } ThreadPost
 */


import $ from "jquery";

import { getCookie } from "./cookies";
import { alertLightbox, promptLightbox } from "./lightbox";
import { getBooleanStorageVal, getNumberStorageVal } from "./storage";
import { handleActions } from "./boardevents";
import { isThreadWatched } from "./watcher";

let doClickPreview = false;
let doHoverPreview = false;
let $hoverPreview = null;
let threadWatcherInterval = 0;

const threadRE = /^\d+/;
const videoTestRE = /\.(mp4)|(webm)$/;
const postrefRE = /\/([^\s/]+)\/res\/(\d+)\.html(#(\d+))?/;
const idRe = /^((reply)|(op))(\d+)/;
const opRegex = /\/res\/(\d+)(p(\d)+)?.html$/;


// data retrieved from /<board>/res/<thread>.json
/** @type {BoardThread} */
let currentThreadJSON = {
	posts: []
};

export function getPageThread() {
	let arr = opRegex.exec(window.location.pathname);
	let info = {
		board: currentBoard(),
		boardID: -1,
		op: -1,
		page: 0
	};
	if(arr == null) return info;
	if(arr.length > 1) info.op = arr[1];
	if(arr.length > 3) info.page = arr[3];
	if(arr.board != "") info.boardID = $("form#postform input[name=boardid]").val() -1;
	return info;
}

export function getUploadPostID(upload, container) {
	// if container, upload is div.upload-container
	// otherwise it's img or video
	let jqu = container? $(upload) : $(upload).parent();
	return insideOP(jqu) ? jqu.siblings().eq(4).text() : jqu.siblings().eq(3).text();
}

export function currentBoard() {
	let board = $("form#main-form input[type=hidden][name=board]").val();
	if(typeof board == "string")
		return board;
	return "";
}

export function currentThread() {
	// returns the board and thread ID if we are viewing a thread
	let thread = {board: currentBoard(), thread: 0};
	let splits = location.pathname.split("/");
	if(splits.length != 4)
		return thread;
	let reArr = threadRE.exec(splits[3]);
	if(reArr.length > 0)
		thread.thread = reArr[0];
	return thread;
}

/* export function hidePost(id) {
	let posttext = $("div#"+id+".post .posttext");
	if(posttext.length > 0) posttext.remove();
	let fileinfo = $("div#"+id+".post .file-info")
	if(fileinfo.length > 0) fileinfo.remove();
	let postimg = $("div#"+id+".post img")
	if(postimg.length > 0) postimg.remove();
} */

export function insideOP(elem) {
	return $(elem).parents("div.op-post").length > 0;
}


/**
 * creates an element from the given post data
 * @param {ThreadPost} post
 * @param {string} boardDir
 */
function createPostElement(post, boardDir, elementClass = "inlinepostprev") {
	let $post = $("<div/>")
		.prop({class: elementClass});
	$post.append(
		$("<input/>")
			.prop({
				type: "checkbox",
				id: `check${post.no}`,
				name: `check${post.no}`
			}),
		$("<label/>")
			.prop({
				class: "post-info",
				for: `check${post.no}`
			}).append(post.time),
		" ",
		$("<a/>")
			.prop({
				href: webroot + boardDir + "/res/" + ((post.resto > 0)?post.resto:post.no) + ".html#" + post.no
			}).text("No."),
		" ",
		$("<a/>")
			.prop({
				href: `javascript:quote(${post.no})`
			}).text(post.no),
	);
	let $postInfo = $post.find("label.post-info");
	let postName = (post.name == "" && post.trip == "")?"Anonymous":post.name;
	let $postName = $("<span/>").prop({class: "postername"});
	if(post.email == "") {
		$postName.text(postName);
	} else {
		$postName.append($("<a/>").prop({
			href: "mailto:" + post.email
		}).text(post.name));
	}
	$postInfo.prepend($postName);
	if(post.trip != "") {
		$postInfo.prepend($postName, $("<span/>").prop({class: "tripcode"}).text("!" + post.trip), " ");
	} else {
		$postInfo.prepend($postName, " ");
	}

	if(post.sub != "")
		$postInfo.prepend($("<span/>").prop({class:"subject"}).text(post.sub), " ");


	if(post.filename != "" && post.filename != "deleted") {
		let thumbFile = getThumbFilename(post.tim);
		$post.append(
			$("<div/>").prop({class: "file-info"})
				.append(
					"File: ",
					$("<a/>").prop({
						href: webroot + boardDir + "/src/" + post.tim,
						target: "_blank"
					}).text(post.tim),
					` - (## MB , ${post.w}x${post.h},`,
					$("<a/>").prop({
						class: "file-orig",
						href: webroot + boardDir + "/src/" + post.tim,
						download: post.filename,
					}).text(post.filename),
					")"
				),
			$("<a/>").prop({class: "upload-container", href: webroot + boardDir + "/src/" + post.tim})
				.append(
					$("<img/>")
						.prop({
							class: "upload",
							src: webroot + boardDir + "/thumb/" + thumbFile,
							alt: webroot + boardDir + "/src/" + post.tim,
							width: post.tn_w,
							height: post.tn_h
						})
				)	
		);
	}
	$post.append(
		"<br/>",
		$("<div/>").prop({
			class: "post-text"
		}).html(post.com)
	)
	return $post;
}

/**
 * Return the appropriate thumbnail filename for the given upload filename (replacing gif/webm with jpg, etc)
 * @param {string} filename
 */
function getThumbFilename(filename) {
	let nameParts = /([^.]+)\.([^.]+)$/.exec(filename);
	if(nameParts === null) return filename;
	let name = nameParts[1] + "t";
	let ext = nameParts[2];
	if(ext == "gif" || ext == "webm")
		ext = "jpg";

	return name + "." + ext;
}

function isPostLoaded(id) {
	return currentThreadJSON.posts.filter((post, p) => post.no == id).length > 0;
}

export function updateThreadJSON() {
	let thread = currentThread();
	if(thread.thread === 0) return; // not in a thread
	return getThreadJSON(thread.thread, thread.board).then((json) => {
		if(!(json.posts instanceof Array) || json.posts.length == 0)
			return;
		currentThreadJSON = json;
	}).catch(e => {
		console.error(`Failed updating current thread: ${e}`);
		clearInterval(threadWatcherInterval);
	});
}

function updateThreadHTML() {
	let thread = currentThread();
	if(thread.thread === 0) return; // not in a thread
	let numAdded = 0;
	for(const post of currentThreadJSON.posts) {
		let selector = "";
		if(post.resto == 0)
			selector += `div#${post.no}.thread`;
		else
			selector += `a#${post.no}.anchor`;
		let elementExists = $(selector).length > 0;
		if(elementExists)
			continue; // TODO: check for edits
		
		let $replyContainer = $("<div/>").prop({
			id: `replycontainer${post.no}`,
			class: "reply-container"
		}).append(
			createPostElement(post, thread.board, "reply")
		);

		$replyContainer.appendTo(`div#${post.resto}.thread`);
		console.log(`added post #${post.no}`);
		numAdded++;
	}
	console.log(`Added ${numAdded} posts`);
}

function createPostPreview(e, $post, inline = true) {
	let $preview = $post.clone();
	if(inline) $preview = addPostDropdown($post.clone());
	$preview
		.prop({class: "inlinepostprev"})
		.find("div.inlinepostprev").remove()
		.find("a.postref").on("click", expandPost);
	if(inline) {
		$preview.insertAfter(e.target);
	}
	initPostPreviews($preview);
	return $preview;
}

function previewMoveHandler(e) {
	if($hoverPreview == null) return;
	$hoverPreview.css({position: "absolute"}).offset({
		top: e.pageY + 8,
		left: e.pageX + 8
	});
}

function expandPost(e) {
	e.preventDefault();
	if($hoverPreview != null) $hoverPreview.remove();
	let $next = $(e.target).next();
	if($next.prop("class") == "inlinepostprev" && e.type == "click") {
		// inline preview is already opened, close it
		$next.remove();
		return;
	}
	let href = e.target.href;
	let hrefArr = postrefRE.exec(href);
	if(hrefArr == null) return; // not actually a link to a post, abort
	let postID = hrefArr[4]?hrefArr[4]:hrefArr[2];

	let $post = $(`div#op${postID}, div#reply${postID}`).first();
	if($post.length > 0) {
		let $preview = createPostPreview(e, $post, e.type == "click");
		if(e.type == "mouseenter") {
			$hoverPreview = $preview.insertAfter(e.target);
			$(document.body).on("mousemove", previewMoveHandler);
		}
		return;
	}
	if(e.type == "click") {
		$.get(href, data => {
			$post = $(data).find(`div#op${postID}, div#reply${postID}`).first();
			if($post.length < 1) return; // post not on this page.
			createPostPreview(e, $post, true);
		}).catch((t, u, v) => {
			alertLightbox(v, "Error");
			return;
		});
	}
}

export function initPostPreviews($post = null) {
	if(getPageThread().board == "" && $post == null) return;
	doClickPreview = getBooleanStorageVal("enablepostclick", true);
	doHoverPreview = getBooleanStorageVal("enableposthover", false);
	let $refs = null;
	$refs = $post == null ? $("a.postref") : $post.find("a.postref");

	if(doClickPreview) {
		$refs.on("click", expandPost);
	} else {
		$refs.off("click", expandPost);
	}

	if(doHoverPreview) {
		$refs.on("mouseenter", expandPost).on("mouseleave", () => {
			if($hoverPreview != null) $hoverPreview.remove();
			$hoverPreview = null;
			$(document.body).off("mousemove", previewMoveHandler);
		});
	} else {
		$refs.off("mouseenter").off("mouseleave").off("mousemove");
	}
}

export function prepareThumbnails() {
	// set thumbnails to expand when clicked
	$("a.upload-container").on("click", function(e) {
		e.preventDefault();
		let a = $(this);
		let thumb = a.find("img.upload");
		let thumbURL = thumb.attr("src");
		let uploadURL = thumb.attr("alt");
		thumb.removeAttr("width").removeAttr("height");

		var fileInfoElement = a.prevAll(".file-info:first");
		
		if(videoTestRE.test(thumbURL + uploadURL)) {
			// Upload is a video
			thumb.hide();
			var video = $("<video />")
			.prop({
				src: uploadURL,
				autoplay: true,
				controls: true,
				class: "upload",
				loop: true
			}).insertAfter(fileInfoElement);

			fileInfoElement.append($("<a />")
			.prop("href", "javascript:;")
			.on("click", function() {
				video.remove();
				thumb.show();
				this.remove();
				thumb.prop({
					src: thumbURL,
					alt: uploadURL
				});
			}).css({
				"padding-left": "8px"
			}).html("[Close]<br />"));
		} else {
			// upload is an image
			thumb.attr({
				src: uploadURL,
				alt: thumbURL
			});
		}
		return false;
	});
}

// heavily based on 4chan's quote() function, with a few tweaks
export function quote(e) {
	let msgboxID = "postmsg";

	let msgbox = document.getElementById(msgboxID);

	if(document.selection) {
		document.getElementById(msgboxID).focus();
		let t = document.getselection.createRange();
		t.text = `>>${e}\n`;
	} else if(msgbox.selectionStart || "0" == msgbox.selectionStart) {
		let n = msgbox.selectionStart,
		o = msgbox.selectionEnd;
		msgbox.value = msgbox.value.substring(0, n) + ">>" + e + "\n" + msgbox.value.substring(o, msgbox.value.length);
	} else msgbox.value += `>>${e}\n`;
	window.scroll(0,msgbox.offsetTop - 48);
}
window.quote = quote;

export function addPostDropdown($post) {
	if($post.find("select.post-actions").length > 0)
		return $post;
	let $postInfo = $post.find("label.post-info");
	let isOP = $postInfo.parents("div.reply-container").length == 0;
	let hasUpload = $postInfo.siblings("div.file-info").length > 0;
	let postID = $postInfo.parent().attr("id");
	let threadPost = isOP?"thread":"post";
	let $ddownMenu = $("<select />", {
		class: "post-actions",
		id: postID
	}).append("<option disabled selected>Actions</option>");
	if(isOP) {
		let threadID = idRe.exec(postID)[4];
		if(isThreadWatched(threadID, currentBoard())) {
			$ddownMenu.append("<option>Unwatch thread</option>");
		} else {
			$ddownMenu.append("<option>Watch thread</option>");
		}
	}
	$ddownMenu.append(
		// `<option>Show/hide ${threadPost}</option>`,
		`<option>Edit post</option>`,
		`<option>Report post</option>`,
		`<option>Delete ${threadPost}</option>`,
	).insertAfter($postInfo)
	.on("click", event => {
		if(event.target.nodeName != "OPTION")
			return;
		handleActions($ddownMenu.val(), postID);
	});
	if(hasUpload)
		$ddownMenu.append(`<option>Delete file</option>`);
	return $post;
}

export function editPost(id, board) {
	let cookiePass = getCookie("password");
	promptLightbox(cookiePass, true, () => {
		$("input[type=checkbox]").prop("checked", false);
		$(`input#check${id}`).prop("checked", true);
		$("input[name=edit_btn]").trigger("click");
	}, "Edit post");
}

export function reportPost(id, board) {
	promptLightbox("", false, ($lb, reason) => {
		if(reason == "" || reason == null) return;
		let xhrFields = {
			board: board,
			report_btn: "Report",
			reason: reason,
			json: "1"
		};
		xhrFields[`check${id}`] = "on";
		$.post(webroot + "util", xhrFields).fail(data => {
			let errStr = data.error;
			if(errStr == undefined)
				errStr = data.statusText;
			alertLightbox(`Report failed: ${errStr}`, "Error");
		}).done(() => {
			alertLightbox("Report sent", "Success");
		}, "json");
	}, "Report post");
}
window.reportPost = reportPost;

function deletePostFile(id) {
	let $elem = $(`div#op${id}.op-post`);
	alertLightbox("File deleted", "Success");
	// TODO: Replace this with a thing that replaces the image element with a File Deleted block
	return;
}

function deletePostElement(id) {
	let $elem = $(`div#op${id}.op-post`);
	if($elem.length > 0) {
		$elem.parent().next().remove(); // also removes the <hr> element after
		$elem.parent().remove();
	} else {
		$(`div#replycontainer${id}`).remove();
	}
}

export function deletePost(id, board, fileOnly) {
	let cookiePass = getCookie("password");
	promptLightbox(cookiePass, true, ($lb, password) => {
		let xhrFields = {
			board: board,
			boardid: $("input[name=boardid]").val(),
			delete_btn: "Delete",
			password: password,
			json: "1"
		};
		xhrFields[`check${id}`] = "on";
		if(fileOnly) {
			xhrFields.fileonly = "on";
		}
		$.post(webroot + "util", xhrFields).fail(data => {
			if(data !== "");
				alertLightbox(`Delete failed: ${data.error}`, "Error");
		}).done(data => {
			if(data.error == undefined || data == "") {
				if(location.href.indexOf(`/${board}/res/${id}.html`) > -1) {
					alertLightbox("Thread deleted", "Success");
				} else if(fileOnly) {
					deletePostFile(id);
				} else {
					deletePostElement(id);
				}
			} else {
				if(data.boardid == 0 && data.postid == 0) {
					alertLightbox(`Error deleting post #${id}: Post doesn't exist`, "Error");
				} else if(data !== "") {
					alertLightbox(`Error deleting post #${id}`, "Error");
					console.log(data);
				}
			}
		}, "json");
	}, "Password");
}
window.deletePost = deletePost;

export function getThreadJSON(threadID, board) {
	return $.ajax({
		url: `${webroot}${board}/res/${threadID}.json`,
		cache: false,
		dataType: "json"
	});
}

$(() => {
	let pageThread = getPageThread();
	if(pageThread.op < 1) return; // not in a thread

	threadWatcherInterval = setInterval(() => {
		updateThreadJSON().then(updateThreadHTML).catch(e => {
			console.error(`Error updating current thread: ${e}`);
		});
	}, getNumberStorageVal("watcherseconds", 10) * 1000);
})