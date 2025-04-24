import $ from "jquery";

import "./vars";
import "./cookies";
import "./notifications";
import { setPageBanner } from "./dom/banners";
import { setCustomCSS, setCustomJS, setTheme, updateExternalLinks, updateSpoilerTextReveal, updateSpoilerThreadReveal } from "./settings";
import { handleKeydown } from "./boardevents";
import { initStaff, createStaffMenu, addStaffThreadOptions } from "./management/manage";
import { getPageThread } from "./postinfo";
import { prepareThumbnails, initPostPreviews } from "./postutil";
import { addPostDropdown } from "./dom/postdropdown";
import { initFlags } from "./dom/flags";
import { initQR } from "./dom/qr";
import { getBooleanStorageVal } from "./storage";
import { updateBrowseButton } from "./dom/uploaddata";
import "./management/filters";

export function toTop() {
	window.scrollTo(0,0);
}
window.toTop = toTop;

export function toBottom() {
	window.scrollTo(0,document.body.scrollHeight);
}
window.toBottom = toBottom;

const pageThread = getPageThread();
if(pageThread.board !== "") {
	prepareThumbnails();
	if(pageThread.op < 1) {
		updateSpoilerThreadReveal();
		updateSpoilerTextReveal();
	}
	
	updateBrowseButton();
}

setTheme();
setCustomCSS();
setCustomJS();
setPageBanner();

$(() => {
	initStaff()
		.then((staff) => {
			if(staff?.rank < 1)
				return;
			createStaffMenu(staff);
			if(staff.rank >= 2)
				addStaffThreadOptions();
		}).catch(() => {
			// not logged in
		});

	const passwordText = $("input#postpassword").val();
	$("input#delete-password").val(passwordText);

	if(pageThread.board !== "") {
		initPostPreviews();
	}
	$("div.post, div.reply").each((i, elem) => {
		addPostDropdown($(elem));
	});
	$(document).on("keydown", handleKeydown);
	initFlags();
	updateExternalLinks();
});
