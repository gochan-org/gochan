import $ from "jquery";

interface BannerAttributes {
	src: string;
	alt: string;
	width?: number;
	height?: number;
}

export function setPageBanner() {
	const slashArr = location.pathname.split("/");
	const board = (slashArr.length >= 2)?slashArr[1]:"";
	const $bannerImg = $<HTMLImageElement>("<img/>").attr({
		src: "/static/banners/gochan_go-parody.png", // placeholder, may or may not actually exist
		width: 300,
		height: 100,
		alt: "Page banner",
	}).insertBefore("header h1#board-title");

	$.get({
		url: `${webroot}util/banner`,
		data: {
			board: board
		},
		dataType: "json"
	}).then(data => {
		if((data?.Filename ?? "") === "") {
			// no banners :(
			$bannerImg.remove();
			return;
		}
		const attributes: BannerAttributes = {
			src: `${webroot}static/banners/${data.Filename}`,
			alt: "Page banner"
		};
		if(data.Width > 0 && data.Height > 0) {
			attributes.width = data.Width;
			attributes.height = data.Height;
		}
		$bannerImg.attr(attributes);
	});
}