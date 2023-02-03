import $ from "jquery";

import { $topbar, TopBarButton } from "../dom/topbar";
import { currentThread } from "../postinfo";
import { getJsonStorageVal } from "../storage";
import { unwatchThread } from "./watcher";
import { downArrow } from "../vars";

let watcherBtn = null;
/** @type {JQuery<HTMLElement>} */
let $watcherMenu = null;
let numUpdatedThreads = 0; // incremented for each watched thread with new posts, added to the watcher button

function addThreadToMenu(thread) {
	if($watcherMenu.find(`div#thread${thread.id}.watcher-item`).length > 0) {
		// thread is already in menu, check for updates to it
		updateThreadInWatcherMenu(thread);
		return;
	}
	if(thread.op == "") thread.op = "Anonymous";
	let $replyCounter = $("<span/>")
		.prop({id: "reply-counter"})
		.text(`(Replies: ${thread.posts - 1})`);
	let infoElem = ` - <b>OP:</b> ${thread.op}<br/>`;
	if(thread.subject === undefined || thread.subject == "") {
		infoElem += "<b>Subject:</b> <i>[None]</i>";
	} else {
		infoElem += `<b>Subject: </b> ${thread.subject}`;
	}
	let $watcherItem = $("<div/>").prop({
		id: `thread${thread.id}`,
		class: "watcher-item"
	}).append(
		$("<a/>").prop({
			id: "threadlink",
			href: `${webroot}${thread.board}/res/${thread.id}.html`
		}).text(`/${thread.board}/${thread.id}`)," ",
		$replyCounter," ",
		$("<a/>").prop({
			id: `unwatch${thread.id}`,
			href: "javascript:;",
			title: `Unwatch thread #${thread.id}`
		}).on("click", () => {
			unwatchThread(thread.id, thread.board);
		}).text("X"), " "
	);
	if(thread.err !== undefined)
	$watcherItem.append($("<span/>")
		.prop({class: "warning"})
		.text(`(${thread.err})`)
	);
	$watcherMenu.append(
		$watcherItem.append(infoElem)
	);
	$watcherMenu.append($watcherItem);
	$watcherMenu.find("i#no-threads").remove();
}

function removeThreadFromMenu(threadID) {
	$watcherMenu.find(`div#thread${threadID}`).remove();
	if($watcherMenu.find("div.watcher-item").length == 0)
		$watcherMenu.append(`<i id="no-threads">no watched threads</i>`);
}

function updateThreadInWatcherMenu(thread) {
	let currentPage = currentThread();
	
	let $item = $watcherMenu.find(`div#thread${thread.op}`);
	if($item.length == 0) return; // watched thread isn't in the menu
	$item.find("span#reply-counter").remove();
	let $replyCounter = $("<span>").prop({
		id: "reply-counter"
	}).insertBefore($item.find(`a#unwatch${thread.op}`));

	if(currentPage.board == thread.board && currentPage.thread == thread.op) {
		// we're currently in the thread
		$replyCounter.text(` (Replies: ${thread.newNumPosts - 1}) `);
	} else {
		// we aren't currently in the thread, show a link to the first new thread
		$replyCounter.append(
			"(Replies: ", thread.newNumPosts - 1,", ",
			$("<a/>").prop({
				id: "newposts",
				href: `${webroot}${thread.board}/res/${thread.op}.html#${thread.newPosts[0].no}`
			}).text(`${thread.newPosts.length} new`),
			") "
		);
		watcherBtn.button.find(".warning").remove();
		watcherBtn.button.text("Watcher");
		watcherBtn.button.append(
			$("<span/>")
				.prop({class: "warning"})
				.text(`(${++numUpdatedThreads})`),
			downArrow
		);
	}
}

$(() => {
	if($watcherMenu === null) {
		$watcherMenu = $("<div/>").prop({
			id: "watchermenu",
			class: "dropdown-menu"
		}).append(`<b>Watched threads</b><br/><i id="no-threads">no watched threads</i>`);
	}
	if(watcherBtn === null) {
		watcherBtn = new TopBarButton("Watcher", () => {
			$topbar.trigger("menuButtonClick", [$watcherMenu, $(document).find($watcherMenu).length == 0]);
		}, {
			before: "a#settings.dropdown-button"
		});
	}
	$(document)
		.on("watchThread", (_e,thread) => addThreadToMenu(thread))
		.on("unwatchThread", (_e, threadID) => removeThreadFromMenu(threadID))
		.on("watcherNewPosts", (_e, thread) => updateThreadInWatcherMenu(thread))
		.on("beginNewPostsCheck", () => {
			numUpdatedThreads = 0;
		});
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);
	for(const board of boards) {
		for(const thread of watched[board]) {
			addThreadToMenu(thread);
		}
	}
});