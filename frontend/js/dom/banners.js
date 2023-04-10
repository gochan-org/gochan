import $ from "jquery";

export function setPageBanner() {
	const data = {
		url: `${webroot}util/banner`,
		dataType: "json"
	};
	const board = location.pathname.split("/")[0];
	if(board !== "") {
		data.data = {
			board: board
		};
	}
	$.get(data).then(data => {
		if(!data || data.Filename == undefined || data.Filename == "") {
			return; // no banners :(
		}
		const props = {
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