import { opRegex } from "./vars";
import { getCookie } from "./cookies";
import { alertLightbox, promptLightbox } from "./lightbox";
import { getBooleanStorageVal } from "./storage";
import { handleActions } from "./boardevents";

let doClickPreview = false;
let doHoverPreview = false;
let $hoverPreview = null;

let threadRE = /^\d+/;
let videoTestRE = /\.(mp4)|(webm)$/;
const postrefRE = /\/([^\s\/]+)\/res\/(\d+)\.html(#(\d+))?/;

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
	if(insideOP(jqu)) return jqu.siblings().eq(4).text();
	else return jqu.siblings().eq(3).text();
}

export function currentBoard() {
	// may or may not actually return the board. For example, if you're at
	// /manage?action=whatever, it will return "manage"
	let splits = location.pathname.split("/");
	if(splits.length > 1)
		return splits[1];
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

export function hidePost(id) {
	let posttext = $("div#"+id+".post .posttext");
	if(posttext.length > 0) posttext.remove();
	let fileinfo = $("div#"+id+".post .file-info")
	if(fileinfo.length > 0) fileinfo.remove();
	let postimg = $("div#"+id+".post img")
	if(postimg.length > 0) postimg.remove();
}

export function insideOP(elem) {
	return $(elem).parents("div.op-post").length > 0;
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
	let href = e.target.href
	let hrefArr = postrefRE.exec(href);
	if(hrefArr == null) return; // not actually a link to a post, abort
	let postID = hrefArr[4]?hrefArr[4]:hrefArr[2];

	let $post = $(`div#op${postID}, div#reply${postID}`).first();
	if($post.length > 0) {
		$preview = createPostPreview(e, $post, e.type == "click");
		if(e.type == "mouseenter") {
			$hoverPreview = $preview.insertAfter(e.target);
			$(document.body).on("mousemove", previewMoveHandler);
		}
		return
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
	if($post == null) {
		$refs = $("a.postref");
	} else {
		$refs = $post.find("a.postref");
	}

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
			.on("click", function(e) {
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
		msgbox.value = msgbox.value.substring(0, n) + ">>" + e + "\n" + msgbox.value.substring(o, msgbox.value.length)
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
	}).append(
		"<option disabled selected>Actions</option>",
	);
	if(isOP) {
		$ddownMenu.append(
			"<option>Watch thread</option>"
		);
	}
	$ddownMenu.append(
		`<option>Show/hide ${threadPost}</option>`,
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
		}).done(data => {
			alertLightbox("Report sent", "Success");
		}, "json");
	}, "Report post");
}
window.reportPost = reportPost;

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
			xhrFields["fileonly"] = "on";
		}
		$.post(webroot + "util", xhrFields).fail(data => {
			alertLightbox(`Delete failed: ${data["error"]}`, "Error");
		}).done(data => {
			if(data["error"] == undefined) {
				alertLightbox(`${fileOnly?"File from post":"Post"} #${id} deleted`, "Success");
			} else {
				alertLightbox(`Error deleting post #${id}: ${data["error"]}`, "Error");
			}
		}, "json");
	}, "Password");
}
window.deletePost = deletePost;