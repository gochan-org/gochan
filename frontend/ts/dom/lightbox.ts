import $ from "jquery";

export function removeLightbox(...customs: any) {
	$(".lightbox, .lightbox-bg").remove();
	for(const custom of customs) {
		$(custom).remove();
	}
}

export function showLightBox(title: string, innerHTML: string) {
	$(document.body).prepend(
		`<div class="lightbox-bg"></div><div class="lightbox"><div class="lightbox-title">${title}<a href="javascript:;" class="lightbox-x">X</a><hr /></div>${innerHTML}</div>`
	);
	$("a.lightbox-x, .lightbox-bg").on("click", removeLightbox);
}


function simpleLightbox(properties: any = {}, customCSS: any = {}, $elements: any[] = []) {
	if(properties["class"] === undefined)
		properties["class"] = "lightbox";
	const defaultCSS: {[key: string]: string} = {
		"display": "inline-block",
		"top": "50%",
		"left": "50%",
		"transform": "translate(-50%, -50%)",
		"max-width": "80%",
		"max-height": "80%",
		"right": "auto",
		"bottom": "auto"
	};
	for(const key in defaultCSS) {
		if(customCSS[key] === undefined)
			customCSS[key] = defaultCSS[key];
	}

	const $box = $("<div/>").prop(properties).css(customCSS).prependTo(document.body).append($elements);
	$("<div />").prop({
		class: "lightbox-bg"
	}).on("click", function() {
		removeLightbox(this);
	}).prependTo(document.body);

	return $box;
}

export function promptLightbox(defVal = "", isMasked = false, onOk?: ($el:JQuery<HTMLElement>, val: any) => any, title = "") {
	const $ok = $("<button/>").prop({
		"id": "okbutton"
	}).text("OK");
	const $cancel = $("<button/>").prop({
		"id": "cancelbutton"
	}).text("Cancel");

	const val = (typeof defVal === "string")?defVal:"";
	const $promptInput = $("<input/>").prop({
		id: "promptinput",
		type: isMasked?"password":"text"
	}).val(val);

	const $form = $("<form/>").prop({
		"action": "javascript:;",
		"autocomplete": "off"
	}).append(
		$("<b/>").text(title),
		$promptInput,
		"<br/><br/>",
		$ok,
		$cancel
	);
	const $lb = simpleLightbox({}, {}, [$form]);
	$promptInput.trigger("focus");
	$ok.on("click", function() {
		if(onOk && onOk($lb, $promptInput.val()) === false)
			return;
		removeLightbox(this, $lb);
	});
	$cancel.on("click", function() {
		removeLightbox(this, $lb);
	});
	return $lb;
}

export function alertLightbox(msg = "", title = location.hostname, onOk?: ($el: JQuery<HTMLElement>) => any) {
	const $ok = $("<button/>").prop({
		"id": "okbutton"
	}).text("OK");
	const $lb = simpleLightbox({}, {}, [
		$("<b/>").prop({id:"alertTitle"}).text(title),
		"<hr/>",
		$("<span/>").prop({id:"alertText"}).text(msg),
		"<br/>",
		$ok
	]);
	$ok.trigger("focus");
	$ok.on("click", function() {
		if(onOk)
			onOk($lb);
		removeLightbox(this, $lb);
	});
	return $lb;
}
