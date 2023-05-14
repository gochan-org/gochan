import $ from "jquery";

import { getBooleanStorageVal } from "../storage";

export let $topbar:JQuery<HTMLElement> = null;
export let topbarHeight = 32;

interface BeforeAfter {
	before?: any;
	after?: any;
}
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
	constructor(title: string, action: ()=>any = ()=>{}, beforeAfter:BeforeAfter = {}) {
		this.title = title;
		this.buttonAction = action;
		this.button = $<HTMLLinkElement>("<a/>").prop({
			"href": "javascript:;",
			"class": "dropdown-button",
			"id": title.toLowerCase()
		}).text(title + "▼");

		let $before = $topbar.find(beforeAfter.before);
		let $after = $topbar.find(beforeAfter.after);		
		if($before.length > 0) {
			this.button.insertBefore($before);
		} else if($after.length > 0) {
			this.button.insertAfter($after);
		} else {
			$topbar.append(this.button);
		}

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

$(initTopBar);