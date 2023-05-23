import $ from "jquery";

import { alertLightbox } from "../dom/lightbox";
import { $topbar, TopBarButton } from "../dom/topbar";
import "./sections";
import "./filebans";
import { isThreadLocked } from "../api/management";

const notAStaff: StaffInfo = {
	ID: 0,
	Username: "",
	Rank: 0
};

const reportsTextRE = /^Reports( \(\d+\))?/;

export let staffActions: StaffAction[] = [];
export let staffInfo = notAStaff;
let loginChecked = false;

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

function setupManagementEvents() {
	$("select.post-actions").each((_i, el) => {
		const $el = $(el);
		const $post = $(el.parentElement);
		const isLocked = isThreadLocked($post);
		if(!dropdownHasItem(el, "Staff Actions")) {
			$el.append(`<option disabled="disabled">Staff Actions</option>`);
		}
		if($post.hasClass("op-post")) {
			if(isLocked) {
				$el.append("<option>Unlock thread</option>");
			} else {
				$el.append("<option>Lock thread</option>");
			}
		}
		if(!dropdownHasItem(el, "Posts from this IP")) {
			$el.append("<option>Posts from this IP</option>");
		}
		const filenameOrig = $post.find("div.file-info a.file-orig").text();
		if(filenameOrig != "" && !dropdownHasItem(el, "Ban filename")) {
			$el.append(
				"<option>Ban filename</option>",
				"<option>Ban file checksum</option>"
			);
		}
	});
	$(document).on("postDropdownAdded", function(_e, data) {
		if(!data.dropdown) return;
		data.dropdown.append("<option>Posts from this IP</option>");
	});
}

interface BanFileJSON {
	bantype: string;
	board: string;
	json: number;
	staffnote: string;
	filename?: string;
	dofilenameban?: string;
	checksum?: string;
	dochecksumban?: string;
}

export function banFile(banType: string, filename: string, checksum: string, staffNote = "") {
	const xhrFields: BanFileJSON = {
		bantype: banType,
		board: "",
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
	return $.ajax({
		method: "POST",
		url: `${webroot}manage/filebans`,
		data: xhrFields
	});
}

export async function initStaff() {
	return $.ajax({
		method: "GET",
		url: `${webroot}manage/actions`,
		async: true,
		cache: false,
		success: result => {
			if(typeof result === "string") {
				try {
					staffActions = JSON.parse(result);

				} catch(e) {
					// presumably not logged in
					staffActions = [];
				}
			} else if(typeof result === "object") {
				staffActions = result;
			}
		},
		error: (e: JQuery.jqXHR) => {
			console.error("Error getting actions list:", e);
		}
	}).then(getStaffInfo).then(info => {
		if(info.Rank > 0) {
			setupManagementEvents();
		}
		return info;
	});
	
}

export async function getStaffInfo() {
	if(loginChecked)
		// don't make multiple unnecessary AJAX requests if we're already logged in
		return staffInfo;
	loginChecked = true;
	return $.ajax({
		method: "GET",
		url: `${webroot}manage/staffinfo`,
		async: true,
		cache: true,
		dataType: "json"
	}).catch(() => {
		return notAStaff;
	}).then((info: any) => {
		if(info.error)
			return notAStaff;
		staffInfo = info;
		return info;
	});
}

export async function getPostInfo(id: number) {
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
	return getStaffInfo().then(info => {
		return info.ID > 0;
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
		if(action.id == id) {
			return action;
		}
	}
}

function filterAction(action: StaffAction, perms: number) {
	return action.title != "Logout"
		&& action.title != "Dashboard"
		&& action.jsonOutput < 2
		&& action.perms == perms;
}

/**
 * Creates a list of staff actions accessible to the user if they are logged in.
 * It is shown when the user clicks the Staff button
 * @param staff an object representing the staff's username and rank
 */
export function createStaffMenu(staff = staffInfo) {
	const rank = staff.Rank;
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
	if(rank == 3) {
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
	if($staffBtn !== null || staffInfo.Rank === 0)
		return;
	$staffBtn = new TopBarButton("Staff", () => {
		$topbar.trigger("menuButtonClick", [$staffMenu, $(document).find($staffMenu).length == 0]);
	});
}

function updateReports(reports: any[]) {
	// append " (#)" to the Reports link, replacing # with the number of reports
	$staffMenu.find("a").each((e, elem) => {
		if(elem.text.search(reportsTextRE) != 0) return;
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