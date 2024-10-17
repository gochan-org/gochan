-- SQL views for simplifying queries in gochan

-- First drop views if they exist in reverse order to avoid dependency issues
DROP VIEW IF EXISTS DBPREFIXv_front_page_posts_with_file;
DROP VIEW IF EXISTS DBPREFIXv_front_page_posts;
DROP VIEW IF EXISTS DBPREFIXv_posts_to_delete_file_only;
DROP VIEW IF EXISTS DBPREFIXv_posts_to_delete;
DROP VIEW IF EXISTS DBPREFIXv_recent_posts;
DROP VIEW IF EXISTS DBPREFIXv_building_posts;
DROP VIEW IF EXISTS DBPREFIXv_top_posts;

-- create views
CREATE VIEW DBPREFIXv_top_posts AS
SELECT id, thread_id FROM DBPREFIXposts WHERE is_top_post;


CREATE VIEW DBPREFIXv_building_posts AS
SELECT DBPREFIXposts.id AS id, DBPREFIXposts.thread_id AS thread_id, ip, name, tripcode,
email, subject, created_on, created_on as last_modified, p.id AS parent_id, t.last_bump as last_bump,
message, message_raw,
(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id LIMIT 1) AS dir,
coalesce(DBPREFIXfiles.original_filename, '') as original_filename,
coalesce(DBPREFIXfiles.filename, '') AS filename,
coalesce(DBPREFIXfiles.checksum, '') AS checksum,
coalesce(DBPREFIXfiles.file_size, 0) AS filesize,
coalesce(DBPREFIXfiles.thumbnail_width, 0) AS tw,
coalesce(DBPREFIXfiles.thumbnail_height, 0) AS th,
coalesce(DBPREFIXfiles.width, 0) AS width,
coalesce(DBPREFIXfiles.height, 0) AS height,
t.locked as locked,
t.stickied as stickied,
t.cyclical as cyclical,
flag, country
FROM DBPREFIXposts
LEFT JOIN DBPREFIXfiles ON DBPREFIXfiles.post_id = DBPREFIXposts.id AND is_deleted = FALSE
LEFT JOIN (
    SELECT id, board_id, last_bump, locked, stickied, cyclical FROM DBPREFIXthreads
) t ON t.id = DBPREFIXposts.thread_id
INNER JOIN DBPREFIXv_top_posts p ON p.thread_id = DBPREFIXposts.thread_id
WHERE is_deleted = FALSE;


CREATE VIEW DBPREFIXv_recent_posts AS
SELECT p.*, b.id as board_id FROM DBPREFIXv_building_posts p
left join DBPREFIXboards b;


CREATE VIEW DBPREFIXv_posts_to_delete AS
SELECT p.id AS postid, (
    SELECT op.id AS opid FROM DBPREFIXposts op
    WHERE op.thread_id = p.thread_id AND is_top_post LIMIT 1
) as opid, is_top_post, COALESCE(filename, "") AS filename, dir
FROM DBPREFIXboards b
LEFT JOIN DBPREFIXthreads t ON t.board_id = b.id
LEFT JOIN DBPREFIXposts p ON p.thread_id = t.id
LEFT JOIN DBPREFIXfiles f ON f.post_id = p.id;


CREATE VIEW DBPREFIXv_posts_to_delete_file_only AS
SELECT * FROM DBPREFIXv_posts_to_delete
WHERE filename IS NOT NULL;


CREATE VIEW DBPREFIXv_front_page_posts AS
SELECT DBPREFIXposts.id, DBPREFIXposts.message_raw,
(SELECT dir FROM DBPREFIXboards WHERE id = t.board_id) as dir,
COALESCE(f.filename, '') as filename, op.id as opid
FROM DBPREFIXposts
LEFT JOIN (SELECT id, board_id FROM DBPREFIXthreads) t ON t.id = DBPREFIXposts.thread_id
LEFT JOIN (SELECT post_id, filename FROM DBPREFIXfiles) f on f.post_id = DBPREFIXposts.id
INNER JOIN (SELECT id, thread_id FROM DBPREFIXposts WHERE is_top_post) op ON op.thread_id = DBPREFIXposts.thread_id
WHERE DBPREFIXposts.is_deleted = FALSE;


CREATE VIEW DBPREFIXv_front_page_posts_with_file AS
SELECT * FROM DBPREFIXv_front_page_posts
WHERE filename IS NOT NULL AND filename != '' AND filename != 'deleted';