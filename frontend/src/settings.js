import { showLightBox } from "./lightbox";
import { initTopBar, TopBarButton } from "./topbar";
import { getStorageVal, setStorageVal } from "./storage";

/**
 * @type {TopBarButton}
 */
let $settingsButton = null;
/**
 * @type {Map<string,Setting>}
 */
const settings = new Map();


class Setting {
	/**
	 * @param {string} key The name of the setting
	 * @param {string} title text that gets shown in the Settings lightbox
	 * @param {string} defaultVal the setting's default value
	 * @param {string} onSave function that gets called when you save the settings
	 */
	constructor(key, title, defaultVal = "", onSave = () => {}) {
		this.key = key;
		this.title = title;
		this.defaultVal = defaultVal;
		this.onSave = onSave;
		this.element = null;
	}
	getElementValue() {
		if(this.element == null) return "";
		return this.element.val();
	}
	setElementValue(newVal) {
		if(this.element == null) return;
		this.element.val(newVal);
	}
	getStorageValue() {
		return getStorageVal(this.key, this.defaultVal);
	}
	setStorageValue(newVal) {
		setStorageVal(this.key, newVal);
	}
	createElement(selector = "<input/>", props = {}) {
		return $(selector).prop(props).prop({
			id: this.key,
			name: this.key
		});
	}
}

class DropdownSetting extends Setting {
	constructor(key, title, options = [], defaultVal = "", onSave = () => {}) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<select/>");
		for(const option of options) {
			$("<option/>").val(option.val).text(option.text).appendTo(this.element);
		}
		this.element.val(this.getStorageValue());
	}
}

class BooleanSetting extends Setting {
	constructor(key, title, defaultVal = false, onSave = () => {}) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<input/>", {
			type: "checkbox",
			checked: this.getStorageValue()
		});
	}
	getElementValue() {
		return this.element.prop("checked");
	}
	setElementValue(newVal) {
		this.element.prop({checked: newVal})
	}
	getStorageValue() {
		let val = super.getStorageValue();
		return val == true || val == "true";
	}
}

class NumberSetting extends Setting {
	constructor(key, title, defaultVal = 0, onSave = () => {}) {
		super(key, title, defaultVal, onSave);
		this.element = this.createElement("<input />", {
			type: "number"
		}).val(this.getStorageValue());
	}
	getStorageValue() {
		let val = Number.parseFloat(super.getStorageValue());
		if(val == NaN)
			val = this.defaultVal;
		return val;
	}
}

function createLightbox() {
	let settingsHTML =
		`<div id="settings-container" style="overflow:auto"><table width="100%"><colgroup><col span="1" width="50%"><col span="1" width="50%"></colgroup></table></div><div class="lightbox-footer"><hr /><button id="save-settings-button">Save Settings</button></div>`;
	showLightBox("Settings", settingsHTML);
	$("button#save-settings-button").on("click", () => {
		settings.forEach((setting, key, map) => {
			setStorageVal(key, setting.getElementValue());
			setting.onSave();
		});
	});
	let $settingsTable = $("#settings-container table");
	settings.forEach((setting, key, map) => {
		let $tr = $("<tr/>").appendTo($settingsTable);
		$("<td/>").append($("<b/>").text(setting.title)).appendTo($tr);
		$("<td/>").append(setting.element).appendTo($tr);
	});
}

export function initSettings() {
	let styleOptions = [];
	for(const style of styles) {
		styleOptions.push({text: style.Name, val: style.Filename});
	}
	settings.set("style", new DropdownSetting("style", "Style", styleOptions, defaultStyle, function() {
		document.getElementById("theme").setAttribute("href",
			`${webroot}css/${this.getElementValue()}`);
	}));
	settings.set("pintopbar", new BooleanSetting("pintopbar", "Pin top bar", true, initTopBar));
	settings.set("enableposthover", new BooleanSetting("enableposthover", "Preview post on hover"));
	settings.set("enablepostclick", new BooleanSetting("enablepostclick", "Preview post on click", true));
	settings.set("useqr", new BooleanSetting("useqr", "Use Quick Reply box", true));

	if($settingsButton == null)
		$settingsButton = new TopBarButton("Settings", createLightbox);
}