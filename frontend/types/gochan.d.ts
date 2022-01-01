import "jquery";

declare interface GochanStyle {
	Name: string;
	Filename: string;
}

// stored in /js/consts.json
declare var styles: StyleSheet[];
declare var defaultStyle: string;
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

/**
 * An object representing a staff member retreived by requesting /manage?action=staffinfo
 */
 interface StaffInfo {
	/**
	 * The staff member's ID in the database
	 */
	ID: number;
	/**
	 * The staff member's username
	 */
	Username: string;
	/**
	 * The staff member's rank.
	 * 0 = not logged in.
	 * 1 = janitor.
	 * 2 = moderator.
	 * 3 = administrator.
	 */
	Rank: number;
}

/**
 * An object representing a management action available to the current staff member
 */
interface StaffAction {
	/**
	 * The GET key used when requesting /manage?action=<id>
	 */
	id?:string;
	/**
	 * The title of the action, to be shown in the staff menu
	 */
	title: string;
	/**
	 * The permission level required to access the action.
	 * 0 = accessible by anyone.
	 * 1 = user needs to be a janitor or higher.
	 * 2 = user needs to be a moderator or higher.
	 * 3 = user needs to be an administrator.
	 */
	perms: number;
	/**
	 * The setting for how the request output is handled.
	 * 0 = never JSON.
	 * 1 = sometimes JSON depending on whether the `json` GET key is set to 1.
	 * 2 = always JSON.
	 */
	jsonOutput: number;
}

/**
 * The result of requestiong /manage?action=actions
 */
interface StaffActionMap {
	/**
	 * The "id" of the action. Retreived by requesting /manage?action=<id>
	 */
	[id:string]: StaffAction;
}

/**
 * The menu shown when the Staff button on the top bar is clicked
 */
declare let $staffMenu: JQuery<HTMLElement>;

/**
 * Defaults to "/"
 */
declare let webroot:string;