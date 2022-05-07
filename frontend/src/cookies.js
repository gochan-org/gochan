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

function randomPassword(len = 8) {
	const printableStart = 33;
	const printableEnd = 126;
	
	let pass = "";
	for(let p = 0; p < len; p++) {
		pass += String.fromCharCode(
			Math.floor(Math.random() * (printableEnd-printableStart+1)) + printableStart
		);
	}
	return pass;
}

/**
 * gets cookies ready to be used elsewhere
 */
export function initCookies() {
	let pwCookie = getCookie("password");
	if(pwCookie == "") {
		pwCookie = randomPassword();
		setCookie("password", pwCookie);
	}
	$("input[name=postname]").val(getCookie("name"));
	$("input[name=postemail]").val(getCookie("email"));
	$("input[name=postpassword]").val(pwCookie);
	$("input[name=delete-password]").val(pwCookie);
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
	document.cookie = `${name}=${value}${expiresStr};path=${root};sameSite=strict`;
}