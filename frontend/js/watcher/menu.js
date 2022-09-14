/* global webroot */

import $ from "jquery";

import { $topbar, TopBarButton } from "../dom/topbar";
import { getJsonStorageVal } from "../storage";
import { unwatchThread } from "./watcher";

let watcherBtn = null;
/** @type {JQuery<HTMLElement>} */
let $watcherMenu = null;


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
			href: `${webroot}${thread.board}/res/${thread.id}.html`
		}).css({
			"font-weight": "bold"
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
		.css({color: "red"})
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
	let $item = $watcherMenu.find(`div#thread${thread.op}`);
	if($item.length == 0) return; // watched thread isn't in the menu
	$item.find("span#reply-counter").remove();
	$("<span>").prop({
		id: "reply-counter"
	}).append(
		"(Replies: ", thread.newNumPosts - 1,", ",
		$("<a/>").prop({
			href: `${webroot}${thread.board}/res/${thread.op}.html#${thread.newPosts[0].no}`
		}).css({
			"font-weight": "bold"
		}).text(`${thread.newPosts.length} new`),
		") "
	).insertBefore($watcherMenu.find(`a#unwatch${thread.op}`));
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
		.on("watcherNewPosts", (_e, thread) => updateThreadInWatcherMenu(thread));
	let watched = getJsonStorageVal("watched", {});
	let boards = Object.keys(watched);
	for(const board of boards) {
		for(const thread of watched[board]) {
			addThreadToMenu(thread);
		}
	}
});