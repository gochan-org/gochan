import path from "path";

import { $qr } from "./qr";

function updateFlagPreview($sel: JQuery<HTMLSelectElement>) {
	const $preview = $sel.next("img");
	let val = $sel.val() as string;
	if(val === "" || val === "geoip") {
		val = "blank.gif";
	}
	$preview.attr("src", path.join(webroot || "/", "static/flags", val));
}

function setupQRFlags($flags: JQuery<HTMLSelectElement>) {
	if(!$qr) return;
	$qr.find("div#qrbuttons").before(
		$("<div/>").append(
			$flags.clone(true, true).attr("id", "qrpost-flag"),
			$("<img/>").addClass("flag-preview")
				.attr("src", path.join(webroot || "/", "static/flags/blank.gif"))
		)
	);
}


export function initFlags() {
	const $flagChanger = $<HTMLSelectElement>("select")
		.filter((_, el) => el.name == "post-flag");
	if($flagChanger.length < 1) return;

	updateFlagPreview($flagChanger);
	$flagChanger.on("change", (ev:JQuery.ChangeEvent) => {
		updateFlagPreview($(ev.target))
	});
	setupQRFlags($flagChanger);
}