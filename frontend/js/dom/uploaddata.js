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

/**
 * @returns {string}
 */
export function getUploadFilename() {
	let elem = document.getElementById("imagefile");
	if(elem === null) return "";
	if(elem.files === undefined || elem.files.length < 1) return "";
	return elem.files[0].name;
}