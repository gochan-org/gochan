import { test, expect, vi } from "vitest";

import $ from "jquery";
import "../ts/vars";
import "./inittests";

import { applyBBCode, handleKeydown } from "../ts/boardevents";
import { $qr, initQR, openQR } from "../ts/dom/qr";

document.documentElement.innerHTML = (global as unknown as {simpleHTML:string}).simpleHTML;

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

	text = doBBCode("r", `strike${text}`, 0, 6);
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
	const submitHandler = vi.fn();
	$form.find("textarea#postmsg").text(doBBCode("s", "text", 0, 4));
	$("<input/>").prop({
		"type": "hidden",
		"name": "boardid"
	}).appendTo($form);
	initQR();
	$form.on("submit", (e: JQuery.SubmitEvent) => {
		e.preventDefault();
		submitHandler();
	});
	const e = $.Event("keydown", {
		ctrlKey: true,
		key: "Enter",
		target: $form.find("textarea#postmsg").first()[0]
	}) as JQuery.KeyDownEvent;
	$(document).on("keydown", handleKeydown);
	$form.find("textarea#postmsg").first().trigger(e);
	expect(submitHandler).toHaveBeenCalled();
	expect($qr).toHaveLength(1);
	expect($qr).toSatisfy((el: JQuery<HTMLElement>) => el.css("display") === "block");
});

test("Tests QR box open/close", () => {
	const $form = $("form#postform");
	expect($form).toHaveLength(1);
	$("<input/>").prop({
		"type": "hidden",
		"name": "boardid"
	}).appendTo($form);
	initQR();
	openQR();
	$(document).on("keydown", handleKeydown);

	expect($qr).toHaveLength(1);
	expect($qr).toSatisfy((el: JQuery<HTMLElement>) => el.css("display") === "block");
	const oldHide = $qr.hide;
	const hideSpy = vi.spyOn($qr, "hide").mockImplementation(function(this: JQuery<HTMLElement>) {
		oldHide.apply(this);
		return this;
	});
	const oldShow = $qr.show;
	const showSpy = vi.spyOn($qr, "show").mockImplementation(function(this: JQuery<HTMLElement>) {
		oldShow.apply(this);
		return this;
	});
	expect(hideSpy).not.toHaveBeenCalled();
	expect(showSpy).not.toHaveBeenCalled();

	const $closeBtn = $qr.find("a#close-btn");
	expect($closeBtn).toHaveLength(1);
	$closeBtn.trigger("click");
	expect(hideSpy).toHaveBeenCalled();

	const qPress = $.Event("keydown", {
		key: "q",
		target: document.body
	}) as JQuery.KeyDownEvent;
	$(document).trigger(qPress);
	// expect(showSpy).toHaveBeenCalled();
});