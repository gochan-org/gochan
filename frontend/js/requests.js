/* global webroot */

import { currentBoard, currentThread } from "./postutil";

export function getBoardList() {
	return fetch(webroot + "boards.json")
	.then(response => {
		if(!response.ok)
			throw new Error(response.statusText);
		return response.json();
	}).then((/** @type {BoardsJSON} */ data) => {
		return {boards: data.boards, currentBoard: currentBoard()};
	});
}

export function getCatalog(board = "") {
	let useBoard = (board != "")?board:currentBoard();

	return fetch(webroot + useBoard + "/catalog.json")
		.then(response => {
			if(!response.ok)
				throw new Error(response.statusText);
			return response.json();
		}).then((/** @type {CatalogBoard[]} */ data) => {
			if(data.length == 0)
				return [];
			if(data[0] == null)
				data.shift();
			return data;
		});
}

export function getThread(board = "", thread = 0) {
	let threadInfo = currentThread();
	if(board != "")
		threadInfo.board = board;
	if(thread > 0)
		threadInfo.thread = thread;
	
	if(threadInfo.board == "" || threadInfo.thread < 1) 
		// return a Promise with the board info
		return new Promise((resolve, reject) => {
			reject(threadInfo);
		});

	return fetch(`${webroot}${threadInfo.board}/res/${threadInfo.thread}.json`)
		.then(response => {
			if(!response.ok)
				throw new Error(response.statusText);
			return response.json();
		}).then((/** @type {BoardThread[]} */ data) => {
			return data;
		});
}