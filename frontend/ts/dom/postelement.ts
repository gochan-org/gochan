import $ from "jquery";
import { extname } from "path";
import { formatDateString, formatFileSize } from "../formatting";
import { getThumbFilename } from "../postinfo";
import { getBooleanStorageVal } from "../storage";

/**
 * creates an element from the given post data
 */
export function createPostElement(post: ThreadPost, boardDir: string, elementClass = "inlinepostprev") {
	const $post = $("<div/>")
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
	const $postInfo = $post.find("label.post-info");
	const postName = (post.name === "" && post.trip === "")?"Anonymous":post.name;
	const $postName = $("<span/>").prop({class: "postername"});
	if(post.email === "") {
		$postName.text(postName);
	} else {
		$postName.append($("<a/>").prop({
			href: "mailto:" + post.email
		}).text(post.name));
	}
	$postInfo.prepend($postName);
	if(post.trip !== "") {
		$postInfo.prepend($postName, $("<span/>").prop({class: "tripcode"}).text("!" + post.trip), " ");
	} else {
		$postInfo.prepend($postName, " ");
	}

	if(post.sub !== "")
		$postInfo.prepend($("<span/>").prop({class:"subject"}).text(post.sub), " ");

	if(post.filename !== "" && post.filename !== "deleted") {
		const thumbFile = getThumbFilename(post.tim);
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

export function shrinkOriginalFilenames(elem = $(document.body)) {
	elem.find<HTMLAnchorElement>("a.file-orig").each((i, el) => {
		const ext = extname(el.innerText);
		const noExt = el.innerText.slice(0,el.innerText.lastIndexOf("."));
		if(noExt.length > 16) {
			const trimmed = noExt.slice(0, 15).trim() + "…" + ext;
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

export function prepareHideBlocks() {
	$("div.hideblock").each((_i,el) => {
		const $el = $(el);
		const $button = $("<button />").prop({
			class: "hideblock-button",
		}).text($el.hasClass("open") ? "Hide" : "Show").on("click", e => {
			e.preventDefault();
			const hidden = $el.hasClass("hidden");
			$button.text(hidden ? "Hide" : "Show");
			if(el.onanimationend === undefined || !getBooleanStorageVal("smoothhidetoggle", true)) {
				$el.toggleClass("hidden");
			} else {
				$el.removeClass("close");
				if(hidden) {
					$el.removeClass("hidden").addClass("open");
				} else {
					$el.addClass("close").removeClass("open");
				}
			}
		}).insertBefore($el);
		$el.on("animationend", () => {
			if($el.hasClass("close")) {
				$el.addClass("hidden").removeClass("close");
			}
		});
	});
}

$(() => {
	prepareHideBlocks();
	shrinkOriginalFilenames();
});