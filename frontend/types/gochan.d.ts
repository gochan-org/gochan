declare interface GochanStyle {
	Name: string;
	Filename: string;
}

// stored in /js/consts.json
declare var styles: StyleSheet[];
declare var defaultStyle: string;
declare var webroot: string;
declare var serverTZ: number;


// /boards.json
declare interface BoardsJSON {
	boards: BoardJSON[];
}

declare interface BoardCooldowns {
	threads: number;
	replies: number
	images: number;
}

declare interface BoardJSON {
	pages: number;
	board: string;
	title: string;
	meta_description: string;
	max_filesize: number;
	max_pages: number;
	is_archived: boolean;
	bump_limit: number;
	image_limit: number;
	max_comment_chars: number;
	ws_board: boolean;
	cooldowns: BoardCooldowns
	per_page: number;
}

// an array of these are in /boarddir/catalog.json
declare interface CatalogBoard {
	page: number;
	threads: CatalogThread[];
}

declare interface CatalogThread {
	replies: number;
	images: number;
	omitted_posts: number;
	omitted_images: number;
	sticky: number;
	locked: number;
}

// /boarddir/res/#.json
declare interface BoardThread {
	posts: ThreadPost[];
}

declare interface ThreadPost {
	no: number;
	resto: number;
	name: string; 
	trip: string;
	email: string;
	sub: string;
	com: string;
	tim: string;
	filename: string;
	md5: string;
	extension: string;
	fsize: number;
	w: number;
	h: number;
	tn_w: number;
	tn_h: number;
	capcode: string;
	time: string;
	last_modified: string;
}
