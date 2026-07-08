function getCooldown(data: BoardsJSON, board: string, type: "threads"|"replies"|"images"): number {
	return (data.boards.find(boardData => boardData.board === board)?.cooldowns as BoardCooldowns)[type] ?? 0;
}

export async function getThreadCooldown(board: string) {
	const boards:BoardsJSON = await fetch(`${webroot}boards.json`).then(response => response.json());
	return getCooldown(boards, board, "threads");
}

export async function getReplyCooldown(board: string) {
	const boards:BoardsJSON = await fetch(`${webroot}boards.json`).then(response => response.json());
	return getCooldown(boards, board, "replies");
}