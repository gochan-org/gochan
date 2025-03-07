/* global simpleHTML */
import {test, expect} from "@jest/globals";

import $ from "jquery";
import "../ts/vars";
import "./inittests";

import { applyBBCode, handleKeydown } from "../ts/boardevents";

document.documentElement.innerHTML = simpleHTML;

function doBBCode(key:string, text: string, start: number, end: number) {
	const $ta = $<HTMLTextAreaElement>("<textarea/>");
	$ta.text(text);
	const e = $.Event("keydown");
	e.ctrlKey = true;
	$ta[0].selectionStart = start;
	$ta[0].selectionEnd = end;
	e.key = key;
	$ta.first().trigger(e);
	applyBBCode(e as JQuery.KeyDownEvent);
	return $ta.text();
}

test("Tests BBCode events", () => {
	let text = doBBCode("b", "bold", 0, 4);
	expect(text).toEqual("[b]bold[/b]");
	text += "italics";
	text = doBBCode("i", text, text.length - 7, text.length);
	expect(text).toEqual("[b]bold[/b][i]italics[/i]");

	text = doBBCode("r", "strike" + text, 0, 6);
	expect(text).toEqual("[s]strike[/s][b]bold[/b][i]italics[/i]");

	text = doBBCode("s", text, 0, 13);
	expect(text).toEqual("[?][s]strike[/s][/?][b]bold[/b][i]italics[/i]");

	text = doBBCode("u", text, text.length, text.length);
	expect(text).toEqual("[?][s]strike[/s][/?][b]bold[/b][i]italics[/i][u][/u]");

	const invalidKeyCode = doBBCode("x", text, 0, 1); // passes an invalid keycode to applyBBCode, no change
	expect(invalidKeyCode).toEqual(text);
});

test("Tests proper form submission via JS", () => {
	const $form = $("form#postform");
	const text = doBBCode("s", "text", 0, 4);
	$form.find("textarea#postmsg").text(text);
	let submitted = false;
	$form.on("submit", () => {
		submitted = true;
		return false;
	});
	const e = $.Event("keydown");
	e.ctrlKey = true;
	e.key = "Enter";
	$form.find("textarea#postmsg").first().trigger(e);
	handleKeydown(e as JQuery.KeyDownEvent);
	expect(submitted).toBeTruthy();
});