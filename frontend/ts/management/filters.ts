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
	const isBoolean = e.target.value === "firsttime" || e.target.value === "hasfile" || e.target.value === "isop";
	const noRegex = isBoolean || e.target.value === "filechecksum" || e.target.value === "imgfingerprint";
	const $searchInput = $fieldset.find("tr.search-cndtn");

	if(isBoolean) {
		$searchInput.hide();
	} else {
		$searchInput.show();
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
	$("input#allboards").on("change", () => {
		$<HTMLInputElement>("td#boardslist input[type=checkbox]").each((_i, el) => {
			if(el.id !== "allboards") {
				el.disabled = $<HTMLInputElement>("input#allboards")[0].checked;
			}
		});
	});
}

$(() => {
	applyConditionEvents($("form#filterform fieldset.fld-cndtns"));

	$<HTMLSelectElement>("select#action").on("change", e => {
		switch(e.target.value) {
		case "reject":
			$("th#reason").parent().show();
			break;
		case "ban":
			$("th#reason").parent().show();
			break;
		case "log":
			$("th#reason").parent().hide();
			break;
		default:
			console.log(e.target.value);
			break;
		}
	});
});