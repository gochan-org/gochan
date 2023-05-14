/* eslint no-unused-vars: ["warn", {"args":"none"}] */

import $ from "jquery";

import { currentBoard, currentThread } from "../postinfo";

interface BoardsList {
	boards: any[];
	currentBoard: string;
}

const nullBoardsList: BoardsList = {
	boards: [],
	currentBoard: ""
};

export async function getBoardList() {
	try {
		const data = await $.ajax({
			url: webroot + "boards.json",
			cache: false,
			dataType: "json",
			success: (d2 => {}),
			error: function(_err, _status, statusText) {
				console.error("Error getting board list: " + statusText);
				return nullBoardsList;
			},
		});
		return { boards: data.boards, currentBoard: currentBoard() };
	} catch(e) {
		return nullBoardsList;
	}
}

export async function getCatalog(board = "") {
	let useBoard = (board != "")?board:currentBoard();

	const data = await $.ajax({
		url: webroot + useBoard + "/catalog.json",
		cache: false,
		dataType: "json",
		success: (() => { }),
		error: function (err, status, statusText) {
			console.error(`Error getting catalog for /${board}/: ${statusText}`);
		}
	});
	if(data.length === 0)
		return [];
	if(data[0] === null)
		data.shift();
	return data;
}

export async function getThread(board = "", thread = 0) {
	let threadInfo = currentThread();
	if(board != "")
		threadInfo.board = board;
	if(thread > 0)
		threadInfo.id = thread;
	
	if(threadInfo.board === "") {
		return Promise.reject("not in a board");
	}
	if(threadInfo.id < 1) {
		return Promise.reject("not in a thread");
	}

	const data = await $.ajax({
		url: `${webroot}${threadInfo.board}/res/${threadInfo.id}.json`,
		cache: false,
		dataType: "json",
		error: function (err, status, statusText) {
			console.error(`Error getting catalog for /${board}/: ${statusText}`);
		}
	});
	return data;
}