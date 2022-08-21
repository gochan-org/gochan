/* global webroot */

import { currentBoard, currentThread } from "../postinfo";

const nullBoardsList = {
	boards: [],
	currentBoard: ""
}

export async function getBoardList() {
	try {
		const data = await $.ajax({
			url: webroot + "boards.json",
			cache: false,
			dataType: "json",
			success: (d2 => {}),
			error: function(err, status, statusText) {
				console.error("Error getting board list: " + statusText);
				return nullBoardsList;
			},
		});
		return { boards: data.boards, currentBoard: currentBoard() };
	} catch (e) {
		return nullBoardsList;
	}
}

export async function getCatalog(board = "") {
	let useBoard = (board != "")?board:currentBoard();

	const data = await $.ajax({
		url: webroot + useBoard + "/catalog.json",
		cache: false,
		dataType: "json",
		success: (d2_1 => { }),
		error: function (err, status, statusText) {
			console.error(`Error getting catalog for /${board}/: ${statusText}`);
		}
	});
	if (data.length === 0)
		return [];
	if (data[0] === null)
		data.shift();
	return data;
}

export async function getThread(board = "", thread = 0) {
	let threadInfo = currentThread();
	if(board != "")
		threadInfo.board = board;
	if(thread > 0)
		threadInfo.thread = thread;
	
	if(threadInfo.board === "") {
		return Promise.reject("not in a board");
	}
	if(threadInfo.thread < 1) {
		return Promise.reject("not in a thread");
	}

	const data = await $.ajax({
		url: `${webroot}${threadInfo.board}/res/${threadInfo.thread}.json`,
		cache: false,
		dataType: "json",
		error: function (err, status, statusText) {
			console.error(`Error getting catalog for /${board}/: ${statusText}`);
		}
	});
	return data;
}