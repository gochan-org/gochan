import $ from "jquery";

import { alertLightbox } from "./lightbox";

const uploadReader = new FileReader();
uploadReader.onload = onReaderLoad;
uploadReader.onerror = onReaderError;
const noop = () => {
	return;
};

export function updateUploadImage($elem: JQuery<HTMLElement>, onLoad = noop) {
	if($elem.length === 0) return;
	$elem[0].onchange = function() {
		const img = new Image();
		img.src = URL.createObjectURL((this as any).files[0]);
		img.onload = onLoad;
	};
}

export function getUploadFilename(): string {
	const elem = document.getElementById("imagefile") as HTMLInputElement;
	if(elem === null) return "";
	if(elem.files === undefined || elem.files.length < 1) return "";
	return elem.files[0].name;
}

function dragAndDrop(e:JQuery.DragEnterEvent|JQuery.DragOverEvent|JQuery.DropEvent) {
	e.preventDefault();
	const $browseBtn = $<HTMLInputElement>("input[name=imagefile]");
	if($browseBtn.length < 1) return;

	if(e.type === "dragenter" || e.type === "dragover") {
		e.stopPropagation();
	} else {
		$browseBtn[0].files = e.originalEvent.dataTransfer.files;
		uploadReader.readAsDataURL($browseBtn[0].files[0]);
	}
}

function onReaderLoad(e:ProgressEvent<FileReader>) {
	const base64Str = e.target.result.toString();
	const isImage = base64Str.slice(5, 10) === "image";

	if(isImage) {
		$("<img/>").attr({
			"src": base64Str,
		}).addClass("upload-preview").appendTo("div#new-upload-box");
	}
}

function onReaderError(e:ProgressEvent<FileReader>) {
	alertLightbox(`Unable to load file: ${e.target.error.message}`);
}

function onBrowseBtnChange(e:JQuery.ChangeEvent) {
	uploadReader.readAsDataURL(e.target.files[0]);
}

$(() => {
	const $browseBtn = $<HTMLInputElement>("input[name=imagefile]").hide();
	if($browseBtn.length !== 1) return;
	$browseBtn.on("change", onBrowseBtnChange);

	$("<div/>").attr("id", "new-upload-box").append(
		$("<a/>")
			.attr("href", "javascript:;")
			.text("Select/drop/paste upload here")
			.on("click", () => $browseBtn.trigger("click"))
	).on("dragenter dragover drop", dragAndDrop).insertBefore($browseBtn);
});