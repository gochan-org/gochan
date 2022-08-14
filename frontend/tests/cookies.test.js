import { expect, test } from "@jest/globals";
import { getBooleanCookie, getCookie, getNumberCookie, initCookies, setCookie } from "../js/cookies";
import { getBooleanStorageVal, getJsonStorageVal, getNumberStorageVal, getStorageVal, setStorageVal } from "../js/storage";

global.webroot = "/";
initCookies();

test("Test cookie types", () => {
	setCookie("name", "value", 100, "/");
	let value = getCookie("name");
	expect(value).toStrictEqual("value");

	// test number storage
	setCookie("numberCookie", 32, 100, "/");
	let numberCookie = getNumberCookie("numberCookie");
	expect(numberCookie).toStrictEqual(32);

	setCookie("boolCookie", true, 100, "/");
	let boolCookie = getBooleanCookie("boolCookie");
	expect(boolCookie).toStrictEqual(true);

});

test("Test localStorage", () => {
	setStorageVal("name", "value");
	let value = getStorageVal("name");
	expect(value).toStrictEqual("value");

	setStorageVal("numberVal", 33.2);
	let numberVal = getNumberStorageVal("numberVal");
	expect(numberVal).toStrictEqual(33.2);

	setStorageVal("boolVal", true);
	let boolVal = getBooleanStorageVal("boolVal");
	expect(boolVal).toStrictEqual(true);

	setStorageVal("jsonVal", `{
		"key1": "val1",
		"key2": 33,
		"aaa": [1,2,3]
	}`);
	let jsonVal = getJsonStorageVal("jsonVal");
	expect(jsonVal).toStrictEqual({
		"key1": "val1",
		"key2": 33,
		"aaa": [1,2,3]
	});
});
