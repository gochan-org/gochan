/* eslint no-unused-vars: ["warn", {"args":"none"}] */

import $ from "jquery";

export async function getThreadJSON(threadID, board) {
	return $.ajax({
		url: `${webroot}${board}/res/${threadID}.json`,
		cache: false,
		dataType: "json",
	});
}