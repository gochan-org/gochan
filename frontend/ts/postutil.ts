import $ from "jquery";

import { alertLightbox } from "./dom/lightbox";
import { getBooleanStorageVal } from "./storage";
import { currentThread, getPageThread, insideOP } from "./postinfo";
import { addPostDropdown } from "./dom/postdropdown";
import { createPostElement } from "./dom/postelement";
import { getThreadJSON } from "./api/threads";
import { openQR } from "./dom/qr";

let doClickPreview = false;
let doHoverPreview = false;
let $hoverPreview: JQuery<HTMLElement> = null;

const videoTestRE = /\.(mp4)|(webm)$/;
const imageTestRE = /\.(gif)|(jfif)|(jpe?g)|(png)|(webp)$/;
const postrefRE = /\/([^\s/]+)\/res\/(\d+)\.html(#(\d+))?/;

// data retrieved from /<board>/res/<thread>.json
let currentThreadJSON: BoardThread = {
	posts: []
};

export function getUploadPostID(upload: any, container: any) {
	// if container, upload is div.upload-container
	// otherwise it's img or video
	const jqu = container? $(upload) : $(upload).parent();
	return insideOP(jqu) ? jqu.siblings().eq(4).text() : jqu.siblings().eq(3).text();
}

export async function updateThreadJSON() {
	const thread = currentThread();
	if(thread.id === 0) return; // not in a thread
	const json = await getThreadJSON(thread.id, thread.board);
	if(!(json.posts instanceof Array) || json.posts.length === 0)
		return;
	currentThreadJSON = json;
}

function updateThreadHTML() {
	const thread = currentThread();
	if(thread.id === 0) return; // not in a thread
	let numAdded = 0;
	for(const post of currentThreadJSON.posts) {
		let selector = "";
		if(post.resto === 0 || post.resto === post.no)
			selector += `div#op${post.no}`;
		else
			selector += `div#reply${post.no}`;
		const elementExists = $(selector).length > 0;
		if(elementExists)
			continue; // TODO: check for edits
		
		const $post = createPostElement(post, thread.board, "reply");
		const $replyContainer = $("<div/>").prop({
			id: `replycontainer${post.no}`,
			class: "reply-container"
		}).append($post);
		$replyContainer.appendTo(`div#${post.resto}.thread`);
		addPostDropdown($post);
		prepareThumbnails($post);
		initPostPreviews($post);
		numAdded++;
	}
	if(numAdded === 0) return;
}

export function updateThread() {
	return updateThreadJSON().then(updateThreadHTML);
}

function createPostPreview(e: JQuery.MouseEventBase, $post: JQuery<HTMLElement>, inline = true) {
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

function previewMoveHandler(e: JQuery.Event) {
	if($hoverPreview === null) return;
	$hoverPreview.css({position: "absolute"}).offset({
		top: e.pageY + 8,
		left: e.pageX + 8
	});
}

function expandPost(e: JQuery.MouseEventBase) {
	e.preventDefault();
	if($hoverPreview !== null) $hoverPreview.remove();
	const $next = $(e.target).next();
	if($next.prop("class") === "inlinepostprev" && e.type === "click") {
		// inline preview is already opened, close it
		$next.remove();
		return;
	}
	const href = e.target.href;
	const hrefArr = postrefRE.exec(href);
	if(hrefArr === null) return; // not actually a link to a post, abort
	const postID = hrefArr[4]?hrefArr[4]:hrefArr[2];

	let $post = $(`div#op${postID}, div#reply${postID}`).first();
	if($post.length > 0) {
		const $preview = createPostPreview(e, $post, e.type === "click");
		if(e.type === "mouseenter") {
			$hoverPreview = $preview.insertAfter(e.target);
			$(document.body).on("mousemove", previewMoveHandler);
		}
		return;
	}
	if(e.type === "click") {
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

export function initPostPreviews($post: JQuery<HTMLElement> = null) {
	if(getPageThread().board === "" && $post === null) return;
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

/**
 * Sets thumbnails to expand when clicked. If a parent is provided, prepareThumbnails will only
 * be applied to that parent
 * @param $post the post (if set) to prepare the thumbnails for
 */
export function prepareThumbnails($parent: JQuery<HTMLElement> = null) {
	const $container = $parent === null ? $("a.upload-container") : $parent.find("a");
	$container.on("click", function(e) {
		const $a = $(this);
		const uploadHref = $a.siblings("div.file-info").children("a.file-orig").attr("href");
		if(imageTestRE.exec(uploadHref) === null && videoTestRE.exec(uploadHref) === null)
			return true; // not an image or a video

		e.preventDefault();

		const $thumb = $a.find("img.upload");
		const thumbURL = $thumb.attr("src");
		const uploadURL = $thumb.attr("alt");
		$thumb.removeAttr("width").removeAttr("height");

		const $fileInfo = $a.prevAll(".file-info:first");
		
		if(videoTestRE.test(thumbURL + uploadURL)) {
			// Upload is a video
			$thumb.hide();
			const $video = $("<video />")
				.prop({
					src: uploadURL,
					autoplay: true,
					controls: true,
					class: "upload",
					loop: true
				}).insertAfter($fileInfo);

			$fileInfo.append($("<a />")
				.prop("href", "javascript:;").on("click", function() {
					$video.remove();
					$thumb.show();
					this.remove();
					$thumb.prop({
						src: thumbURL,
						alt: uploadURL
					});
				}).css({
					"padding-left": "8px"
				}).html("[Close]<br />"));
		} else {
			// upload is an image
			$thumb.attr({
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

export function quote(no: number) {
	if(getBooleanStorageVal("useqr", true)) {
		openQR();
	}
	const msgboxID = "postmsg";	
	let msgbox = document.getElementById("qr" + msgboxID) as HTMLInputElement;
	if(msgbox === null)
		msgbox = document.getElementById(msgboxID) as HTMLInputElement;
	const selected = selectedText();
	const lines = selected.split("\n");

	if(selected !== "") {
		for(let l = 0; l < lines.length; l++) {
			lines[l] = ">" + lines[l];
		}
	}
	const cursor = (msgbox.selectionStart !== undefined)?msgbox.selectionStart:msgbox.value.length;
	let quoted = lines.join("\n");
	if(quoted !== "") quoted += "\n";
	msgbox.value = msgbox.value.slice(0, cursor) + `>>${no}\n` +
		quoted + msgbox.value.slice(cursor);
	
	if(msgbox.id === "postmsg")
		window.scroll(0,msgbox.offsetTop - 48);
	msgbox.focus();
}
window.quote = quote;
