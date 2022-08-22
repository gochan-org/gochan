/**
 * @typedef { import("./types/gochan").BoardThread } BoardThread
 * @typedef { import("./types/gochan").ThreadPost } ThreadPost
 */


import $ from "jquery";

import { alertLightbox } from "./dom/lightbox";
import { getBooleanStorageVal, getNumberStorageVal } from "./storage";
import { currentThread, getPageThread, insideOP } from "./postinfo";
import { addPostDropdown } from "./dom/postdropdown";
import { createPostElement } from "./dom/postelement";
import { getThreadJSON } from "./api/threads";
import { openQR } from "./dom/qr";

let doClickPreview = false;
let doHoverPreview = false;
let $hoverPreview = null;
let threadWatcherInterval = 0;

const videoTestRE = /\.(mp4)|(webm)$/;
const postrefRE = /\/([^\s/]+)\/res\/(\d+)\.html(#(\d+))?/;

// data retrieved from /<board>/res/<thread>.json
/** @type {BoardThread} */
let currentThreadJSON = {
	posts: []
};

export function getUploadPostID(upload, container) {
	// if container, upload is div.upload-container
	// otherwise it's img or video
	let jqu = container? $(upload) : $(upload).parent();
	return insideOP(jqu) ? jqu.siblings().eq(4).text() : jqu.siblings().eq(3).text();
}

export function updateThreadJSON() {
	let thread = currentThread();
	if(thread.thread === 0) return; // not in a thread
	return getThreadJSON(thread.thread, thread.board).then((json) => {
		if(!(json.posts instanceof Array) || json.posts.length === 0)
			return;
		currentThreadJSON = json;
	});
}

function updateThreadHTML() {
	let thread = currentThread();
	if(thread.thread === 0) return; // not in a thread
	let numAdded = 0;
	for(const post of currentThreadJSON.posts) {
		let selector = "";
		if(post.resto === 0)
			selector += `div#${post.no}.thread`;
		else
			selector += `a#${post.no}.anchor`;
		let elementExists = $(selector).length > 0;
		if(elementExists)
			continue; // TODO: check for edits
		
		let $post = createPostElement(post, thread.board, "reply");
		let $replyContainer = $("<div/>").prop({
			id: `replycontainer${post.no}`,
			class: "reply-container"
		}).append($post);
		$replyContainer.appendTo(`div#${post.resto}.thread`);
		addPostDropdown($post);
		numAdded++;
	}
	if(numAdded === 0) return;
}

export function updateThread() {
	return updateThreadJSON().then(updateThreadHTML);
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
	if($hoverPreview === null) return;
	$hoverPreview.css({position: "absolute"}).offset({
		top: e.pageY + 8,
		left: e.pageX + 8
	});
}

function expandPost(e) {
	e.preventDefault();
	if($hoverPreview !== null) $hoverPreview.remove();
	let $next = $(e.target).next();
	if($next.prop("class") == "inlinepostprev" && e.type == "click") {
		// inline preview is already opened, close it
		$next.remove();
		return;
	}
	let href = e.target.href;
	let hrefArr = postrefRE.exec(href);
	if(hrefArr === null) return; // not actually a link to a post, abort
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
	if(getPageThread().board == "" && $post === null) return;
	doClickPreview = getBooleanStorageVal("enablepostclick", true);
	doHoverPreview = getBooleanStorageVal("enableposthover", false);
	let $refs = null;
	$refs = $post === null ? $("a.postref") : $post.find("a.postref");

	if(doClickPreview) {
		$refs.on("click", expandPost);
	} else {
		$refs.off("click", expandPost);
	}

	if(doHoverPreview) {
		$refs.on("mouseenter", expandPost).on("mouseleave", () => {
			if($hoverPreview !== null) $hoverPreview.remove();
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

function selectedText() {
	if(!window.getSelection) return "";
	return window.getSelection().toString();
}

export function quote(no) {
	if(getBooleanStorageVal("useqr", true)) {
		openQR();
	}
	let msgboxID = "postmsg";	

	let msgbox = document.getElementById("qr" + msgboxID);
	if(msgbox === null)
		msgbox = document.getElementById(msgboxID);
	let selected = selectedText();
	let lines = selected.split("\n");

	if(selected !== "") {
		for(let l = 0; l < lines.length; l++) {
			lines[l] = ">" + lines[l];
		}
	}
	let cursor = (msgbox.selectionStart !== undefined)?msgbox.selectionStart:msgbox.value.length;
	let quoted = lines.join("\n");
	if(quoted != "") quoted += "\n";
	msgbox.value = msgbox.value.slice(0, cursor) + `>>${no}\n` +
		quoted + 
		msgbox.value.slice(cursor);
	
	if(msgbox.id == "postmsg")
		window.scroll(0,msgbox.offsetTop - 48);
	msgbox.focus();
}
window.quote = quote;

export function stopThreadWatcher() {
	clearInterval(threadWatcherInterval);
}

$(() => {
	let pageThread = getPageThread();
	if(pageThread.op >= 1) {
		threadWatcherInterval = setInterval(updateThread, getNumberStorageVal("watcherseconds", 10) * 1000);
	}
});
