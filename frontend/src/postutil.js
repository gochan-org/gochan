let movablePostPreviews = null;
let expandablePostrefs = true;
let threadRE = /^\d+/;
let videoTestRE = /\.(mp4)|(webm)$/;

function deleteCheckedPosts() {
	if(confirm('Are you sure you want to delete these posts?')) {
		let form = $("form#main-form");
		form.append("<input type=\"hidden\" name=\"action\" value=\"delete\" ");
		form.get(0).submit();
		return true;
	}
	return false;
}
window.deleteCheckedPosts = deleteCheckedPosts;

export function deletePost(id, board) {
	let password = prompt("Password");
	// if(password == "") return;
	// let xhrFields = {
	// 	board: board,
	// 	report_btn: "Report",
	// 	password: password
	// }
	// xhrFields[`check${id}`] = "on";
	// $.ajax({
	// 	url: webroot + "/util",
	// 	method: "POST",
	// 	xhrFields: xhrFields,
	// 	success: function() {
	// 		console.log(arguments);
	// 	},
	// 	error: function() {
	// 		console.log(arguments);
	// 	}
	// });
	//window.location = webroot + "util?action=delete&posts="+id+"&board="+board+"&password";
}
window.deletePost = deletePost;

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

export function preparePostPreviews(isInline) {
	let mType = "mousemove";
	if(!movablePostPreviews) mType = "mouseover";

	var hvrStr = "a.postref";
	if(isInline) hvrStr = "div.inlinepostprev " + hvrStr;

	let $hover = $(hvrStr)
	$hover.on("mouseenter", function() {
		console.log("mouseenter");
		let replaced = $hover[0].innerHTML.replace("&gt;&gt;","");
		let postID = `div.reply#reply${replaced},div.op-post#op${replaced}`;
		let $clone = $(postID).clone();
		$(document.body).append($clone.attr({
			class: "postprev",
			id: postID + "preview"
		}));
		$clone.find(".inlinepostprev").remove();
		$(document).on(mType, e => {
			$('.postprev').css({
				left:	e.pageX + 8,
				top:	e.pageY + 8
			});
		});
	}).on("mouseleave", () => {
		console.log("mouseleave")
		$(".postprev").remove();
	});

	if(expandablePostrefs) {
		let clkStr = "a.postref";
		if(isInline) clkStr = "div.inlinepostprev " + clkStr;
		$(clkStr).on("click", function() {
			let $this = $(this);
			if($this.next().attr("class") != "inlinepostprev") {
				$(".postprev").remove();
				let replaced = this.innerHTML.replace("&gt;&gt;","");
				let postID = `div.reply#reply${replaced},div.op-post#op${replaced}`;
				let $clone = $(postID).clone()
				$clone.find("postprev").remove();
				$this.after(
					$clone.attr("class", "inlinepostprev")
				);
			} else {
				$this.next().remove();
			}
			return false;
		});
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
			.on("click", e =>{
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

export function reportPost(id, board) {
	let reason = prompt("Reason");
	if(reason == "") return;
	// let xhrFields = {
	// 	board: board,
	// 	report_btn: "Report",
	// 	reason: reason
	// }
	// xhrFields[`check${id}`] = "on";
	// $.ajax({
	// 	url: webroot + "/util",
	// 	method: "POST",
	// 	xhrFields: xhrFields,
	// 	success: function() {
	// 		console.log(arguments);
	// 	},
	// 	error: function() {
	// 		console.log(arguments);
	// 	}
	// });
}
window.reportPost = reportPost;