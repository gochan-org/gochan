import $ from "jquery";
import path from "path-browserify";

import { showLightBox } from "./dom/lightbox";
import { initTopBar, TopBarButton } from "./dom/topbar";
import { getBooleanStorageVal, getStorageVal, setStorageVal } from "./storage";
import { initPostPreviews } from "./postutil";
import { closeQR, initQR } from "./dom/qr";
import { initWatcher } from "./watcher/watcher";
import { updateBrowseButton } from "./dom/uploaddata";

let $settingsButton: TopBarButton = null;

const settings: Map<string, Setting<boolean|number|string,HTMLElement>> = new Map();

type ElementValue = string|number|string[];

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
	constructor(key: string, title: string, defaultVal:T, onSave?:()=>any) {
		this.key = key;
		this.title = title;
		this.defaultVal = defaultVal;
		this.onSave = onSave ?? (()=>true);
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
		return getStorageVal(this.key, this.defaultVal as any) as T;
	}
	setStorageValue(newVal: T) {
		setStorageVal(this.key, newVal);
		this.onSave();
	}
	createElement(selector = "<input/>", props = {}) {
		return $<E>(selector).prop(props).prop({
			id: this.key,
			name: this.key
		});
	}
}

class TextSetting extends Setting<string, HTMLTextAreaElement> {
	constructor(key: string, title: string, defaultVal = "", onSave?:()=>any) {
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
	constructor(key: string, title: string, options:any[] = [], defaultVal: ElementValue, onSave?:()=>any) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<select/>");
		for(const option of options) {
			$<HTMLSelectElement>("<option/>").val(option.val).text(option.text).appendTo(this.element);
		}
		this.element.val(this.getStorageValue());
	}
}

class BooleanSetting extends Setting<boolean, HTMLInputElement> {
	constructor(key: string, title: string, defaultVal = false, onSave?:()=>any) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<input/>", {
			type: "checkbox",
			checked: this.getStorageValue()
		});
	}
	getStorageValue(): boolean {
		return super.getStorageValue() as any === "true";
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
	constructor(key: string, title: string, defaultVal = 0, minMax: MinMax = {min: null, max: null}, onSave?:()=>any) {
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
		let val = Number.parseFloat(super.getStorageValue() as any);
		if(isNaN(val))
			val = this.defaultVal;
		return val;
	}
}

function updateSettingsTextArea(_i: number, el:HTMLTextAreaElement) {
	switch(el.id) {
	case "customjs":
		el.placeholder = "// JavaScript entered here will run when the page loads";
		break;
	case "customcss":
		el.placeholder = "body {\n  background: darkblue;\n}";
		break;
	default:
		break;
	}
	$("<button/>").attr({
		id: `${el.id}-apply`
	}).css("display", "block").on("click", () => {
		setStorageVal(el.id, el.value);
		switch(el.id) {
		case "customcss":
			setCustomCSS();
			break;
		case "customjs":
			setCustomJS();
			break;
		default:
			break;
		}
	}).text("Apply").insertAfter(el);
}

function createLightbox() {
	const settingsHTML =
		'<div id="settings-container" style="overflow:auto"><table width="100%"><colgroup><col span="1" width="50%"><col span="1" width="50%"></colgroup></table></div>';
	showLightBox("Settings", settingsHTML);

	const $settingsTable = $("#settings-container table");
	settings.forEach((setting) => {
		const $tr = $("<tr/>").appendTo($settingsTable);
		const val = getStorageVal(setting.key, setting.defaultVal as any) as string|boolean|number;
		if(val === true)
			setting.element.prop("checked", true);
		else
			setting.element.val(val as string|number);
		$("<td/>").append($("<b/>").text(setting.title)).appendTo($tr);
		$("<td/>").append(setting.element).appendTo($tr);
	});

	$settingsTable.find<HTMLInputElement>("input,select").on("change", (ev: JQuery.ChangeEvent) => {
		const $el: JQuery<HTMLInputElement> = $(ev.target);
		const elType = $el.attr("type");
		const val: string|boolean = (elType === "checkbox")?$el.prop("checked"):$el.val();
		setStorageVal($el.attr("id"), val);
		settings.get($el.attr("id"))?.onSave();


		if(ev.target.id === "style") {
			setTheme();
		}
	});

	$settingsTable
		.find<HTMLTextAreaElement>("textarea")
		.each(updateSettingsTextArea);
}

/**
 * applies the theme set by the user, or the default if none is set
 */
export function setTheme() {
	const style = getStorageVal("style", "");
	const themeElem = document.getElementById("theme");

	if(themeElem) {
		if(!themeElem.hasAttribute("default-href"))
			themeElem.setAttribute("default-href", themeElem.getAttribute("href"));
		if(style === "")
			themeElem.setAttribute("href", themeElem.getAttribute("default-href"));
		else
			themeElem.setAttribute("href", path.join(webroot ?? "/", "css", style));
	}
	setLineHeight();
}

function setLineHeight() {
	if(getBooleanStorageVal("increaselineheight", false)) {
		document.body.classList.add("increase-line-height");
	} else {
		document.body.classList.remove("increase-line-height");
	}
}

/**
 * executes the custom JavaScript set in the settings
 */
export function setCustomJS() {
	const customJS = getStorageVal("customjs", "");
	$("script.customjs").remove();
	if(customJS === "") return;
	$("<script/>")
		.addClass("customjs")
		.text(customJS)
		.appendTo(document.head);
}

/**
 * applies the custom CSS set in the settings
 */
export function setCustomCSS() {
	const customCSS = getStorageVal("customcss", "");
	$("style.customcss").remove();
	if(customCSS === "") return;
	$("<style/>")
		.addClass("customcss")
		.text(customCSS)
		.appendTo(document.head);
}

$(() => {
	const styleOptions = [];
	for(const style of styles) {
		styleOptions.push({text: style.Name, val: style.Filename});
	}
	settings.set("style", new DropdownSetting("style", "Style", styleOptions, defaultStyle, function() {
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
	settings.set("increaselineheight", new BooleanSetting("increaselineheight", "Increase line height", false, setLineHeight));
	settings.set("enableposthover", new BooleanSetting("enableposthover", "Preview post on hover", true, initPostPreviews));
	settings.set("enablepostclick", new BooleanSetting("enablepostclick", "Preview post on click", true, initPostPreviews));
	settings.set("useqr", new BooleanSetting("useqr", "Use Quick Reply box", true, () => {
		if(getBooleanStorageVal("useqr", true)) initQR();
		else closeQR();
	}));
	settings.set("persistentqr", new BooleanSetting("persistentqr", "Persistent Quick Reply", false));
	settings.set("watcherseconds", new NumberSetting("watcherseconds", "Check watched threads every # seconds", 15, {
		min: 2
	}, initWatcher));
	settings.set("newuploader", new BooleanSetting("newuploader", "Use new upload element", true, updateBrowseButton));
	settings.set("smoothhidetoggle", new BooleanSetting("smoothhidetoggle", "Smooth hide block toggle", true));

	settings.set("customjs", new TextSetting("customjs", "Custom JavaScript", ""));
	settings.set("customcss", new TextSetting("customcss", "Custom CSS", "", setCustomCSS));

	if($settingsButton === null)
		$settingsButton = new TopBarButton("Settings", createLightbox, {before: "a#watcher"});
});