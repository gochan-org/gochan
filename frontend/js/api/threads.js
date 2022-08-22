/* eslint no-unused-vars: ["warn", {"args":"none"}] */
/* global webroot */

import $ from "jquery";

export function getThreadJSON(threadID, board) {
	return $.ajax({
		url: `${webroot}${board}/res/${threadID}.json`,
		cache: false,
		dataType: "json",
		error: function(e, status, statusText) {
			// clearInterval(threadWatcherInterval);
			return {};
		}
	}).catch(e => {
		return {};
	});
}