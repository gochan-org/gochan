import { expect, test } from "@jest/globals";
import { getBooleanCookie, getCookie, getNumberCookie, initCookies, setCookie } from "../ts/cookies";
import { getBooleanStorageVal, getJsonStorageVal, getNumberStorageVal, getStorageVal, setStorageVal } from "../ts/storage";

global.webroot = "/";
initCookies();

test("Test cookie types", () => {
	setCookie("name", "value", "100", "/");
	const value = getCookie("name");
	expect(value).toStrictEqual("value");

	// test number storage
	setCookie("numberCookie", "32", "100", "/");
	const numberCookie = getNumberCookie("numberCookie");
	expect(numberCookie).toStrictEqual(32);

	setCookie("boolCookie", "true", "100", "/");
	const boolCookie = getBooleanCookie("boolCookie");
	expect(boolCookie).toStrictEqual(true);

});

test("Test localStorage", () => {
	setStorageVal("name", "value");
	const value = getStorageVal("name");
	expect(value).toStrictEqual("value");

	setStorageVal("numberVal", 33.2);
	const numberVal = getNumberStorageVal("numberVal");
	expect(numberVal).toStrictEqual(33.2);

	setStorageVal("boolVal", true);
	const boolVal = getBooleanStorageVal("boolVal");
	expect(boolVal).toStrictEqual(true);

	setStorageVal("jsonVal", `{
		"key1": "val1",
		"key2": 33,
		"aaa": [1,2,3]
	}`);
	const jsonVal = getJsonStorageVal<{[k:string]:any}>("jsonVal", {});
	expect(jsonVal).toStrictEqual({
		"key1": "val1",
		"key2": 33,
		"aaa": [1,2,3]
	});
});
