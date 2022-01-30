import $ from "jquery";

import { expect, test } from "@jest/globals";
import { getCookie, initCookies, setCookie } from "../src/cookies";

test("Test cookie types", () => {
	global.webroot = "/";
	setCookie("name", "value", 100, "/");
	let value = getCookie("name");
	expect(value).toEqual("value");

	// test integer storage
	setCookie("intCookie", 32, 100, "/");
	let intCookie = getCookie("intCookie", {type: "int"});
	expect(intCookie).toStrictEqual(32);

	// test invalid integer
	setCookie("intCookie", "thirty-two", 100, "/");
	intCookie = getCookie("intCookie", {type: "int"});
	expect(intCookie).toStrictEqual(NaN);

	// test float storage
	setCookie("floatCookie", 3.14, 100, "/");
	let floatCookie = getCookie("floatCookie", {type: "float"});
	expect(floatCookie).toStrictEqual(3.14);

	// test invalid float
	setCookie("floatCookie", "abc", 100, "/");
	floatCookie = getCookie("floatCookie", {type: "float"});
	expect(floatCookie).toStrictEqual(NaN);

	// test boolean
	setCookie("boolCookie", true, 100, "/");
	let boolCookie = getCookie("boolCookie", {type: "bool"});
	expect(boolCookie).toStrictEqual(true);
});

test("Test JSON cookie storage", () => {
	let jsonObj = {
		"a": 32,
		"b": [1,2,3],
		"c": {
			"fffff": null
		}
	};
	// valid JSON
	setCookie("jsonCookie", JSON.stringify(jsonObj));
	let cookieObj = getCookie("jsonCookie", {type: "json"});
	expect(cookieObj).toEqual(jsonObj);

	// Invalid JSON stored, getCookie returns {} here
	setCookie("invalidJSON", "{");
	cookieObj = getCookie("invalidJSON", {type: "json"});
	expect(cookieObj).toEqual({});
});