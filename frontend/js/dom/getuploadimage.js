let noop = ()=>{};

/**
 * @param {JQuery<HTMLElement>} $elem
 */
export function updateUploadImage($elem, onLoad = noop) {
	if($elem.length == 0) return;
	$elem[0].onchange = function() {
		let img = new Image();
		img.src = URL.createObjectURL(this.files[0]);
		img.onload = onLoad;
	};
}