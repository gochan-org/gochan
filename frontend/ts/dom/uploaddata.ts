import $ from "jquery";

import { alertLightbox } from "./lightbox";
import { getBooleanStorageVal } from "../storage";


export function updateUploadImage($elem: JQuery<HTMLElement>, onLoad?: () => unknown) {
	if($elem.length === 0) return;
	$elem[0].onchange = function() {
		const img = new Image();
		const file = (this as HTMLInputElement).files?.[0];
		if(!file) return;
		img.src = URL.createObjectURL(file);
		if(onLoad)
			img.onload = onLoad;
	};
}

export function getUploadFilename(): string {
	const elem = document.getElementById("imagefile") as HTMLInputElement;
	if(!elem) return "";
	if(!elem.files || elem.files.length < 1) return "";
	return elem.files[0].name;
}

function dragAndDrop(e:JQuery.DragEnterEvent|JQuery.DragOverEvent|JQuery.DropEvent) {
	e.preventDefault();
	const $browseBtn = $<HTMLInputElement>("input[name=imagefile]");
	if($browseBtn.length < 1) return;

	if(e.type === "dragenter" || e.type === "dragover") {
		e.stopPropagation();
	} else {
		const dataTransfer = e.originalEvent?.dataTransfer;
		if(!dataTransfer) return;
		$browseBtn.each((_, el) => {
			el.files = dataTransfer.files;
			addFileUpload(el.files[0]);
		});
	}
}

function onReaderLoad(name:string, e:ProgressEvent<FileReader>) {
	if(!e.target?.result) return;
	const base64Str = e.target.result.toString();
	const isImage = base64Str.slice(5, 10) === "image";
	const extPos = name.lastIndexOf(".");
	const namePart = (extPos > 0)?name.substring(0, extPos):name;
	const extPart = (extPos <= 0)?"":name.substring(extPos);
	const maxLen = 20;
	const nameShortened = namePart.substring(0, maxLen) + ((namePart.length > maxLen)?"…":"") + extPart;
	const $container = ($("div#upload-box").children(".upload-preview-container").length > 0)?
		$(".upload-preview-container"):
		$("<div/>").addClass("upload-preview-container").appendTo("div#upload-box");

	$container.empty().append(
		$("<a/>").attr({
			"class": "upload-x",
			"href": "#"
		}).text("X").on("click", (e:JQuery.ClickEvent) => {
			const $target = $(e.target);
			const $browseBtn = $target.parents<HTMLInputElement>("#upload-box").siblings<HTMLInputElement>("input[name=imagefile]");
			$browseBtn.each((_, el) => {
				el.value = "";
			});
			$target.parents(".upload-preview-container").remove();
		}),

		isImage?$("<img/>").attr({
			"class": "upload-preview",
			"src": base64Str,
		}):$("<div/>").addClass("placeholder-thumb"),

		$("<span/>").addClass("upload-filename").attr({title: name}).text(nameShortened)
	);
}

function addFileUpload(file:File) {
	if(!file) return;
	const uploadReader = new FileReader();
	uploadReader.onload = (e => onReaderLoad(file.name, e));
	uploadReader.onerror = (e) => alertLightbox(`Unable to load file: ${e.target?.error?.message}`);
	uploadReader.readAsDataURL(file);
}

function replaceBrowseButton() {
	const $browseBtn = $<HTMLInputElement>("input[name=imagefile]").hide();
	if($browseBtn.length < 1 || $("div#upload-box").length > 0) return;
	$browseBtn.on("change", e => addFileUpload(e.target?.files?.[0] as File));

	$("<div/>").attr("id", "upload-box").append(
		$("<a/>").addClass("browse-text")
			.attr("href", "#")
			.text("Select/drop/paste upload here")
			.on("click", e => {
				e.preventDefault();
				$browseBtn.trigger("click");
			})
	).on("dragenter dragover drop", dragAndDrop as () => void).insertBefore($browseBtn);

	$("form#postform, form#qrpostform").on("paste", e => {
		const clipboardData = (e.originalEvent as ClipboardEvent).clipboardData;
		if(!clipboardData) return;
		if(clipboardData.items.length < 1) {
			alertLightbox("No files in clipboard", "Unable to paste");
			return;
		}
		if(clipboardData.items[0].kind !== "file") {
			return;
		}
		const clipboardFile = clipboardData.items[0].getAsFile();
		if(!clipboardFile) return;
		addFileUpload(clipboardFile);
		$browseBtn.each((_i,el) => {
			el.files = clipboardData.files;
		});
	});
}

export function updateBrowseButton() {
	const useNewUploader = getBooleanStorageVal("newuploader", true);
	if(useNewUploader) {
		replaceBrowseButton();
	} else {
		$("div#upload-box").remove();
		$("input[name=imagefile]").show();
	}
}