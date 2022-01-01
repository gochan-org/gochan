import $ from 'jquery';
import { showLightBox } from "./lightbox";
import { $topbar, TopBarButton } from './topbar';

/**
 * @type {StaffInfo}
 */
const notAStaff = {
	ID: 0,
	Username: "",
	Rank: 0
};

/**
 * @type {StaffActionMap}
 */
export let staffActions = {};
export let staffInfo = notAStaff;

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
		success: result => {
			staffActions = result;
		},
		error: (xhr, status, err) => {
		}
	}).then(getStaffInfo);
}

export async function getStaffInfo() {
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
		return info.ID > 0
	});
}

export function banSelectedPost() {
	let boardDirArr = location.pathname.split("/");
	if(boardDirArr.length < 2) return;
	let boardDir = boardDirArr[1];
	let checks = $("input[type=checkbox]");
	if(checks.length == 0) {
		alert("No posts selected");
		return false;
	}
	let postID = 0;
	for(let i = 0; i < checks.length; i++) {
		if(checks[i].id.indexOf("check") == 0) {
			postID = checks[i].id.replace("check", "");
			break;
		}
	}
	window.location = `${webroot}manage?action=bans&dir=${boardDir}&postid=${postID}`
}

/**
 * A helper function for creating a menu item
 * @param {StaffAction} action
 */
function menuItem(action, id = "") {
	if(id == "") {
		return $("<div/>").append($("<b/>").text(action));
	} else {
		return $("<div/>").append(
			$("<a/>").prop({
				href: `${webroot}manage?action=${id}`
			}).text(action.title)
		);
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
	let actions = Object.keys(staffActions);
	let adminActions = actions.filter(val => filterAction(staffActions[val], 3));
	let modActions = actions.filter(val => filterAction(staffActions[val], 2));
	let janitorActions = actions.filter(val => filterAction(staffActions[val], 1));

	$staffMenu.append(menuItem(staffActions["logout"], "logout"), menuItem(staffActions["dashboard"], "dashboard"))
	$staffMenu.append(menuItem("Janitorial"));
	for(const actionID of janitorActions) {
		$staffMenu.append(menuItem(staffActions[actionID], actionID));
	}
	if(rank < 2) return $staffMenu;

	$staffMenu.append(menuItem("Moderation"));
	for(const actionID of modActions) {
		$staffMenu.append(menuItem(staffActions[actionID], actionID));
	}
	if(rank < 3) return $staffMenu;

	$staffMenu.append(menuItem("Administration"));
	for(const actionID of adminActions) {
		$staffMenu.append(menuItem(staffActions[actionID], actionID));
	}
	$staffBtn = new TopBarButton("Staff", () => {
		let exists = $(document).find($staffMenu).length > 0;
		if(exists)
			$staffMenu.remove();
		else
			$topbar.after($staffMenu);
	});
}

/**
 * Opens a lightbox for using staff tools without having to go load the page
 * @param {string} actionURL The URL to get the action HTML from
 * @deprecated
 */
export function openStaffLightBox(actionURL) {
	$.ajax({
		method: 'GET',
		url: `${webroot}manage`,
		data: {
			action: actionURL,
		},
		dataType:"html",
		async:false,

		success: result => {
			let body = `<div id="body-mock">${result.replace(/^[\s\S]*<body.*?>|<\/body>[\s\S]*$/ig, "")}</div>`;
			let $body = $(body);
			let header = $body.find("h1");
			let headerText = header.text();
			header.remove();
			if(headerText == "") headerText = "Manage";
			showLightBox(headerText,$body.html());
		},
		error: result => {
			let responseText = result.responseText;
			header = responseText.substring(responseText.indexOf("<h1>")+4,responseText.indexOf("</h1>"));

			responseText = responseText.substring(responseText.indexOf("</h1>") + 5, responseText.indexOf("</body>"));
			if(header == "") {
				showLightBox("Manage", responseText);
			} else {
				showLightBox(header, responseText);
			}
		}
	});
}