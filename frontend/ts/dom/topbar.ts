import $ from "jquery";

import { getBooleanStorageVal } from "../storage";

export const $topbar = $("div#topbar");
export let topbarHeight = $topbar.height() + 4;

/**
 * TopBarButton A button to be added to the right side of the top bar
 */
export class TopBarButton {
	title: string;
	buttonAction: ()=>any;
	button: JQuery<HTMLLinkElement>;
	/**
	 * @param title The text shown on the button
	 * @param action The function executed when the button is clicked
	 */
	constructor(title: string, action: ()=>any = ()=>{}, container: string = ".topbar-right") {
		this.title = title;
		this.buttonAction = action;
		this.button = $<HTMLLinkElement>("<a/>").prop({
			"href": "javascript:;",
			"class": "dropdown-button",
			"id": title.toLowerCase()
		}).text(title + "▼");
		if(container && $(container).length > 0) {
			$(container).append(this.button);
		}

		this.button.on("click", e => {
			e.preventDefault();
			this.buttonAction();
			return false;
		});
	}
}

/**
 * A helper function for creating a menu item
 */
export function menuItem(text:string, href?:string) {
	const isCategory = href === undefined;
	return isCategory ? $("<div/>").append($("<b/>").text(text)) : $("<div/>").append(
		$("<a/>").prop({
			href: href
		}).text(text)
	);
}

/**
 * Initialize the bar at the top of the page with board links and buttons
 */
export function initTopBar() {
	$topbar.find(".topbar-right").append(
		`<div class="topbar-watcher"></div>`,
		`<div class="topbar-settings"></div>`
	);

	const $responsiveBoardsMenu = $(`<div id="boards-menu" class="dropdown-menu"><nav><ul></ul></nav></div>`);
	$responsiveBoardsMenu.find("ul").append(
		`<li><a href="${webroot}">home</a></li>`,
		`<li><b>Boards</b></li>`
	);
	const $boardSections = $topbar.find("div.topbar-boards > div.topbar-section");
	for(const section of $boardSections) {
		const $boards = $(section).find<HTMLAnchorElement>("a");
		for(const board of $boards) {
			$responsiveBoardsMenu.append(
				`<li><a href="${board.href}">${board.innerText}</a> &mdash; ${board.title}</li>`
			);
		}
	}
	const responsiveBoardsBtn = new TopBarButton("Links", () => {
		$topbar.trigger("menuButtonClick", [$responsiveBoardsMenu, $(document).find($responsiveBoardsMenu).length === 0]);
	}, null);
	responsiveBoardsBtn.button.addClass("boards-button").insertBefore($topbar.find("div.topbar-boards"));

	if(getBooleanStorageVal("pintopbar", true)) {
		$topbar.css({
			"z-index": "10",
			"position": "fixed"
		});
	} else {
		$topbar.css({
			"position": "absolute",
			"top": "0px"
		});
	}
	topbarHeight = $topbar.outerHeight() + 4;
	$topbar.on("menuButtonClick", (e, $menu, open) => {
		$("div.dropdown-menu").remove();
		if(open) {
			$topbar.after($menu);
		} else {
			$menu.remove();
		}
	});
}

$(() => {
	initTopBar();
	$("body").on("click", () => $(".dropdown-menu").remove());
});