import $ from "jquery";

interface BannerProps {
	src: string;
	alt: string;
	width?: number;
	height?: number;
}

export function setPageBanner() {
	const slashArr = location.pathname.split("/");
	const board = (slashArr.length >= 2)?slashArr[1]:"";

	$.get({
		url: `${webroot}util/banner`,
		data: {
			board: board
		},
		dataType: "json"
	}).then(data => {
		if(!data || data.Filename === undefined || data.Filename === "") {
			return; // no banners :(
		}
		const props: BannerProps = {
			src: `${webroot}static/banners/${data.Filename}`,
			alt: "Page banner"
		};
		if(data.Width > 0 && data.Height > 0) {
			props.width = data.Width;
			props.height = data.Height;
		}
		$("<img/>").prop(props).insertBefore("header h1#board-title");
	});
}