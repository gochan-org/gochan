import $ from "jquery";

const YEAR_IN_MS = 365*24*60*60*1000;

/**
 * @param {string} name
 */
export function getCookie(name: string, defaultVal = "") {
	let val = defaultVal;
	let cookieArr = document.cookie.split("; ");

	for(const cookie of cookieArr) {
		let pair = cookie.split("=");
		if(pair[0] != name) continue;
		try {
			val = decodeURIComponent(pair[1]).replace("+", " ");
		} catch(err) {
			console.error(`Error parsing cookie value for "${name}": ${err}`);
			return defaultVal;
		}
	}
	return val;
}

export function getNumberCookie(name: string, defaultVal = "0") {
	return parseFloat(getCookie(name, defaultVal));
}

export function getBooleanCookie(name: string, defaultVal = "true") {
	return getCookie(name, defaultVal) == "true";
}

function randomPassword(len = 8) {
	const validChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!#$%&'*+-.^_`|~";
	
	let pass = "";
	for(let p = 0; p < len; p++) {
		pass += validChars[Math.floor(Math.random() * validChars.length)];
	}
	return pass;
}

/**
 * Set a cookie
 */
export function setCookie(name: string, value: string, expires = "", root = webroot) {
	let expiresStr = "";
	if(expires == "") {
		expiresStr = ";expires=";
		let d = new Date();
		d.setTime(d.getTime() + YEAR_IN_MS);
		expiresStr += d.toUTCString();
	}
	document.cookie = `${name}=${value}${expiresStr};path=${root};sameSite=strict`;
}

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

$(initCookies);
