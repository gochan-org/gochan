import $ from "jquery";

interface LogFilter {
	showFatal: boolean;
	showErrors: boolean;
	showWarnings: boolean;
	showInfo: boolean;
}

let originalLog = "";

function updateLogFilter($log: JQuery<HTMLTextAreaElement>, filter: LogFilter) {
	let lines = originalLog.split("\n").filter((line) => {
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
				console.log(lineObj);
			}
			return false;
		} catch(_) {
			return false;
		}
	})

	$log.text(lines.join("\n"));
	$("span#log-lines").text(lines.length);
}

$(() => {
	if(location.pathname.indexOf(webroot + "manage/viewlog") != 0)
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
			"<tr/><th>Visible lines:</th><td><span id='log-lines'></span>"
		).insertBefore($log);
	const $filterChecks = $filterTable.find<HTMLInputElement>("input[type=checkbox]");
	$filterChecks.on("change", () => {
		const filter: LogFilter = {
			showFatal: $filterChecks.filter("#level-fatal-chk")[0].checked,
			showErrors: $filterChecks.filter("#level-error-chk")[0].checked,
			showWarnings: $filterChecks.filter("#level-warning-chk")[0].checked,
			showInfo: $filterChecks.filter("#level-info-chk")[0].checked,
		};
		updateLogFilter($log, filter);
	});
	updateLogFilter($log, {showFatal: true, showErrors: true, showWarnings: true, showInfo: true});
});