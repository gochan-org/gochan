import { downArrow, upArrow } from "./vars";
import { getCookie } from "./cookies";

export let $topbar = null;
export let topbarHeight = 32;

export class TopBarButton {
	constructor(title, onOpen = () => {}, onClose = () => {}) {
		this.title = title;
		this.onOpen = onOpen;
		this.onClose = onClose;
		$topbar.append(`<a href="javascript:;" class="dropdown-button" id="${title.toLowerCase()}">${title}${downArrow}</a>`);
		let buttonOpen = false;
		let self = this;
		$topbar.find("a#" + title.toLowerCase()).on("click", event => {
			if(!buttonOpen) {
				self.onOpen();
				$(document).bind("click", () => {
					self.onClose();
				});
				buttonOpen = true;
			} else {
				self.onClose();
				buttonOpen = false;
			}
			return false;
		});
	}
}

export function initTopBar() {
	$topbar = $("div#topbar");
	if(!getCookie("pintopbar", {default: true, type: "bool"})) {
		$topbar.css({
			"position": "absolute",
			"top": "0px",
			"padding-left": "0px",
			"padding-right": "0px",
		});
	}

	topbarHeight = $topbar.outerHeight() + 4;
}

export class DropDownMenu {
	constructor(title, menuHTML) {
		this.title = title;
		this.menuHTML = menuHTML;
		let titleLower = title.toLowerCase();
		// console.log($(`a#${titleLower}-menu`).length);

		this.button = new TopBarButton(title, () => {
			$topbar.after(`<div id="${titleLower}-menu" class="dropdown-menu">${menuHTML}</div>`);
			$(`a#${titleLower}-menu`).children(0).text(title + upArrow);
			$(`div#${titleLower}`).css({
				top: $topbar.outerHeight()
			});
		}, () => {
			$(`div#${titleLower}.dropdown-menu`).remove();
			$(`a#${titleLower}-menu`).children(0).html(title + downArrow);
		});
	}
}