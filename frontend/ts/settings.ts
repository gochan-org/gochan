import $ from "jquery";

import { showLightBox } from "./dom/lightbox";
import { initTopBar, TopBarButton } from "./dom/topbar";
import { getBooleanStorageVal, getStorageVal, setStorageVal } from "./storage";
import { initPostPreviews } from "./postutil";
import { closeQR, initQR } from "./dom/qr";
import { initWatcher } from "./watcher/watcher";

let $settingsButton: TopBarButton = null;

const settings: Map<string, Setting<boolean|number|string,HTMLElement>> = new Map();

type ElementValue = string|number|string[];

const noop = () => {
	return;
};

class Setting<T = any, E extends HTMLElement = HTMLElement> {
	key: string;
	title: string;
	defaultVal: T;
	onSave: () => any;
	element: JQuery<E>;
	/**
	 * @param key The name of the setting
	 * @param title text that gets shown in the Settings lightbox
	 * @param defaultVal the setting's default value
	 * @param onSave function that gets called when you save the settings
	 */
	constructor(key: string, title: string, defaultVal:T, onSave = noop) {
		this.key = key;
		this.title = title;
		this.defaultVal = defaultVal;
		this.onSave = onSave;
		this.element = null;
	}
	getElementValue(): T {
		if(this.element === null) return this.defaultVal;
		return this.element.val() as T;
	}
	setElementValue(newVal: T) {
		if(this.element === null) return;
		this.element.val(newVal as ElementValue);
	}
	getStorageValue(): T {
		return getStorageVal(this.key, this.defaultVal as string) as T;
	}
	setStorageValue(newVal: T) {
		setStorageVal(this.key, newVal);
	}
	createElement(selector = "<input/>", props = {}) {
		return $<E>(selector).prop(props).prop({
			id: this.key,
			name: this.key
		});
	}
}

class TextSetting extends Setting<string, HTMLTextAreaElement> {
	constructor(key: string, title: string, defaultVal = "", onSave = noop) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<textarea/>");
		this.element.text(defaultVal);
		const val = this.getStorageValue();
		if(val !== "") {
			this.setElementValue(val);
		}
	}
	setElementValue(text = "") {
		this.element.text(text);
	}
}

class DropdownSetting extends Setting<ElementValue, HTMLSelectElement> {
	constructor(key: string, title: string, options:any[] = [], defaultVal: ElementValue, onSave = noop) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<select/>");
		for(const option of options) {
			$<HTMLSelectElement>("<option/>").val(option.val).text(option.text).appendTo(this.element);
		}
		this.element.val(this.getStorageValue());
	}
}

class BooleanSetting extends Setting<boolean, HTMLInputElement> {
	constructor(key: string, title: string, defaultVal = false, onSave = noop) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<input/>", {
			type: "checkbox",
			checked: this.getStorageValue()
		});
	}
	getElementValue() {
		return this.element.prop("checked") as boolean;
	}
	setElementValue(newVal: boolean) {
		this.element.prop("checked", newVal);
	}
}

interface MinMax {
	type?: string;
	min?: number;
	max?: number;
}
class NumberSetting extends Setting<number, HTMLInputElement> {
	constructor(key: string, title: string, defaultVal = 0, minMax: MinMax = {min: null, max: null}, onSave = noop) {
		super(key, title, defaultVal, onSave);
		const props: MinMax = {
			type: "number"
		};
		if(typeof minMax.min === "number" && !isNaN(minMax.min))
			props.min = minMax.min;
		if(typeof minMax.max === "number" && !isNaN(minMax.max))
			props.max = minMax.max;
		this.element = this.createElement("<input />", props).val(this.getStorageValue());
	}
	getStorageValue() {
		let val = Number.parseFloat(super.getStorageValue() as unknown as string);
		if(isNaN(val))
			val = this.defaultVal;
		return val;
	}
}

function createLightbox() {
	const settingsHTML =
		'<div id="settings-container" style="overflow:auto"><table width="100%"><colgroup><col span="1" width="50%"><col span="1" width="50%"></colgroup></table></div><div class="lightbox-footer"><hr /><button id="save-settings-button">Save Settings</button></div>';
	showLightBox("Settings", settingsHTML);
	$("button#save-settings-button").on("click", () => {
		settings.forEach((setting, key) => {
			setStorageVal(key, setting.getElementValue());
			setting.onSave();
		});
	});
	const $settingsTable = $("#settings-container table");
	settings.forEach((setting) => {
		const $tr = $("<tr/>").appendTo($settingsTable);
		$("<td/>").append($("<b/>").text(setting.title)).appendTo($tr);
		$("<td/>").append(setting.element).appendTo($tr);
	});
}

/**
 * executes the custom JavaScript set in the settings
 */
export function setCustomJS() {
	const customJS = getStorageVal("customjs");
	if(customJS !== "") {
		eval(customJS);
	}
}

/**
 * applies the custom CSS set in the settings
 */
export function setCustomCSS() {
	const customCSS = getStorageVal("customcss");
	if(customCSS !== "") {
		$("style#customCSS").remove();
		$("<style/>").prop({
			id: "customCSS"
		}).html(customCSS).appendTo(document.head);
	}
}

$(() => {
	const styleOptions = [{text: "Default", val: ""}];
	for(const style of styles) {
		styleOptions.push({text: style.Name, val: style.Filename});
	}
	settings.set("style", new DropdownSetting("style", "Style", styleOptions, "", function() {
		const val:string = this.getElementValue();
		const themeElem = document.getElementById("theme");
		if(!themeElem) return;
		if(val === "" && themeElem.hasAttribute("default-href")) {
			themeElem.setAttribute("href", themeElem.getAttribute("default-href"));
		} else if(val !== "") {
			themeElem.setAttribute("href", `${webroot}css/${val}`);
		}
	}) as Setting);
	settings.set("pintopbar", new BooleanSetting("pintopbar", "Pin top bar", true, initTopBar));
	settings.set("enableposthover", new BooleanSetting("enableposthover", "Preview post on hover", true, initPostPreviews));
	settings.set("enablepostclick", new BooleanSetting("enablepostclick", "Preview post on click", true, initPostPreviews));
	settings.set("useqr", new BooleanSetting("useqr", "Use Quick Reply box", true, () => {
		if(getBooleanStorageVal("useqr", true)) initQR();
		else closeQR();
	}));
	settings.set("watcherseconds", new NumberSetting("watcherseconds", "Check watched threads every # seconds", 15, {
		min: 2
	}, initWatcher));
	settings.set("persistentqr", new BooleanSetting("persistentqr", "Persistent Quick Reply", false));

	settings.set("customjs", new TextSetting("customjs", "Custom JavaScript (ran on page load)", ""));
	settings.set("customcss", new TextSetting("customcss", "Custom CSS", "", setCustomCSS));

	if($settingsButton === null)
		$settingsButton = new TopBarButton("Settings", createLightbox, {
			before: "a#watcher"
		});
});