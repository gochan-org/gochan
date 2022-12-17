/*
Migrates the pre-refactor database of april 2020 to database verion 1

rename all tables to table_old
run version 1 population script
filter, copy change data

A deactivated, unauthorized user with the name "unknown_staff" is created to attribute to unknown staff
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
ALTER TABLE DBPREFIXposts DROP CONSTRAINT posts_thread_id_fk;
ALTER TABLE DBPREFIXfiles ADD oldpostid int;
ALTER TABLE DBPREFIXfiles ADD oldboardid int;
ALTER TABLE DBPREFIXfiles DROP CONSTRAINT files_post_id_fk;

INSERT INTO DBPREFIXthreads(board_id, locked, stickied, anchored, last_bump, is_deleted, deleted_at, oldpostid)
SELECT boardid, locked, stickied, autosage, bumped, deleted_timestamp > '2000-01-01', deleted_timestamp, id FROM DBPREFIXposts_old WHERE parentid = 0;

INSERT INTO DBPREFIXposts(is_top_post, ip, created_on, name, tripcode, is_role_signature, email, subject, message, message_raw, password, deleted_at, is_deleted, oldparentid, oldboardid, oldselfid)
SELECT parentid = 0, ip, timestamp, name, tripcode, false, email, subject, 
	message, message_raw, password, deleted_timestamp, deleted_timestamp > '2000-01-01', id, boardid, id from DBPREFIXposts_old WHERE parentid = 0;

INSERT INTO DBPREFIXposts(is_top_post, ip, created_on, name, tripcode, is_role_signature, email, subject, message, message_raw, password, deleted_at, is_deleted, oldparentid, oldboardid, oldselfid)
SELECT parentid = 0, ip, timestamp, name, tripcode, false, email, subject, 
	message, message_raw, password, deleted_timestamp, deleted_timestamp > '2000-01-01', parentid, boardid, id from DBPREFIXposts_old WHERE parentid <> 0;



INSERT INTO DBPREFIXfiles(file_order, original_filename, filename, checksum, file_size, is_spoilered, width, height, thumbnail_width, thumbnail_height, oldpostid, oldboardid)
SELECT 1, filename_original, filename, file_checksum, filesize, false, image_w, image_h, thumb_w, thumb_h, id, boardid FROM DBPREFIXposts_old WHERE filename <> '' AND filename <> 'deleted';



ALTER TABLE DBPREFIXthreads DROP COLUMN oldpostid;
ALTER TABLE DBPREFIXposts DROP COLUMN oldparentid;
ALTER TABLE DBPREFIXposts DROP COLUMN oldboardid;
ALTER TABLE DBPREFIXfiles DROP COLUMN oldpostid;
ALTER TABLE DBPREFIXfiles DROP COLUMN oldboardid;
ALTER TABLE DBPREFIXfiles ADD CONSTRAINT files_post_id_fk FOREIGN KEY (post_id) REFERENCES DBPREFIXposts(id);
ALTER TABLE DBPREFIXposts ADD CONSTRAINT posts_thread_id_fk FOREIGN KEY (thread_id) REFERENCES DBPREFIXthreads(id);



--Staff
INSERT INTO DBPREFIXstaff(id, username, password_checksum, global_rank, added_on, last_login)
SELECT id, username, password_checksum, rank, added_on, last_active
FROM DBPREFIXstaff_old;

--Bans--

--Step 1, normalisation from comma seperated boards to seperate entries per board

--Create copy of table structure and drop not null constraint on boards

--needed because id sequence is otherwise shared between this and the old table
ALTER TABLE DBPREFIXbanlist_old_normalized DROP COLUMN id;
ALTER TABLE DBPREFIXbanlist_old_normalized ADD COLUMN old_id int; 

/*
Joins every ban on every entry in the numbers list (lists 1 to 1000 ints), then filters to only return results where numbers.num <= #elementsInCommaList
Cuts out element in list using the num as index
*/
INSERT INTO DBPREFIXbanlist_old_normalized(old_id, allow_read, ip, name, name_is_regex, filename, file_checksum, staff, timestamp, expires, permaban, reason, type, staff_note, appeal_at, can_appeal, boards)
(SELECT
	bans.id,
	bans.allow_read,
	bans.ip,
	bans.name,
	bans.name_is_regex,
	bans.filename,
	bans.file_checksum,
	bans.staff,
	bans.timestamp,
	bans.expires,
	bans.permaban,
	bans.reason,
	bans.type,
	bans.staff_note,
	bans.appeal_at,
	bans.can_appeal,
FROM
DBPREFIXnumbersequel_temp AS nums INNER JOIN DBPREFIXbanlist_old AS bans
ON TRUE
WHERE CHAR_LENGTH(bans.boards)-CHAR_LENGTH(REPLACE(bans.boards, ',', '')) >= nums.num-1);

--replace * with null
UPDATE DBPREFIXbanlist_old_normalized
SET boards = null
WHERE boards = '*';

ALTER TABLE DBPREFIXbanlist_old_normalized ADD COLUMN board_id int;
ALTER TABLE DBPREFIXbanlist_old_normalized ADD COLUMN staff_id int;

--Fix board id


--Fix staff_id


--Step 2, copy them all to the live ban board per ban type

/*
ban types:
1 = thread ban
2 = image ban
3 = full ban
if image <> "" or null create image ban
*/
--Add an old_id column to each table for later foreign key linking for appeals, remove at the end
ALTER TABLE DBPREFIXip_ban ADD COLUMN old_id int;

--ip bans
INSERT INTO DBPREFIXip_ban(old_id, staff_id, board_id, is_thread_ban, is_active, ip, issued_at, expires_at, permanent, staff_note, message, can_appeal, appeal_at, copy_post_text)
(
	SELECT old_id, staff_id, board_id, TRUE, TRUE, ip, timestamp, expires, permaban, staff_note, reason, can_appeal, appeal_at, ''
	FROM DBPREFIXbanlist_old_normalized WHERE type = 1
);

INSERT INTO DBPREFIXip_ban(old_id, staff_id, board_id, is_thread_ban, is_active, ip, issued_at, expires_at, permanent, staff_note, message, can_appeal, appeal_at, copy_post_text)
(
	SELECT old_id, staff_id, board_id, FALSE, TRUE, ip, timestamp, expires, permaban, staff_note, reason, can_appeal, appeal_at, ''
	FROM DBPREFIXbanlist_old_normalized WHERE type = 3
);

--appeals
INSERT INTO DBPREFIXip_ban_appeals(ip_ban_id, appeal_text, staff_response, is_denied)(
	SELECT ban.id, appeal.message, appeal.staff_response, appeal.denied
	FROM DBPREFIXappeals_old as appeal
	JOIN DBPREFIXip_ban as ban ON ban.old_id = appeal.id
);

ALTER TABLE DBPREFIXip_ban DROP COLUMN old_id;

--file ban
INSERT INTO DBPREFIXfile_ban(board_id, staff_id, staff_note, issued_at, checksum)(
	SELECT board_id, staff_id, staff_note, timestamp, file_checksum 
	FROM DBPREFIXbanlist_old_normalized WHERE file_checksum <> ''
);

--filename ban
INSERT INTO DBPREFIXfilename_ban(board_id, staff_id, staff_note, issued_at, filename, is_regex)(
	SELECT board_id, staff_id, staff_note, timestamp, filename, name_is_regex 
	FROM DBPREFIXbanlist_old_normalized WHERE filename <> ''
);

--username ban
INSERT INTO DBPREFIXusername_ban(board_id, staff_id, staff_note, issued_at, username, is_regex)(
	SELECT board_id, staff_id, staff_note, timestamp, name, name_is_regex 
	FROM DBPREFIXbanlist_old_normalized WHERE name <> ''
);

--reports
INSERT INTO DBPREFIXreports(post_id, ip, reason, is_cleared)(
	SELECT post.id, report.id, report.reason, report.cleared
	FROM DBPREFIXreports_old as report
	JOIN DBPREFIXposts as post on post.oldselfid = report.postid
);
