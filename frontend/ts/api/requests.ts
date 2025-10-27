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
	const data:BoardsJSON|void = await fetch(`${webroot}boards.json`)
		.then<BoardsJSON>(response => response.json())
		.catch(() => {});
	if(data) {
		return { boards: data.boards, currentBoard: currentBoard() };
	} else {
		return nullBoardsList;
	}
}

export async function getCatalog(board = "") {
	const useBoard = (board !== "")?board:currentBoard();
	const data:CatalogBoard[] = await fetch(`${webroot}${useBoard}/catalog.json`)
		.then(response => response.json())
		.catch((reason):CatalogBoard[] => {
			console.error(`Error getting catalog for /${board}/: ${reason}`);
			return [];
		});

	if(data.length === 0)
		return [];
	if(data[0] === null)
		data.shift();
	return data;
}

export async function getThread(board = "", thread = 0) {
	const threadInfo = currentThread();
	if(board !== "")
		threadInfo.board = board;
	if(thread > 0)
		threadInfo.id = thread;

	if(threadInfo.board === "") {
		return Promise.reject("not in a board");
	}
	if(threadInfo.id < 1) {
		return Promise.reject("not in a thread");
	}

	const data = await fetch(`${webroot}${threadInfo.board}/res/${threadInfo.id}.json`, {
		method: "GET",
		cache: "no-cache"
	}).then(response => response.json()).catch((reason):null => {
		console.error(`Error getting catalog for /${threadInfo.board}/: ${reason}`);
		return null;
	});
	return data;
}