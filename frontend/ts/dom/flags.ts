import path from "path";

import { $qr } from "./qr";

function updateFlagPreview($sel: JQuery<HTMLSelectElement>) {
	const $preview = $sel.next("img");
	let val = $sel.val() as string;
	if(val === "" || val === "geoip") {
		val = "blank.gif";
	}
	$preview.attr("src", path.join(webroot ?? "/", "static/flags", val));
}

function setupQRFlags($flags: JQuery<HTMLSelectElement>) {
	if(!$qr) return;
	$qr.find("div#qrbuttons").before(
		$("<div/>").append(
			$flags.clone(true, true).attr("id", "qrpost-flag"),
			$("<img/>").addClass("flag-preview")
				.attr("src", path.join(webroot ?? "/", "static/flags/blank.gif"))
		)
	);
}

function getBoard() {
	const pathParts = location.pathname.split("/");
	if(pathParts.length < 2) return null;
	return pathParts[1];
}

function loadFlagSelection() {
	const board = getBoard();
	const savedFlag = localStorage.getItem(`flag_${board}`);
	if(board !== null && savedFlag !== null) {
		const $sel = $<HTMLSelectElement>("select")
			.filter((_,el) => el.name === "post-flag");
		const num = $sel.find("option")
			.filter((_,el) => el.value === savedFlag).length;
		if(num > 0) {
			$sel.val(savedFlag);
			$sel.trigger("change");
		}
	}
}

function saveFlagSelection(ev: JQuery.SubmitEvent) {
	const board = getBoard();
	if(!board) return;
	const flag = $(ev.target)
		.find<HTMLSelectElement>("select")
		.filter((_,ev) => ev.name === "post-flag")
		.val() as string;
	localStorage.setItem(`flag_${board}`, flag);
}

export function initFlags() {
	const $flagChanger = $<HTMLSelectElement>("select")
		.filter((_, el) => el.name === "post-flag");
	if($flagChanger.length < 1) return;

	updateFlagPreview($flagChanger);
	$flagChanger.on("change", (ev:JQuery.ChangeEvent) =>
		updateFlagPreview($(ev.target)));
	setupQRFlags($flagChanger);

	loadFlagSelection();
	$("form").filter((_,el) =>
		el.getAttribute("action") === path.join(webroot ?? "/", "post"))
		.on("submit", saveFlagSelection);
}