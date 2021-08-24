import { LightBox, showLightBox } from "./lightbox";
import { TopBarButton } from "./topbar";
import { getCookie, setCookie } from "./cookies";
export let $settingsMenu = null;
export let settings = [];
let settingsLB = null;

export function getSetting(id) {
	for(let s = 0; s < settings.length; s++) {
		if(settings[s].id == id) return settings[s];
	}
	return {};
}

class Setting {
	constructor(id, text, type, defaultVal, cb, options) {
		this.id = id;
		this.text = text;
		this.type = type; // text, textarea, checkbox, select
		this.defaultVal = defaultVal;
		this.options = [];
		if(getCookie(this.id) == undefined) {
			this.setCookie(this.defaultVal, 7);
		}
		if(this.type == "select") this.options = options;
		this.cb = () => {};
		if(cb) this.cb = cb;
	}

	save(newVal, expires) {
		setCookie(this.id, newVal, expires);
		this.cb();
	}

	getCookie(type = "string", defaultVal) {
		let val = getCookie(this.id, {type: type, default: defaultVal});

		if(this.type == "checkbox") val = (val == "true");
		return val;
	}
	
	setCookie(val,expires) {
		setCookie(this.id, val,expires);
	}

	getVal() {
		let elem = document.getElementById(this.id);
		if(elem != null) {
			if(elem.type == "checkbox") return elem.checked;
			return elem.value;
		}
	}

	html() {
		let html = "";
		switch(this.type) {
			case "checkbox":
				if(this.getCookie() == true)
					html = `<input id="${this.id}" type="checkbox" checked="checked" />`;
				else
					html = `<input id="${this.id}" type="checkbox" />`;
				break;
			case "select":
				html = `<select id="${this.id}" name="${this.id}" style="min-width:50%">`;
				for(const option of this.options) {
					html += `<option value="${option.val}"`;
					if(this.getCookie() == option.val) html += `selected="${this.getCookie()}"`;
					html += `>${option.text}</option>`;
				}
				html += "</select>";
				break;
			case "textarea":
				html = `<textarea id="${this.id}" name="${this.id}">${this.getCookie()}</textarea>`;
				break;
			default:
				html = `<input id="${this.id}" type="checkbox" val="${this.getCookie()}" />`;
				break;
		}
		return html;
	}
}

export function initSettings() {
	let settingsHTML =
		`<div id="settings-container" style="overflow:auto"><table width="100%"><colgroup><col span="1" width="50%"><col span="1" width="50%"></colgroup>`;

	settings.push(
		new Setting("style", "Style", "select", defaultStyle, function() {
			document.getElementById("theme").setAttribute("href",
				`${webroot}css/${this.getCookie(defaultStyle)}`
			)
		}, []),
		new Setting("pintopbar", "Pin top bar", "checkbox", true),
		new Setting("enableposthover", "Preview post on hover", "checkbox", true),
		new Setting("enablepostclick", "Preview post on click", "checkbox", true),
		new Setting("useqr", "Use Quick Reply box", "checkbox", true)
	);

	for(const style of styles) {
		settings[0].options.push({text: style.Name, val: style.Filename});
	}
	for(const setting of settings) {
		settingsHTML += `<tr><td><b>${setting.text};</b></td><td>${setting.html()}</td></tr>`
	}
	settingsHTML += "</table></div><div class=\"lightbox-footer\"><hr /><button id=\"save-settings-button\">Save Settings</button></div>";

	$settingsMenu = new TopBarButton("Settings", () => {
		showLightBox("Settings", settingsHTML);
		$("button#save-settings-button").on("click", () => {
			for(const setting of settings) {
				setting.save(setting.getVal());
			}
		});
	});
}