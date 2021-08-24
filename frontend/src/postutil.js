let movablePostPreviews = null;
let expandablePostrefs = true;
let videoTestRE = /\.(mp4)|(webm)$/;
function deleteCheckedPosts() {
	if(confirm('Are you sure you want to delete these posts?') == true) {
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
	if(isInline) hvr_str = "div.inlinepostprev "+hvr_str;

	$(hvrStr).hover(() => {
		let replaced = this.innerHTML.replace("&gt;&gt;","");
		let postID = `div.reply#reply${replaced},div.op-post#op${replaced}`;
		let $clone = $(postID).clone();
		$(document.body).append($clone.attr({
			class: "postprev",
			id: postID + "preview"
		}));
		$clone.find(".inlinepostprev").remove();
		$(document).bind(mType, e => {
			$('.postprev').css({
				left:	e.pageX + 8,
				top:	e.pageY + 8
			});
		});
	},
	() => {
		$(".postprev").remove();
	});

	if(expandablePostrefs) {
		let clkStr = "a.postref";
		if(isInline) clkStr = "div.inlinepostprev " + clkStr;
		$(clkStr).on("click", () => {
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

	if (document.selection) {
		document.getElementById(msgboxID).focus();
		let t = document.getselection.createRange();
		t.text = `>>${e}\n`;
	} else if(document.getElementById(msgboxID).selectionStart || "0" == document.getElementById(msgboxID).selectionStart) {
		let n = document.getElementById(msgboxID).selectionStart,
		o = document.getElementById(msgboxID).selectionEnd;
		document.getElementById(msgboxID).value = document.getElementById(msgboxID).value.substring(0, n) + ">>" + e + "\n" + document.getElementById(msgboxID).value.substring(o, document.getElementById(msgboxID).value.length)
	} else document.getElementById(msgboxID).value += `>>${e}\n`;
	window.scroll(0,document.getElementById(msgboxID).offsetTop - 48);
}

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