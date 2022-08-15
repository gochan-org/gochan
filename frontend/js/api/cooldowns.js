/* global webroot */

import $ from "jquery";

/**
 * @param {string} board
 * @param {string} type
 * @returns {number}
 */
function getCooldown(data, board, type) {
	for(const boardData of data.boards) {
		if(boardData.board != board) continue;
		return boardData.cooldowns[type];
	}
}

export async function getThreadCooldown(board) {
	const boards = await $.getJSON(`${webroot}boards.json`);
	return getCooldown(boards, board, "threads");
}

export async function getReplyCooldown(board) {
	const boards = await $.getJSON(`${webroot}boards.json`);
	return getCooldown(boards, board, "replies");
}