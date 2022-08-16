/* global webroot */

import $ from "jquery";

/**
 * @param {string} name
 */
export function getCookie(name, defaultVal = "") {
	let val = defaultVal;
	let cookieArr = document.cookie.split("; ");

	for(const cookie of cookieArr) {
		let pair = cookie.split("=");
		if(pair[0] != name) continue;
		try {
			val = decodeURIComponent(pair[1]);
		} catch(err) {
			console.error(`Error parsing cookie value for "${name}": ${err}`);
			return defaultVal;
		}
	}
	return val;
}

export function getNumberCookie(name, defaultVal = "0") {
	return parseFloat(getCookie(name, defaultVal));
}

export function getBooleanCookie(name, defaultVal = "true") {
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
		expiresStr = ";expires=";
		let d = new Date();
		d.setTime(d.getTime() + 1000*60*60*24*expires);
		expiresStr += d.toUTCString();
	}
	document.cookie = `${name}=${value}${expiresStr};path=${root};sameSite=strict`;
}