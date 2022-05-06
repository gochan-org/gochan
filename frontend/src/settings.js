import { LightBox, showLightBox } from "./lightbox";
import { TopBarButton } from "./topbar";
import { getCookie, setCookie } from "./cookies";

const validTypes = ["text", "textarea", "password", "number", "checkbox", "select"];
const genericOptions = {
	type: "text",
	dropdownOptions: null, // [{val: "", text: ""}]
	defaultVal: null,
	onSave: () => {},
	customProperties: {},
	customCSS: {}
};

export let $settingsMenu = null;
let $settingsTable = null;
export let settings = [];

function getStorageVal(key, defaultVal) {
	if(localStorage == undefined)
		return getCookie(key, {default: defaultVal});
	let val = localStorage.getItem(key);
	if(val === null && defaultVal !== undefined)
		return defaultVal;
	return val;
}

function setStorageVal(key, val) {
	if(localStorage == undefined)
		setCookie(key, val);
	else
		localStorage.setItem(key, val);
}

function genericDefaultVal(type, options = []) {
	switch(type) {
		case "text":
		case "textarea":
		case "password":
			return "";
		case "number":
			return 0;
		case "checkbox":
			return false;
		case "select":
			if(Array.isArray(options) && options.length > 0)
				return options[0];
			return "";
		default:
			return "";;
	}
}

function fixOptions(options) {
	let fixed = {}

	if(validTypes.indexOf(options.type) > -1)
		fixed.type = options.type;
	else
		fixed.type = validTypes[0];

	fixed.defaultVal = options.defaultVal;
	if(fixed.defaultVal == undefined)
		fixed.defaultVal = genericDefaultVal(fixed, options.dropdownOptions);

	if(options.hasOwnProperty("onSave"))
		fixed.onSave = options.onSave;
	else
		fixed.onSave = () => {};

	if(options.hasOwnProperty("customProperties"))
		fixed.customProperties = options.customProperties;
	else
		fixed.customProperties = genericOptions.customProperties;

	if(options.hasOwnProperty("dropdownOptions"))
		fixed.dropdownOptions = options.dropdownOptions;
	else
		fixed.dropdownOptions = genericOptions.dropdownOptions;

	if(options.hasOwnProperty("customCSS"))
		fixed.customCSS = options.customCSS;
	else
		fixed.customCSS = genericOptions.customCSS;

	return fixed;
}


export class Setting {
	/**
	 * @param {string} key The name of the setting
	 * @param {string} title text that gets shown in the Settings lightbox
	 */
	constructor(key, title, options = genericOptions) {
		this.key = key;
		this.title = title;
		
		let fixedOpts = fixOptions(options);
		this.type = fixedOpts.type;
		this.defaultVal = fixedOpts.defaultVal;
		this.onSave = fixedOpts.onSave;
		this.customProperties = fixedOpts.customProperties;
		this.dropdownOptions = fixedOpts.dropdownOptions;
		this.customCSS = fixedOpts.customCSS;
		this.element = this.createElement();
	}
	saveElementValue() {
		let val = this.element.val();
		if(this.type == "checkbox") {
			val = this.element.prop("checked");
		}
		console.log(this.key);
		console.log(this.element[0]);
		console.log(val);
		setStorageVal(this.key, val);
	}
	setValue(newVal) {
		setStorageVal(this.key, newVal);
	}
	getValue() {
		return getStorageVal(this.key, this.defaultVal);
	}
	createElement() {
		let selector = "<input/>";
		let props = {
			id: this.key,
			name: this.key
		}
		let propKeys = Object.keys(this.customProperties);
		for(const key in propKeys) {
			props[key] = this.customProperties[key];
		}

		switch (this.type) {
			case "text":
				props.type = "text";
				if(this.defaultVal === null)
					this.defaultVal = "";
				break;
			case "textarea":
				selector = "<textarea/>"
				if(this.defaultVal === null)
					this.defaultVal = "";
				break;
			case "number":
				props.type = "number";
				if(this.defaultVal === null)
					this.defaultVal = 0;
				break;
			case "checkbox":
				props.type = "checkbox";
				if(this.defaultVal === null)
					this.defaultVal = false;
				break;
			case "select":
				if(this.dropdownOptions === null)
					break;
				selector = "<select/>";
				break;
			default:
				break;
		}
		let $elem = $(selector);
		if(this.type == "select") {
			for(const option of this.dropdownOptions) {
				$("<option/>").val(option.val).text(option.text).appendTo($elem);
			}
		}
		$elem.prop(props);
		if(Object.keys(this.customCSS).length > 0)
			$elem.css(this.customCSS);

		let val = this.getValue();
		// console.log(this.key, "=>", val, "default:", this.defaultVal);
		if(this.type == "checkbox") {
			console.log(this.key, val, "checked:", val == "true");
			$elem.prop({checked: val == "true" || val == true});
		} else {
			$elem.val(val);
		}
		return $elem;
	}
}

export function initSettings() {
	let settingsHTML =
		`<div id="settings-container" style="overflow:auto"><table width="100%"><colgroup><col span="1" width="50%"><col span="1" width="50%"></colgroup>`;
	
	let styleOptions = [];
	for(const style of styles) {
		styleOptions.push({text: style.Name, val: style.Filename});
	}
	settings.push(
		new Setting("style", "Style", {
			type: "select",
			dropdownOptions: styleOptions,
			defaultVal: defaultStyle
		}),
		new Setting("pintopbar", "Pin top bar", {
			type: "checkbox",
			defaultVal: true
		}),
		new Setting("enableposthover", "Preview post on hover", {
			type: "checkbox",
			defaultVal: false
		}),
		new Setting("enablepostclick", "Preview post on click", {
			type: "checkbox",
			defaultVal: true
		}),
		new Setting("useqr", "Use Quick Reply box", {
			type: "checkbox",
			defaultVal: true
		})
	);

	settingsHTML += `</table></div><div class="lightbox-footer"><hr /><button id="save-settings-button">Save Settings</button></div>`;
	$settingsMenu = new TopBarButton("Settings", () => {
		showLightBox("Settings", settingsHTML);
		if($settingsTable === null) {
			$settingsTable = $("#settings-container table");
			for(const setting of settings) {
				let $tr = $("<tr/>").appendTo($settingsTable);
				$("<td/>").append($("<b/>").text(setting.title)).appendTo($tr);
				$("<td/>").append(setting.element).appendTo($tr);
			}
		}
		
		$("#settings-container").find("input,select,textarea").on("change", function(e) {
			let key = e.currentTarget.id;
			let val = e.currentTarget.value;
			let type = e.currentTarget.attributes.getNamedItem("type")
			console.log(type);
			console.log(key, "=>", val);
		})
		$("button#save-settings-button").on("click", () => {
			for(const setting of settings) {
				setting.saveElementValue();
			}
		});
	});
}