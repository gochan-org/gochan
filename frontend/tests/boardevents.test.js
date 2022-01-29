'use strict';
import {test, expect, jest} from "@jest/globals";
// jest.dontMock("jquery");

import $ from "jquery";
import "./inittests";

import { applyBBCode, handleActions, handleKeydown } from "../src/boardevents";

document.documentElement.innerHTML = simpleHTML;

let $mockTextArea = $("form#postform textarea#postmsg");


test("Checks to make sure post form is ready for testing", () => {
	expect($mockTextArea.length).toBeGreaterThan(0);
});

test("Checks BBCode application", () => {
	$(document).on("keydown", handleKeydown);
	let e = $.Event("keydown");
	e.ctrlKey = true;
	e.which = 85;
	e.keyCode = 85;
	$mockTextArea.text("text here");
	$mockTextArea[0].selectionStart = 0;
	$mockTextArea[0].selectionEnd = 4;
	$mockTextArea.trigger(e);
	expect($mockTextArea.text()).toEqual("[u]text[/u] here");

	e.which = 66;
	e.keyCode = 66;
	$mockTextArea[0].selectionStart = 12;
	$mockTextArea[0].selectionEnd = 16;
	$mockTextArea.trigger(e);
	expect($mockTextArea.text()).toEqual("[u]text[/u] [b]here[/b]");
});