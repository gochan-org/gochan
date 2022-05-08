import { getCookie, setCookie } from "./cookies";


export function getStorageVal(key, defaultVal = "") {
	if(localStorage == undefined)
		return getCookie(key, defaultVal);
	let val = localStorage.getItem(key);
	if(val === null)
		return defaultVal;
	return val;
}

export function getBooleanStorageVal(key, defaultVal = false) {
	let val = getStorageVal(key, defaultVal);
	return val == true || val == "true";
}

export function getNumberStorageVal(key, defaultVal = 0) {
	return Number.parseFloat(getStorageVal(key, defaultVal))
}

export function getJsonStorageVal(key, defaultVal) {
	let val = defaultVal;
	try {
		val = JSON.parse(getStorageVal(key, defaultVal))
	} catch(e) {
		val = defaultVal;
	}
	return val;
}

export function setStorageVal(key, val) {
	if(localStorage == undefined)
		setCookie(key, val);
	else
		localStorage.setItem(key, val);
}