/* global webroot */

import $ from 'jquery';

import { alertLightbox } from "./lightbox";
import { $topbar, TopBarButton } from './topbar';

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

export async function initStaff() {
	return $.ajax({
		method: "GET",
		url: `${webroot}manage`,
		data: {
			action: "actions"
		},
		async: true,
		cache: true,
		dataType: "json",
		/** 
		 * @param {StaffAction[]} result
		*/
		success: result => {
			staffActions = result;
		},
		error: () => {
		}
	}).then(getStaffInfo);
}

export async function getStaffInfo() {
	if(loginChecked)
		// don't make multiple unnecessary AJAX requests if we're already logged in
		return staffInfo;
	loginChecked = true;
	return $.ajax({
		method: "GET",
		url: `${webroot}manage`,
		data: {
			action: "staffinfo",
		},
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
	if(checks.length == 0) {
		alertLightbox("No posts selected");
		return false;
	}
	let postID = 0;
	for(let i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") == 0) {
			postID = checks[i].id.replace("check", "");
			break;
		}
	}
	window.location = `${webroot}manage?action=bans&dir=${boardDir}&postid=${postID}`;
}

/**
 * A helper function for creating a menu item
 * @param {StaffInfo} action
 */
function menuItem(action, isCategory = false) {
	return isCategory ? $("<div/>").append($("<b/>").text(action)) : $("<div/>").append(
			$("<a/>").prop({
				href: `${webroot}manage?action=${action.id}`
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
 * @param {number} rank 0 = not logged in, 1 = janitor, 2 = moderator
 * 3 = administrator
 */
export function createStaffMenu(rank = staffInfo.Rank) {
	if(rank == 0) return;
	$staffMenu = $("<div/>").prop({
		id: "staffmenu",
		class: "dropdown-menu"
	});
	let adminActions = staffActions.filter(val => filterAction(val, 3));
	let modActions = staffActions.filter(val => filterAction(val, 2));
	let janitorActions = staffActions.filter(val => filterAction(val, 1));
	
	$staffMenu.append(
		menuItem(getAction("logout")),
		menuItem(getAction("dashboard")));

	$staffMenu.append(menuItem("Janitorial", true));
	for(const action of janitorActions) {
		$staffMenu.append(menuItem(action));
	}
	if(rank < 2) return $staffMenu;

	if(modActions.length > 0)
		$staffMenu.append(menuItem("Moderation", true));
	for(const action of modActions) {
		$staffMenu.append(menuItem(action));
	}
	if(rank < 3) return $staffMenu;

	if(adminActions.length > 0)
		$staffMenu.append(menuItem("Administration", true));
	for(const action of adminActions) {
		$staffMenu.append(menuItem(action));
	}
	if($staffBtn === null) $staffBtn = new TopBarButton("Staff", () => {
		let exists = $(document).find($staffMenu).length > 0;
		if(exists)
			$staffMenu.remove();
		else
			$topbar.after($staffMenu);
	});

	getReports().then(r => {
		// append " (#)" to the Reports link, replacing # with the number of reports
		$staffMenu.find("a").each((e, elem) => {
			if(elem.text.search(reportsTextRE) != 0) return;
			let $span = $("<span/>").text(` (${r.length})`).appendTo(elem);
			if(r.length > 0) {
				// make it bold and red if there are reports
				$span.css({
					"font-weight": "bold",
					"color": "red"
				});
			}
		});
	});
}

function getReports() {
	return $.ajax({
		method: "GET",
		url: `${webroot}manage`,
		data: {
			action: "reports",
			json: "1"
		},
		async: true,
		cache: true,
		dataType: "json"
	}).catch(e => {
		return e;
	});
}