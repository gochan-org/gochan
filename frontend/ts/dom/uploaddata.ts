let noop = ()=>{};

export function updateUploadImage($elem: JQuery<HTMLElement>, onLoad = noop) {
	if($elem.length == 0) return;
	$elem[0].onchange = function() {
		let img = new Image();
		img.src = URL.createObjectURL((this as any).files[0]);
		img.onload = onLoad;
	};
}

export function getUploadFilename(): string {
	let elem = document.getElementById("imagefile") as HTMLInputElement;
	if(elem === null) return "";
	if(elem.files === undefined || elem.files.length < 1) return "";
	return elem.files[0].name;
}