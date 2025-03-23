const embedWidth = 400;
const embedHeight = 300;

function closeEmbedClicked(e) {
	e.preventDefault();
	const $this = $(this);
	const $fileInfo = $this.parents(".file-info");
	const $uploadContainer = $fileInfo.siblings("a.upload-container");
	const $embed = $uploadContainer.find(".embed");
	if($embed.hasClass("embed-rawvideo")) {
		$embed.attr({
			"controls": false,
			"width": $embed.attr("orig-width"),
			"height": $embed.attr("orig-height")
		}).css({
			"max-width": $embed.attr("orig-width"),
			"max-height": $embed.attr("orig-height")
		});
		$embed[0].pause();
		$embed[0].currentTime = 0;
		$fileInfo.find(".close-container").remove();
		return;
	} else {
		$embed.addClass("thumb");
		$uploadContainer.show();
		$fileInfo.siblings(".embed.video").remove();
	}
	$this.parent().remove();
}


$("a.upload-container").filter((_i,el) =>  el.querySelector(".embed")).on("click", function(e) {
	e.preventDefault();
	const $this = $(this);
	const $embed = $this.find(".embed");
	const $fileInfo = $this.siblings(".file-info");
	const videoURL = new URL($fileInfo.find("a.embed-orig").attr("href"));
	$this.parent(".reply").find(".embed.video").remove();
	if($embed.hasClass("embed-youtube") && $embed.hasClass("thumb")) {
		const videoID = videoURL.searchParams.get("v");
		$embed.removeClass("thumb");
		$embed.attr("thumb-src", $embed.attr("src"));
		$fileInfo.after(`<iframe class="embed video embed-youtube" src="https://www.youtube.com/embed/${videoID}" frameborder="0" width="${embedWidth}" height="${embedHeight}" allowfullscreen></iframe>`);
		$this.hide();
		$fileInfo.append(` <span class="close-container">[<a class="close-thumb" href="#">Close</a>]</span>`);
		$fileInfo.find("a.close-thumb").on("click", closeEmbedClicked);
	} else if($embed.hasClass("embed-vimeo") && $embed.hasClass("thumb")) {
		const videoID = videoURL.pathname.split("/").pop();
		$embed.removeClass("thumb");
		$embed.attr("thumb-src", $embed.attr("src"));
		$fileInfo.after(`<iframe class="embed video embed-vimeo" src="https://player.vimeo.com/video/${videoID}" frameborder="0" width="${embedWidth}" height="${embedHeight}" allowfullscreen></iframe>`);
		$this.hide();
		$fileInfo.append(` <span class="close-container">[<a class="close-thumb" href="#">Close</a>]</span>`);
		$fileInfo.find("a.close-thumb").on("click", closeEmbedClicked);
	} else if($embed.hasClass("embed-rawvideo")) {
		if($embed[0].tagName !== "VIDEO") {
			return;
		}
		$embed.attr({
			"orig-width": $embed.width(),
			"orig-height": $embed.height(),
		});
		$embed.attr({
			"controls": true,
			"width": embedWidth,
			"height": embedHeight
		}).css({
			"max-width": embedWidth,
			"max-height": embedHeight
		});
		$fileInfo.append(` <span class="close-container">[<a class="close-thumb" href="#">Close</a>]</span>`);
		$fileInfo.find("a.close-thumb").on("click", closeEmbedClicked);
	}
});