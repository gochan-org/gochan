// Make the sections table on /manage/boardsections sortable to make changing the list order easier

import $ from "jquery";
import "jquery-ui/ui/widget";
import "jquery-ui/ui/widgets/mouse";
import "jquery-ui/ui/data";
import "jquery-ui/ui/widgets/sortable";
import { alertLightbox } from "../dom/lightbox";

let $sectionsTable: JQuery<HTMLTableElement> = null;
let changesButtonAdded = false;
const initialOrders: string[] = [];

async function applyOrderChanges() {
	const $sections = $sectionsTable.find("tr.sectionrow");
	let errorShown = false; // only show one error if something goes wrong
	const sectionsArr = $sections.toArray();
	for(const el of sectionsArr) {
		const $el = $(el);
		const updatesection = /^section(\d+)$/.exec(el.id)[1];
		const sectionname = $el.find(":nth-child(1)").html();
		const sectionabbr = $el.find(":nth-child(2)").html();
		const sectionpos = $el.find(":nth-child(3)").html();
		const sectionhidden = $el.find(":nth-child(4)").html().toLowerCase() === "yes"?"on":"off";
		
		const searchParams = new URLSearchParams();
		searchParams.append("updatesection", updatesection);
		searchParams.append("sectionname", sectionname);
		searchParams.append("sectionabbr", sectionabbr);
		searchParams.append("sectionpos", sectionpos);
		searchParams.append("sectionhidden", sectionhidden);
		searchParams.append("save_section", "Save section");
		await fetch(`${webroot}manage/boardsections`, {
			method: "POST",
			body: searchParams,
			credentials: "same-origin"
		}).then(response => {
			if(!response.ok) {
				return Promise.reject(response.statusText);
			}
			alertLightbox("Section order changes saved successfully!", "Success");
			changesButtonAdded = false;
			$("div#save-changes").remove();
		}).catch(errorText => {
			if(!errorShown) {
				alertLightbox(`Received an error when saving changes (only the first one will be shown): ${errorText}`, "Error");
				errorShown = true;
			}
		});
	}
}

function cancelOrderChanges() {
	$sectionsTable.find("tbody").sortable("cancel");
	const $sections = $sectionsTable.find("tr.sectionrow");
	$sections.each((i, el) => {
		$(el).find(":nth-child(3)").text(initialOrders[i]);
	});
	changesButtonAdded = false;
	$("div#save-changes").remove();
}

function addButtons() {
	if(changesButtonAdded) return;
	$("<div/>").prop({id: "save-changes"}).append(
		$("<button/>")
			.text("Apply order changes")
			.on("click", applyOrderChanges),
		$("<button/>")
			.text("Cancel")
			.on("click", cancelOrderChanges)
	).insertAfter($sectionsTable);
	changesButtonAdded = true;
}

$(() => {
	if(window.location.pathname !== webroot + "manage/boardsections")
		return;

	$sectionsTable = $("table#sections");
	$sectionsTable.prev().append(" (drag to rearrange)");
	$sectionsTable.find("tbody").sortable({
		items: "tr.sectionrow",
		stop: () => {
			$sectionsTable.find("tr.sectionrow").each((i, el) => {
				const $order = $(el).find(":nth-child(3)");
				initialOrders.push($order.text());
				$order.text(i + 1);
			});
			addButtons();
		}
	});
});