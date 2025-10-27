/* eslint no-unused-vars: ["warn", {"args":"none"}] */

export async function getThreadJSON(threadID: number, board: string):Promise<BoardThread> {
	return await fetch(`${webroot}${board}/res/${threadID}.json`, {
		method: "GET",
		cache: "no-cache"
	}).then(response => response.json());
}