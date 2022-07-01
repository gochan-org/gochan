import $ from "jquery";

import { getBooleanStorageVal } from "./storage";

/**
 * @type {JQuery<HTMLElement>}
 */
export let $topbar = null;
export let topbarHeight = 32;

/**
 * TopBarButton A button to be added to the right side of the top bar
 */
export class TopBarButton {
	/**
	 * @param {string} title The text shown on the button
	 * @param {()=>any} action The function executed when the button is clicked
	 */
	constructor(title, action = () => {}) {
		this.title = title;
		this.buttonAction = action;
		this.button = $("<a/>").prop({
			"href": "javascript:;",
			"class": "dropdown-button",
			"id": title.toLowerCase()
		}).text(title + "▼");
		$topbar.append(this.button);
		this.button.on("click", e => {
			e.preventDefault();
			this.buttonAction();
			return false;
		});
	}
}

/**
 * Initialize the bar at the top of the page with board links and buttons
 */
export function initTopBar() {
	$topbar = $("div#topbar");
	if(getBooleanStorageVal("pintopbar", true)) {
		$topbar.css({
			"z-index": "9001",
			"position": "fixed"
		});
	} else {
		$topbar.css({
			"position": "absolute",
			"top": "0px"
		});
	}
	topbarHeight = $topbar.outerHeight() + 4;
}
