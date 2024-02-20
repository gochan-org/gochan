/* eslint no-var: 0  */

import "jquery";

declare global {
	interface GochanStyle {
		Name: string;
		Filename: string;
	}

	// stored in /js/consts.json
	var styles: GochanStyle[];
	var defaultStyle: string;
	var serverTZ: number;
	/**
	 * Defaults to "/"
	 */
	var webroot: string;
	interface Window {
		$: JQueryStatic;
		jQuery: JQueryStatic;
		styles: GochanStyle[];
		defaultStyle: string;
		webroot: string;
		serverTZ: number;

		openQR: () => void;
		closeQR: () => void;
		toTop: () => void;
		toBottom: () => void;
		quote: (no: number) => void;
	}

	// /boards.json
	interface BoardsJSON {
		boards: BoardJSON[];
	}

	interface BoardCooldowns {
		threads: number;
		replies: number
		images: number;
	}

	interface BoardJSON {
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
	interface CatalogBoard {
		page: number;
		threads: CatalogThread[];
	}

	interface CatalogThread {
		replies: number;
		images: number;
		omitted_posts: number;
		omitted_images: number;
		sticky: number;
		locked: number;
	}

	// /boarddir/res/#.json
	interface BoardThread {
		posts: ThreadPost[];
	}

	interface ThreadPost {
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

	interface PostSubmitResponse {
		error?: string;
		id: number;
		time: Date;
		thread: string;
	}

	interface PostInfoPost {
		ID: number;
		ThreadID: number;
		IsTopPost: boolean;
		IP: string;
		CreatedOn: string;
		Name: string;
		Tripcode: string;
		IsRoleSignature: boolean;
		Email: string;
		Subject: string;
		Message: string;
		MessageRaw: string;
		DeletedAt: string;
		IsDeleted: boolean;
		BannedMessage: string;
		Flag: string;
		Country: string;
	}

	/**
	 * Returned by /manage/postinfo?postid=#
	 */
	interface PostInfo {
		post: PostInfoPost;
		ip: string;
		ipFQDN: string[];
		originalFilename?: string;
		checksum?: string;
		fingerprint?: string;
	}

	/**
	 * An object representing the settings for fingerprinting images and if enabled,
	 * video thumbnails
	 */
	interface FingerprintingOptions {
		/**
		 * If true, allow fingerprinting of video thumbnails
		 */
		fingerprintVideoThumbs: boolean;
		/**
		 * A list of file extensions for images that gochan is presumed to be able to
		 * thumbnail
		 */
		imageExtensions: string[];
		/**
		 * A list of file extensions for videos
		 */
		videoExtensions: string[];
	}

	/**
	 * An object representing a staff member retreived by requesting /manage/staffinfo
	 */
	interface StaffInfo {
		/**
		 * The staff member's username
		 */
		username: string;
		/**
		 * The staff member's rank.
		 * 0 = not logged in.
		 * 1 = janitor.
		 * 2 = moderator.
		 * 3 = administrator.
		 */
		rank: number;

		actions?: StaffAction[]

		fingerprinting?: FingerprintingOptions;
	}

	/**
	 * An object representing a management action available to the current staff member
	 */
	interface StaffAction {
		/**
		 * The GET key used when requesting /manage/<id>
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
	 * The result of requesting /manage/actions
	 */
	var staffActions: StaffAction[];

	/**
	 * The menu shown when the Staff button on the top bar is clicked
	 */
	let $staffMenu: JQuery<HTMLElement>;

	// used for testing
	var simpleHTML: string;
}