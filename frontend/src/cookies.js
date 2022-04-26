import $ from "jquery";

/**
 * @param {string} name
 */
export function getCookie(name, options = {type: "string"}) {
	let val = options.default;
	let cookieArr = document.cookie.split("; ");

	for(const cookie of cookieArr) {
		let pair = cookie.split("=");
		if(pair[0] != name) continue;
		try {
			val = decodeURIComponent(pair[1]);
		} catch(err) {
			return options.default;
		}
	}
	switch(options.type) {
		case "int":
			return parseInt(val);
		case "float":
			return parseFloat(val);
		case "bool":
		case "boolean":
			return val == "true" || val == "1";
		case "json":
			try {
				return JSON.parse(val);
			} catch(e) {
				return {};
			}
	}
	if(val == undefined)
		val = "";
	return val;
}

/**
 * gets cookies ready to be used elsewhere
 */
export function initCookies() {
	$("input[name=postname]").val(getCookie("name"));
	$("input[name=postemail]").val(getCookie("email"));
	$("input[name=postpassword]").val(getCookie("password"));
	$("input[name=delete-password]").val(getCookie("password"));
}

/**
 * Set a cookie
 * @param {string} name
 * @param {string} value
 * @param {string} expires
 */
export function setCookie(name, value, expires, root) {
	if(root === undefined || root === "")
		root = webroot;
	let expiresStr = "";
	if(expires !== undefined && expires !== "") {
		expiresStr = ";expires="
		let d = new Date();
		d.setTime(d.getTime() + 1000*60*60*24*expires)
		expiresStr += d.toUTCString();
	}
	document.cookie = `${name}=${escape(value)}${expiresStr};path=${root};sameSite=strict`;
}