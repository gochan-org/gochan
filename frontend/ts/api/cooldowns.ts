import $ from "jquery";

/**
 * @param {string} board
 * @param {string} type
 * @returns {number}
 */
function getCooldown(data: BoardsJSON, board: string, type: string) {
	for(const boardData of data.boards) {
		if(boardData.board != board) continue;
		return (boardData.cooldowns as any)[type];
	}
}

export async function getThreadCooldown(board: string) {
	const boards = await $.getJSON(`${webroot}boards.json`);
	return getCooldown(boards, board, "threads");
}

export async function getReplyCooldown(board: string) {
	const boards = await $.getJSON(`${webroot}boards.json`);
	return getCooldown(boards, board, "replies");
}