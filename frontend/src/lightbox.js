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
	$(document.body).prepend(`<div class="lightbox-bg"></div><div class="lightbox-msg">${msg}<br /><button class="lightbox-msg-ok" style="float: right; margin-top: 8px;">OK</button></div>`);
	let centeroffset = parseInt($(".lightbox-msg").css("transform-origin").replace("px",""),10)+$(".lightbox-msg").width()/2;

	$(".lightbox-msg").css({
		"position": "fixed",
		"left": $(document).width()/2 - centeroffset/2-16
	});

	$(".lightbox-msg-ok, .lightbox-bg").on("click", () => {
		$(".lightbox-msg, .lightbox-bg").remove();
	});
}