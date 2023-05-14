/* eslint no-unused-vars: ["warn", {"args":"none"}] */

import $ from "jquery";

export async function getThreadJSON(threadID: number, board: string) {
	return $.ajax({
		url: `${webroot}${board}/res/${threadID}.json`,
		cache: false,
		dataType: "json",
	});
}