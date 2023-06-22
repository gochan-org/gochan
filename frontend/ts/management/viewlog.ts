import $ from "jquery";

interface LogFilter {
	showFatal: boolean;
	showErrors: boolean;
	showWarnings: boolean;
	showInfo: boolean;
	sortDesc?: boolean;
}

let originalLog = "";

function updateLogFilter($log: JQuery<HTMLTextAreaElement>, filter: LogFilter) {
	const lines = originalLog.split("\n").filter((line) => {
		try {
			const lineObj = JSON.parse(line);
			switch(lineObj.level) {
			case "fatal":
				return filter.showFatal;
			case "error":
				return filter.showErrors;
			case "warn":
				return filter.showWarnings;
			case "info":
				return filter.showInfo;
			default:
				console.warn("Unrecognized log level in line:", lineObj);
			}
			return false;
		} catch(_) {
			return false;
		}
	}).sort((a: string, b: string) => {
		const aObj = JSON.parse(a);
		const bObj = JSON.parse(b);
		if(aObj.time === undefined || bObj.time === undefined)
			return 0;

		if(filter.sortDesc)
			return Date.parse(bObj.time) - Date.parse(aObj.time);
		return Date.parse(aObj.time) - Date.parse(bObj.time);
	});

	$log.text(lines.join("\n"));
	$("span#log-lines").text(lines.length);
}

$(() => {
	if(location.pathname.indexOf(webroot + "manage/viewlog") !== 0)
		return;
	const $log = $<HTMLTextAreaElement>("textarea.viewlog");
	originalLog = $log.text();
	const $filterTable = $("<table/>")
		.attr("id", "log-filter")
		.css({
			"width": "80%",
			"margin-left": "auto",
			"margin-right": "auto",
			"display": "block"
		}).append(
			$("<tr/>")
				.append("<th>Log level:</th>",
					$("<td/>").append(
						$("<label id='level-fatal-lbl'>Fatal:</label>").append(
							$("<input/>").attr({
								id: "level-fatal-chk",
								type: "checkbox",
								checked: true,
								for: "level-fatal-lbl"
							})
						), " ",
						$("<label id='level-error-lbl'>Error:</label>").append(
							$("<input/>").attr({
								id: "level-error-chk",
								type: "checkbox",
								checked: true,
								for: "level-error-lbl"
							})
						), " ",
						$("<label id='level-warning-lbl'>Warning:</label>").append(
							$("<input/>").attr({
								id: "level-warning-chk",
								type: "checkbox",
								checked: true,
								for: "level-warning-lbl"
							})
						), " ",
						$("<label id='level-info-lbl'>Info:</label>").append(
							$("<input/>").attr({
								id: "level-info-chk",
								type: "checkbox",
								checked: true,
								for: "level-info-lbl"
							})
						)
					)
				),
			"<tr><th>Visible lines:</th><td><span id='log-lines'></span></tr>",
			$("<tr/>").append(
				"<th>Sort</th>",
				$("<td/>").append(
					$("<select/>")
						.attr("id", "log-sort")
						.append(
							"<option id='sort-asc' value='asc'>Ascending</option>",
							"<option id='sort-asc' value='desc'>Descending</option>"
						)
				)
			)
		).insertBefore($log);
	const $filters = $filterTable.find<HTMLInputElement>("input[type=checkbox],select");
	$filters.on("change", () => {
		const filter: LogFilter = {
			showFatal: $filters.filter("#level-fatal-chk").get(0).checked,
			showErrors: $filters.filter("#level-error-chk").get(0).checked,
			showWarnings: $filters.filter("#level-warning-chk").get(0).checked,
			showInfo: $filters.filter("#level-info-chk").get(0).checked,
			sortDesc: $filters.filter("select#log-sort").val() === "desc"
		};
		updateLogFilter($log, filter);
	});
	updateLogFilter($log, {showFatal: true, showErrors: true, showWarnings: true, showInfo: true});
});