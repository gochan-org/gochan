import $ from "jquery"

export function showLightBox(title, innerHTML) {
	$(document.body).prepend(
		`<div class="lightbox-bg"></div><div class="lightbox"><div class="lightbox-title">${title}<a href="#" class="lightbox-x">X</a><hr /></div>${innerHTML}</div>`
	);
	$("a.lightbox-x, .lightbox-bg").on("click", () => {
		$(".lightbox, .lightbox-bg").remove();
	});
}

// opens up a lightbox for use as a message box that will look the same on all browsers
export function showMessage(msg) {
	let boxMsg = $("<div />").prop({
		class: "lightbox-msg"
	}).css({
		"text-align": "center"
	}).text(msg);

	let boxBtn = $("<button />").prop({
		class: "lightbox-msg-ok",
	}).css({
		"display": "block",
		"margin-left": "auto",
		"margin-right": "auto",
		"padding": "5px 10px 5px 10px"
	}).text("OK");

	let box = $("<div />").prop({
		class: "lightbox"
	}).css({
		"bottom": "inherit",
		"margin-top": "40px",
		"margin-bottom": "80px",
		"display": "inline-block",
		"padding": "80px"
	}).append(boxMsg, "<br />", boxBtn)
	.prependTo(document.body);

	let boxBg = $("<div />").prop({
			class: "lightbox-bg"
		}).prependTo(document.body);

	$(".lightbox-msg-ok, .lightbox-bg").on("click", () => {
		boxMsg.remove();
		box.remove();
		boxBg.remove();
	});
}