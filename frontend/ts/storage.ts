import { getCookie, setCookie } from "./cookies";


export function getStorageVal(key: string, defaultVal = "") {
	if(localStorage == undefined)
		return getCookie(key, defaultVal);
	let val = localStorage.getItem(key);
	if(val === null)
		return defaultVal;
	return val;
}

export function getBooleanStorageVal(key: string, defaultVal = false) {
	let val = getStorageVal(key, defaultVal?"true":"false");
	return val == "true";
}

export function getNumberStorageVal(key: string, defaultVal = 0) {
	return Number.parseFloat(getStorageVal(key, defaultVal.toString()));
}

export function getJsonStorageVal<T>(key: string, defaultVal: T) {
	let val = defaultVal;
	try {
		val = JSON.parse(getStorageVal(key, defaultVal as string));
	} catch(e) {
		val = defaultVal;
	}
	return val;
}

export function setStorageVal(key: string, val: any, isJSON = false) {
	let storeVal = isJSON?JSON.stringify(val):val;
	if(localStorage == undefined)
		setCookie(key, storeVal);
	else
		localStorage.setItem(key, storeVal);
}