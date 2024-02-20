import $ from "jquery";
import path from "path-browserify";

import { alertLightbox } from "../dom/lightbox";
import { $topbar, TopBarButton } from "../dom/topbar";
import "./sections";
import "./filebans";
import "./viewlog";
import { isThreadLocked } from "../api/management";

const reportsTextRE = /^Reports( \(\d+\))?/;

export let staffActions: StaffAction[] = [];
let staffInfo: StaffInfo = null;

/**
 * The menu shown when the Staff button on the top bar is clicked
 */
let $staffMenu: JQuery<HTMLElement> = null;

/**
 * A button that opens $staffMenu
 */
let $staffBtn: TopBarButton = null;


function dropdownHasItem(dropdown: any, item: string) {
	return [...dropdown.children].filter(v => v.text === item).length > 0;
}

function addManageEvents(_i: number, el: HTMLSelectElement) {
	if(staffInfo === null || staffInfo.rank < 2) return;
	const $el = $(el);
	const $post = $(el.parentElement);
	const isLocked = isThreadLocked($post);
	const $thumb = $post.find("img.upload");

	if(!dropdownHasItem(el, "Staff Actions")) {
		$el.append('<option disabled="disabled">Staff Actions</option>');
	}

	if(staffInfo.rank === 3 && $post.hasClass("op-post")) {
		if(isLocked) {
			$el.append("<option>Unlock thread</option>");
		} else {
			$el.append("<option>Lock thread</option>");
		}
	}
	if(staffInfo.rank >= 2) {
		if(!dropdownHasItem(el, "Posts from this IP")) {
			$el.append("<option>Posts from this IP</option>");
		}
		if(!dropdownHasItem(el, "Ban IP address")) {
			$el.append("<option>Ban IP address</option>");
		}
	}

	if($thumb.length > 0) {
		const fpOpts = staffInfo.fingerprinting;
		const uploadExt = path.extname($thumb.attr("alt")).toLowerCase();
		const isImage = fpOpts.imageExtensions.indexOf(uploadExt) > -1;
		const isVideo = fpOpts.videoExtensions.indexOf(uploadExt) > -1;
		if(!dropdownHasItem(el, "Ban filename")) {
			$el.append(
				"<option>Ban filename</option>",
				"<option>Ban file checksum</option>"
			);
		}
		if(isImage || (isVideo && fpOpts.fingerprintVideoThumbs)) {
			if(!dropdownHasItem(el, "Ban fingerprint")) {
				$el.append("<option>Ban fingerprint</option>");
			}
			if(!dropdownHasItem(el, "Ban fingerprint (IP ban)")) {
				$el.append("<option>Ban fingerprint (IP ban)</option>");
			}
		}
	}
}

function setupManagementEvents() {
	if(staffInfo === null || !staffInfo.actions) return;
	$<HTMLSelectElement>("select.post-actions").each(addManageEvents);
	$(document).on("postDropdownAdded", function(_e, data) {
		if(!data.dropdown) return;
		data.dropdown.append("<option>Posts from this IP</option>");
		data.dropdown.append("<option>Ban IP address</option>");
	});
}

interface BanFileJSON {
	bantype: string;
	board?: string;
	fingerprinter?: string;
	json: number;
	staffnote: string;
	ban?: string;
	banmsg?: string;
	filename?: string;
	dofilenameban?: string;
	checksum?: string;
	dochecksumban?: string;
}

export function banFile(banType: string, filename: string, checksum: string, staffNote = "") {
	const xhrFields: BanFileJSON = {
		bantype: banType,
		staffnote: staffNote,
		json: 1
	};
	switch(banType) {
	case "filename":
		xhrFields.filename = filename;
		xhrFields.dofilenameban = "Create";
		break;
	case "checksum":
		xhrFields.checksum = checksum;
		xhrFields.dochecksumban = "Create";
		break;
	default:
		break;
	}
	return $.post({
		url: `${webroot}manage/filebans`,
		data: xhrFields
	});
}

export function banFileFingerprint(fingerprint: string, ipBan: boolean, reason?: string, staffNote?: string) {
	const xhrFields: BanFileJSON = {
		bantype: "checksum",
		checksum: fingerprint,
		fingerprinter: fingerprint,
		ban: ipBan?"on":"",
		banmsg: reason,
		staffnote: staffNote,
		json: 1,
		dochecksumban: "Create"
	};
	return $.post({
		url: `${webroot}manage/filebans`,
		data: xhrFields
	});
}


export async function initStaff() {
	if(staffInfo !== null || staffActions?.length > 0)
		// don't make multiple unnecessary AJAX requests
		return staffInfo;

	return $.ajax({
		method: "GET",
		url: `${webroot}manage/staffinfo`,
		async: true,
		cache: false,
		dataType: "json",
		success: (result:string|StaffInfo) => {
			if(typeof result === "string") {
				try {
					staffInfo = JSON.parse(result);
				} catch(e) {
					// presumably not logged in
					staffActions = [];
				}
			} else if(typeof result === "object") {
				staffInfo = result;
			}
			staffActions = staffInfo.actions;
			return staffInfo;
		},
		error: (e: JQuery.jqXHR) => {
			console.error("Error getting actions list:", e);
		}
	}).then(() => {
		if(staffInfo.rank > 0)
			setupManagementEvents();
		return staffInfo;
	});
}

export async function getPostInfo(id: number):Promise<PostInfo> {
	return $.ajax({
		method: "GET",
		url: `${webroot}manage/postinfo`,
		data: {
			postid: id
		},
		async: true,
		cache: true,
		dataType: "json"
	});
}

export async function isLoggedIn() {
	return initStaff().then(info => {
		return info.rank > 0;
	});
}

export function banSelectedPost() {
	const boardDirArr = location.pathname.split("/");
	if(boardDirArr.length < 2) return;
	const boardDir = boardDirArr[1];
	const checks = $("input[type=checkbox]");
	if(checks.length === 0) {
		alertLightbox("No posts selected");
		return false;
	}
	let postID = 0;
	for(let i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") === 0) {
			postID = Number.parseInt(checks[i].id.replace("check", ""));
			break;
		}
	}
	window.location.pathname = `${webroot}manage/bans?dir=${boardDir}&postid=${postID}`;
}

/**
 * A helper function for creating a menu item
 */
function menuItem(action: StaffAction|string, isCategory = false) {
	return isCategory ? $("<div/>").append($("<b/>").text(action as string)) : $("<div/>").append(
		$("<a/>").prop({
			href: `${webroot}manage/${(action as StaffAction).id}`
		}).text((action as StaffAction).title)
	);
}

function getAction(id: string) {
	for(const action of staffActions) {
		if(action.id === id) {
			return action;
		}
	}
}

function filterAction(action: StaffAction, perms: number) {
	return action.title !== "Logout"
		&& action.title !== "Dashboard"
		&& action.jsonOutput < 2
		&& action.perms === perms;
}

/**
 * Creates a list of staff actions accessible to the user if they are logged in.
 * It is shown when the user clicks the Staff button
 * @param staff an object representing the staff's username and rank
 */
export function createStaffMenu(staff = staffInfo) {
	const rank = staff.rank;
	if(rank === 0) return;
	$staffMenu = $("<div/>").prop({
		id: "staffmenu",
		class: "dropdown-menu"
	});

	$staffMenu.append(
		menuItem(getAction("logout")),
		menuItem(getAction("dashboard")));

	const janitorActions = staffActions.filter(val => filterAction(val, 1));
	$staffMenu.append(menuItem("Janitorial", true));
	for(const action of janitorActions) {
		$staffMenu.append(menuItem(action));
	}

	if(rank >= 2) {
		const modActions = staffActions.filter(val => filterAction(val, 2));
		if(modActions.length > 0)
			$staffMenu.append(menuItem("Moderation", true));
		for(const action of modActions) {
			$staffMenu.append(menuItem(action));
		}
		getReports().then(updateReports);
	}
	if(rank === 3) {
		const adminActions = staffActions.filter(val => filterAction(val, 3));
		if(adminActions.length > 0)
			$staffMenu.append(menuItem("Administration", true));
		for(const action of adminActions) {
			$staffMenu.append(menuItem(action));
		}
	}
	createStaffButton();
}

function createStaffButton() {
	if($staffBtn !== null || staffInfo === null || staffInfo.rank === 0)
		return;
	$staffBtn = new TopBarButton("Staff", () => {
		$topbar.trigger("menuButtonClick", [$staffMenu, $(document).find($staffMenu).length === 0]);
	});
}

function updateReports(reports: any[]) {
	// append " (#)" to the Reports link, replacing # with the number of reports
	$staffMenu.find("a").each((e, elem) => {
		if(elem.text.search(reportsTextRE) !== 0) return;
		const $span = $("<span/>").text(` (${reports.length})`).appendTo(elem);
		if(reports.length > 0) {
			// make it bold and red if there are reports
			$span.css({
				"font-weight": "bold",
				"color": "red"
			});
		}
	});
}

function getReports() {
	return $.ajax({
		method: "GET",
		url: `${webroot}manage/reports`,
		data: {
			json: "1"
		},
		async: true,
		cache: false,
		dataType: "json"
	}).catch(e => {
		return e;
	});
}