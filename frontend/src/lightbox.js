import $ from "jquery";

const emptyFunc = () => {};

export function showLightBox(title, innerHTML) {
	$(document.body).prepend(
		`<div class="lightbox-bg"></div><div class="lightbox"><div class="lightbox-title">${title}<a href="#" class="lightbox-x">X</a><hr /></div>${innerHTML}</div>`
	);
	$("a.lightbox-x, .lightbox-bg").on("click", () => {
		$(".lightbox, .lightbox-bg").remove();
	});
}

function simpleLightbox(properties = {}, customCSS = {}, $elements = []) {
	if(properties["class"] === undefined)
		properties["class"] = "lightbox"
	defaultCSS = {
		"display": "inline-block",
		"top": "50%",
		"left": "50%",
		"transform": "translate(-50%, -50%)",
		"max-width": "80%",
		"max-height": "80%",
		"right": "auto",
		"bottom": "auto"
	};
	for (const key in defaultCSS) {
		if(customCSS[key] === undefined)
			customCSS[key] = defaultCSS[key];
	}

	let $box = $("<div/>").prop(properties).css(customCSS).prependTo(document.body).append($elements);
	let boxBg = $("<div />").prop({
		class: "lightbox-bg"
	}).on("click", function() {
		this.remove();
		$box.remove();
	}).prependTo(document.body);

	return $box;
}

export function promptLightbox(defVal = "", isMasked = false, onOk = emptyFunc, title = "") {
	let $ok = $("<button/>").prop({
		"id": "okbutton"
	}).text("OK");
	let $cancel = $("<button/>").prop({
		"id": "cancelbutton"
	}).text("Cancel");

	let val = (typeof defVal == "string")?defVal:"";
	let $promptInput = $("<input/>").prop({
		id: "promptinput",
		type: isMasked?"password":"text"
	}).val(val);

	let $form = $("<form/>").prop({
		"action": "javascript:;",
		"autocomplete": "off"
	}).append(
		$("<b/>").text(title),
		$promptInput,
		"<br/><br/>",
		$ok,
		$cancel
	);
	let $lb = simpleLightbox({}, {}, [$form]);

	$ok.on("click", function() {
		if(onOk($lb, $promptInput.val()) == false)
			return;
		this.remove();
		$lb.remove();
		$(".lightbox-bg").remove();
	});
	$cancel.on("click", function() {
	this.remove();
		$lb.remove();
		$(".lightbox-bg").remove();
	});
	return $lb;
}

export function alertLightbox(msg = "", title = location.hostname, onOk = emptyFunc) {
	let $ok = $("<button/>").prop({
		"id": "okbutton"
	}).text("OK");
	let $lb = simpleLightbox({}, {}, [
		$("<b/>").prop({id:"alertTitle"}).text(title),
		"<hr/>",
		$("<span/>").prop({id:"alertText"}).text(msg),
		"<br/>",
		$ok
	]);
	$ok.on("click", function() {
		onOk($lb);
		this.remove();
		$lb.remove();
		$(".lightbox-bg").remove();
	});
	return $lb;
}
