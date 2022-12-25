/* global webroot */
/**
 * @typedef { import("../types/gochan").BoardThread } BoardThread
 * @typedef { import("../types/gochan").ThreadPost } ThreadPost
 */


import $ from "jquery";
import { extname } from "path";
import { formatDateString, formatFileSize } from "../formatting";
import { getThumbFilename } from "../postinfo";

/**
 * creates an element from the given post data
 * @param {ThreadPost} post
 * @param {string} boardDir
 */
export function createPostElement(post, boardDir, elementClass = "inlinepostprev") {
	let $post = $("<div/>")
		.prop({
			id: `reply${post.no}`,
			class: elementClass
		});
	$post.append(
		$("<a/>").prop({
			id: post.no.toString(),
			class: "anchor"
		}),
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
			}).append(formatDateString(post.time)),
		" ",
		$("<a/>")
			.prop({
				href: webroot + boardDir + "/res/" + ((post.resto > 0)?post.resto:post.no) + ".html#" + post.no
			}).text("No."),
		" ",
		$("<a/>")
			.prop({
				class: "backlink-click",
				href: `javascript:quote(${post.no})`
			}).text(post.no), "<br/>",
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
					` - (${formatFileSize(post.fsize)} , ${post.w}x${post.h}, `,
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
		shrinkOriginalFilenames($post);
	}
	$post.append(
		$("<div/>").prop({
			class: "post-text"
		}).html(post.com)
	);
	return $post;
}

/**
 * @param {JQuery<HTMLElement>} elem
 */
export function shrinkOriginalFilenames(elem) {
	if(elem == undefined)
		elem = $(document.body);

	elem.find("a.file-orig").each((i, el) => {
		let ext = extname(el.innerText);
		let noExt = el.innerText.slice(0,el.innerText.lastIndexOf("."));
		if(noExt.length > 16) {
			trimmed = noExt.slice(0, 15).trim() + "…" + ext;
			el.setAttribute("trimmed", trimmed);
			el.text = el.getAttribute("trimmed");
			$(el).on("mouseover", () => {
				el.text = el.getAttribute("download");
			}).on("mouseout", () => {
				el.text = el.getAttribute("trimmed");
			});
		}
	});
}


$(() => {
	shrinkOriginalFilenames();
});