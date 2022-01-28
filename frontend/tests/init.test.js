import {test, expect} from "@jest/globals";
import "./inittests";

test("Checks for valid mock server timezone (serverTZ)", () => {
	expect(isNaN(serverTZ)).toBe(false);
});

test("Checks mock themes to make sure the default one (defaultStyle) exists and is pipes.css", () => {
	let styleName = "";
	for(const style of styles) {
		if(style.Filename == defaultStyle) {
			styleName = style.Name;
		}
	}
	expect(styleName).toBe("Pipes");
});

test("tests mock webroot, should be /", () => {
	expect(webroot).toBe("/");
});