'use strict';
import {test, expect, jest} from "@jest/globals";

import $ from "jquery";
import "./inittests";

import { applyBBCode, handleKeydown } from "../js/boardevents";

document.documentElement.innerHTML = simpleHTML;

function doBBCode(keycode, text, start, end) {
	let $ta = $("<textarea/>")
	$ta.text(text);
	let e = $.Event("keydown");
	e.ctrlKey = true;
	$ta[0].selectionStart = start;
	$ta[0].selectionEnd = end;
	e.keyCode = keycode;
	e.which = keycode;
	e.target = $ta[0];
	applyBBCode(e);
	return $ta.text();
}

test("Tests BBCode events", () => {
	let text = doBBCode(66, "bold", 0, 4);
	expect(text).toEqual("[b]bold[/b]");
	text += "italics";
	text = doBBCode(73, text, text.length - 7, text.length);
	expect(text).toEqual("[b]bold[/b][i]italics[/i]");

	text = doBBCode(82, "strike" + text, 0, 6);
	expect(text).toEqual("[s]strike[/s][b]bold[/b][i]italics[/i]");
	
	text = doBBCode(83, text, 0, 13);
	expect(text).toEqual("[?][s]strike[/s][/?][b]bold[/b][i]italics[/i]");

	text = doBBCode(85, text, text.length, text.length);
	expect(text).toEqual("[?][s]strike[/s][/?][b]bold[/b][i]italics[/i][u][/u]");

	let invalidKeyCode = doBBCode(0, text, 0, 1); // passes an invalid keycode to applyBBCode, no change
	expect(invalidKeyCode).toEqual(text);
});

test("Tests proper form submission via JS", () => {
	let $form = $("form#postform")
	let text = doBBCode(83, "text", 0, 4);
	$form.find("textarea#postmsg").text(text);
	let submitted = false;
	$form.on("submit", function(e) {
		submitted = true;
		return false;
	});
	let e = $.Event("keydown");
	e.ctrlKey = true;
	e.keyCode = 10;
	e.target = $form.find("textarea#postmsg")[0];
	handleKeydown(e);
	expect(submitted).toBeTruthy();
});