import $ from "jquery";

function onAddCondition(e:JQuery.ClickEvent) {
	e.preventDefault();
	const newFieldsetNum = $("fieldset").length + 1;
	const $newFieldset = $("fieldset.fld-cndtns").first().clone(true, true);
	$newFieldset.find<HTMLInputElement>("input,select").each((_i,el) => {
		const matches = /^([^0-9]+)\d+$/.exec(el.name);
		if(matches !== null) {
			el.name = matches[1] + newFieldsetNum;
		}
	});
	$newFieldset.find("select").trigger("change", "name");
	$newFieldset.appendTo("td#conditions");
}

function onRemoveCondition(e:JQuery.ClickEvent) {
	e.preventDefault();
	const $allConditionalGroups = $("fieldset.fld-cndtns");
	if($allConditionalGroups.length > 1)
		$(e.target).parents("fieldset").remove();
}

function onFieldChange(e:JQuery.ChangeEvent) {
	const $fieldset = $(e.target).parents("fieldset");
	const isBoolean = e.target.value === "firsttimeboard" || e.target.value === "notfirsttimeboard" ||
		e.target.value === "firsttimesite" || e.target.value === "notfirsttimesite" || e.target.value === "isop" ||
		e.target.value === "notop" || e.target.value === "hasfile" || e.target.value === "nofile";
	const noRegex = isBoolean || e.target.value === "checksum" || e.target.value === "ahash";
	const $searchContainer = $fieldset.find("tr.search-cndtn");
	if(isBoolean) {
		$searchContainer.hide();
	} else {
		$searchContainer.show();
	}

	if(noRegex) {
		$fieldset.find(".regex-cndtn").hide();
	} else {
		$fieldset.find(".regex-cndtn").show();
	}
}

function applyConditionEvents($fieldset:JQuery<HTMLElement>) {
	$("#add-cndtn").on("click", onAddCondition);
	$fieldset.find(".rem-cndtn").on("click", onRemoveCondition);
	$fieldset.find("select.sel-field").on("change", onFieldChange);
}

$(() => {
	applyConditionEvents($("form#filterform fieldset.fld-cndtns"));

	$<HTMLSelectElement>("select#action").on("change", e => {
		switch(e.target.value) {
		case "reject":
			$("th#detail").parent().show();
			break;
		case "ban":
			$("th#detail").parent().show();
			break;
		case "log":
			$("th#detail").parent().hide();
			break;
		default:
			console.log(e.target.value);
			break;
		}
	});
});