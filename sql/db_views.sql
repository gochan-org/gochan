-- SQL views for simplifying queries in gochan

-- First drop views if they exist in reverse order to avoid dependency issues
DROP VIEW IF EXISTS DBPREFIXv_recent_posts;
DROP VIEW IF EXISTS DBPREFIXv_board_top_posts;
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

CREATE VIEW DBPREFIXv_board_top_posts AS
SELECT * FROM DBPREFIXv_building_posts
WHERE id = parent_id AND ORDER BY t.stickied DESC, last_bump DESC

CREATE VIEW DBPREFIXv_recent_posts AS
SELECT * FROM DBPREFIXv_building_posts
ORDER BY id DESC;