/**
 * @typedef { import("../types/gochan").StaffAction } StaffAction
 * @typedef { import("../types/gochan").StaffInfo } StaffInfo
 */

import $ from 'jquery';

import { alertLightbox } from "../dom/lightbox";
import { $topbar, TopBarButton } from '../dom/topbar';
import "./sections";
import "./filebans";
import { isThreadLocked } from '../api/management';

/**
 * @type {StaffInfo}
 */
const notAStaff = {
	ID: 0,
	Username: "",
	Rank: 0
};

const reportsTextRE = /^Reports( \(\d+\))?/;

/**
 * @type StaffAction[]
 */
export let staffActions = [];
export let staffInfo = notAStaff;
let loginChecked = false;

/**
 * @type {JQuery<HTMLElement>}
 * The menu shown when the Staff button on the top bar is clicked
 */
let $staffMenu = null;
/**
 * @type {TopBarButton}
 * A button that opens $staffMenu
 */
let $staffBtn = null;

/**
 * @param {HTMLElement} dropdown
 * @param {string} item
 */
function dropdownHasItem(dropdown, item) {
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
		let filenameOrig = $post.find("div.file-info a.file-orig").text();
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

export function banFile(banType, filename, checksum, staffNote = "") {
	let xhrFields = {
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
		error: (e) => {
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
	}).then((info) => {
		if(info.error)
			return notAStaff;
		staffInfo = info;
		return info;
	});
}

export async function getPostInfo(id) {
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
	let boardDirArr = location.pathname.split("/");
	if(boardDirArr.length < 2) return;
	let boardDir = boardDirArr[1];
	let checks = $("input[type=checkbox]");
	if(checks.length === 0) {
		alertLightbox("No posts selected");
		return false;
	}
	let postID = 0;
	for(let i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") === 0) {
			postID = checks[i].id.replace("check", "");
			break;
		}
	}
	window.location = `${webroot}manage/bans?dir=${boardDir}&postid=${postID}`;
}

/**
 * A helper function for creating a menu item
 * @param {StaffAction} action
 */
function menuItem(action, isCategory = false) {
	return isCategory ? $("<div/>").append($("<b/>").text(action)) : $("<div/>").append(
			$("<a/>").prop({
				href: `${webroot}manage/${action.id}`
			}).text(action.title)
		);
}

function getAction(id) {
	for(const action of staffActions) {
		if(action.id == id) {
			return action;
		}
	}
}

/**
 * @param {StaffAction} action
 * @param {number} perms
 */
function filterAction(action, perms) {
	return action.title != "Logout"
		&& action.title != "Dashboard"
		&& action.jsonOutput < 2
		&& action.perms == perms;
}

/**
 * Creates a list of staff actions accessible to the user if they are logged in.
 * It is shown when the user clicks the Staff button
 * @param {StaffInfo} staff an object representing the staff's username and rank
 */
export function createStaffMenu(staff = staffInfo) {
	let rank = staff.Rank;
	if(rank === 0) return;
	$staffMenu = $("<div/>").prop({
		id: "staffmenu",
		class: "dropdown-menu"
	});

	$staffMenu.append(
		menuItem(getAction("logout")),
		menuItem(getAction("dashboard")));

	let janitorActions = staffActions.filter(val => filterAction(val, 1));
	$staffMenu.append(menuItem("Janitorial", true));
	for(const action of janitorActions) {
		$staffMenu.append(menuItem(action));
	}

	if(rank >= 2) {
		let modActions = staffActions.filter(val => filterAction(val, 2));
		if(modActions.length > 0)
			$staffMenu.append(menuItem("Moderation", true));
		for(const action of modActions) {
			$staffMenu.append(menuItem(action));
		}
		getReports().then(updateReports);
	}
	if(rank == 3) {
		let adminActions = staffActions.filter(val => filterAction(val, 3));
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

function updateReports(reports) {
	// append " (#)" to the Reports link, replacing # with the number of reports
	$staffMenu.find("a").each((e, elem) => {
		if(elem.text.search(reportsTextRE) != 0) return;
		let $span = $("<span/>").text(` (${reports.length})`).appendTo(elem);
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