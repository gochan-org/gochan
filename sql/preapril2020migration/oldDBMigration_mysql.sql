/*
Migrates the pre-refactor database of april 2020 to database verion 1

rename all tables to table_old
run version 1 population script
filter, copy change data
*/

INSERT INTO DBPREFIXsections (id, name, abbreviation, position, hidden)
SELECT id, name, abbreviation, list_order, hidden > 0
FROM DBPREFIXsections_old;

INSERT INTO DBPREFIXboards (
	id, 
	section_id, 
	uri, 
	dir, 
	navbar_position,
	title,
	subtitle,
	description,
	max_file_size,
	max_threads,
	default_style,
	locked,
	created_at,
	anonymous_name,
	force_anonymous,
	autosage_after,
	no_images_after,
	max_message_length,
	min_message_length,
	allow_embeds,
	redirect_to_thread,
	require_file,
	enable_catalog)
SELECT id, section, dir, dir, list_order, title, subtitle, description, max_file_size, 1000,
default_style, locked, created_on, anonymous, forced_anon, autosage_after, no_images_after, max_message_length, 0, embeds_allowed, 
redirect_to_thread, require_file, enable_catalog
FROM DBPREFIXboards_old;

/*
---Migrating posts
add oldpostid to thread table, make it unique
add oldselfid, oldparentid and oldboardid to posts table
add oldselfid and oldboardid to files
remove foreign key constraint on thread_id in posts
remove foreign key constraint on post_id in files
create thread per top post, add id of said old post to oldpostid
insert top posts with old parent id being own old id and old board id being board id
insert child posts with old parent id being old parent post id and old board id being board id
---
	UPDATE posts, threads
	SET posts.thread_id = threads.id
	WHERE threads.oldpostid = posts.oldparentid AND threads.boardid = posts.oldboardid
---
insert into files values where file values exist
---
	UPDATE posts, files
	SET files.post_id = posts.id
	WHERE files.oldpostid = posts.oldselfid AND files.oldboardid = posts.oldboardid
---
remove all dummy columns
add foreign key constraint on thread_id in posts
add foreign key constraint on post_id in files
*/

ALTER TABLE DBPREFIXthreads ADD oldpostid int;
ALTER TABLE DBPREFIXposts ADD oldselfid int;
ALTER TABLE DBPREFIXposts ADD oldparentid int;
ALTER TABLE DBPREFIXposts ADD oldboardid int;
ALTER TABLE DBPREFIXposts DROP FOREIGN KEY thread_id;
ALTER TABLE DBPREFIXfiles ADD oldpostid int;
ALTER TABLE DBPREFIXfiles ADD oldboardid int;
ALTER TABLE DBPREFIXfiles DROP FOREIGN KEY post_id;

INSERT INTO DBPREFIXthreads(board_id, locked, stickied, anchored, last_bump, is_deleted, deleted_at, oldpostid)
SELECT boardid, locked, stickied, autosage, bumped, deleted_timestamp > '2000-01-01', deleted_timestamp, id FROM DBPREFIXposts_old WHERE parentid = 0;

INSERT INTO DBPREFIXposts(is_top_post, ip, created_on, name, tripcode, is_role_signature, email, subject, message, message_raw, password, deleted_at, is_deleted, oldparentid, oldboardid)
SELECT parentid = 0, ip, timestamp, name, tripcode, false, email, subject, 
	message, message_raw, password, deleted_timestamp, deleted_timestamp > '2000-01-01', CASE WHEN parentid = 0 THEN id ELSE parentid, boardid from DBPREFIXposts_old;

UPDATE DBPREFIXposts as posts, DBPREFIXthreads as threads
SET posts.thread_id = thread.id
WHERE threads.oldpostid = posts.oldparentid AND thread.board_id = posts.oldboardid;

INSERT INTO DBPREFIXfiles(file_oder, original_filename, filename, checksum, file_size, is_spoilered, width, height, thumbnail_width, thumbnail_height, oldpostid, oldboardid)
SELECT 1, filename_original, filename, file_checksum, filesize, false, image_w, image_h, thumb_w, thumb_h, id, boardid FROM DBPREFIXposts_old WHERE filename != '';

UPDATE DBPREFIXfiles as files, DBPREFIXposts as posts
SET files.post_id = posts.id
WHERE files.oldpostid = posts.oldselfid AND files.oldboardid = posts.oldboardid;

