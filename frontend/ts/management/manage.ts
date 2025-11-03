import $ from "jquery";

import { alertLightbox } from "../dom/lightbox";
import { $topbar, TopBarButton, menuItem } from "../dom/topbar";
import "./sections";
import "./viewlog";
import { isThreadLocked } from "../api/management";
import { getNumberStorageVal, setStorageVal } from "../storage";

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

let staffNotificationsInterval: number = null;
let latestReportID: number = getNumberStorageVal("latestreport", -1);
let latestAppealID: number = getNumberStorageVal("latestappeal", -1);
$(document).on("gotStaffRank", (_e, rank:number) => {
	if(rank >= 2 && staffNotificationsInterval === null) {
		const intervalSeconds = getNumberStorageVal("reportinterval", 30);
		staffNotificationsInterval = setInterval(updateStaffNotifications, intervalSeconds * 1000) as any as number;
	}
	if(rank > 0) {
		$(".post select.post-actions").each(addManageEvents);
		setupManagementEvents();
	}
});


function dropdownHasItem(dropdown: any, item: string) {
	return [...dropdown.children].filter(v => v.text === item).length > 0;
}

function addManageEvents(_i: number, el: HTMLSelectElement) {
	if(staffInfo === null || staffInfo.rank < 2) return;
	const $el = $(el);
	const $post = $(el.parentElement);
	const isLocked = isThreadLocked($post);

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
	if(!dropdownHasItem(el, "Filter similar posts")) {
		$el.append("<option>Filter similar posts</option>");
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

export async function initStaff() {
	if(staffInfo !== null || staffActions?.length > 0)
		// don't make multiple unnecessary AJAX requests
		return staffInfo;

	return await fetch(`${webroot}manage/staffinfo`, {
		method: "GET",
		cache: "no-cache",
		credentials: "same-origin"
	}).then(response => {
		if(!response.ok) throw new Error(`Network response was not ok (${response.status})`);
		return response.json();
	}).then((result:StaffInfo) => {
		staffInfo = result;
		updateLatestReportAppeal(staffInfo);
		staffActions = staffInfo?.actions ?? [];
		$(document).trigger("gotStaffRank", staffInfo.rank);
		return staffInfo;
	}).catch((ee) => {
		throw new Error(`Error getting staff info: ${ee.statusText}`);
	});
}

export async function getPostInfo(id: number):Promise<PostInfo> {
	return await fetch(`${webroot}manage/postinfo?postid=${id}`, {
		method: "GET",
		cache: "no-cache",
		credentials: "same-origin"
	}).then(response => {
		if(!response.ok) {
			return Promise.reject(`Error fetching post info: ${response.status} ${response.statusText}`);
		}
		return response.json() as Promise<PostInfo>;
	});
}

export async function isLoggedIn() {
	return await initStaff().then(info => {
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

	const logoutAction = getAction("logout");
	const dashboardAction = getAction("dashboard");
	$staffMenu.append(
		menuItem(logoutAction.title, `${webroot}manage/${logoutAction.id}`),
		menuItem(dashboardAction.title, `${webroot}manage/${dashboardAction.id}`),
	);

	$staffMenu.append(menuItem("Janitorial"));
	staffActions.filter(val => filterAction(val, 1)).map(action => {
		$staffMenu.append(menuItem(action.title, `${webroot}manage/${action.id}`));
	});

	if(rank >= 2) {
		const modActions = staffActions.filter(val => filterAction(val, 2));
		if(modActions.length > 0)
			$staffMenu.append(menuItem("Moderation"));
		for(const action of modActions) {
			const item = menuItem(action.title, `${webroot}manage/${action.id}`);
			if(action.id === "reports" && staffInfo.reports?.length > 0 ||
				action.id === "appeals" && staffInfo.appeals?.length > 0) {
				item
					.find("a").text(`${action.title} (${staffInfo.reports.length} open)`)
					.addClass("text-bold")
					.css("color", "red");
			}
			$staffMenu.append(item);
		}
	}
	if(rank >= 3) {
		const adminActions = staffActions.filter(val => filterAction(val, 3));
		if(adminActions.length > 0)
			$staffMenu.append(menuItem("Administration"));
		for(const action of adminActions) {
			$staffMenu.append(menuItem(action.title, `${webroot}manage/${action.id}`));
		}
	}
	createStaffButton();
}

export function addStaffThreadOptions() {
	const $threadOptionsRow = $("tr#threadoptions");
	if($threadOptionsRow.length < 1) return;
	$threadOptionsRow.show().find("td").append(
		$("<label/>").append(
			$("<input/>").attr({
				type: "checkbox",
				name: "modstickied"
			}), " Sticky thread"
		),
		$("<label/>").append(
			$("<input/>").attr({
				type: "checkbox",
				name: "modlocked"
			}), " Locked thread"
		),
	);

}

function createStaffButton() {
	if($staffBtn !== null || staffInfo === null || staffInfo.rank === 0)
		return;
	if($topbar.find(".topbar-staff").length === 0) {
		$(`<div class="topbar-staff"></div>`).insertBefore($topbar.find(".topbar-watcher"));
	}
	$staffBtn = new TopBarButton("Staff", () => {
		$topbar.trigger("menuButtonClick", [$staffMenu, $(document).find($staffMenu).length === 0]);
	}, ".topbar-staff");
}

function updateLatestReportAppeal(info: StaffInfo) {
	if(info.rank >= 2) {
		createStaffButton();
		$staffBtn.button.empty();
		if(info.reports || info.appeals) {
			const elements = ["Staff ("];
			if(info.reports) {
				elements.push(`<span class="topbar-reports" title="Reports">R:${info.reports?.length ?? 0}</span>`);
			}
			if(info.appeals) {
				if(info.reports) {
					elements.push(", ");
				}
				elements.push(`<span class="topbar-appeals" title="Appeals">A:${info.appeals?.length ?? 0}</span>`);
			}
			elements.push(") ▼");
			$staffBtn.button.append(...elements);
			staffInfo.reports = info.reports;
			staffInfo.appeals = info.appeals;
		} else {
			$staffBtn.button.text("Staff ▼");
		}
	}

	if(info.reports?.length > 0) {
		const latestReport = info.reports?.reduce((prev:PostReport, current:PostReport) => ((prev?.id ?? -1) > current.id) ? prev : current, null);
		if(latestReport && latestReport.id > latestReportID) {
			latestReportID = latestReport.id;
			setStorageVal("latestreport", latestReportID);
			Notification.requestPermission().then(permission => (permission === "granted")?
				new Notification("New report", {
					body: `New report for post ${latestReport.post_link} from ${latestReport.reporter_ip}\nReason: ${latestReport.reason}`,
				}):null
			);
		}
	}
	if(info.appeals?.length > 0) {
		const latestAppeal = info.appeals?.reduce((prev:Appeal, current:Appeal) => ((prev?.id ?? -1) > current.id) ? prev : current, null);
		if(latestAppeal && latestAppeal.id > latestAppealID) {
			latestAppealID = latestAppeal.id;
			setStorageVal("latestappeal", latestAppealID);
			Notification.requestPermission().then(permission => (permission === "granted")?
				new Notification(`New appeal for ban ${latestAppeal.ban_id}`, {
					body: latestAppeal.appeal_text,
				}):null
			);
		}
	}
}

async function updateStaffNotifications() {
	await fetch(`${webroot}manage/staffinfo?noactions=1`, {
		method: "GET",
		cache: "no-cache",
		credentials: "same-origin"
	}).then<StaffInfo>(response => {
		if(!response.ok) throw new Error(`Network response was not ok (${response.status})`);
		return response.json();
	}).then(info => updateLatestReportAppeal(info))
		.catch(err => console.log("Error updating staff notifications:", err));
}
