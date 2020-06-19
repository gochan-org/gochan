-- Gochan PostgreSQL/SQLite startup/update script
-- DO NOT DELETE

INSERT INTO DBPREFIXannouncements (message, poster) VALUES ('announcement message 1', 'i am the poster');
INSERT INTO DBPREFIXannouncements (message, poster) VALUES ('announcement message 2', 'i am the poster two');

INSERT INTO DBPREFIXappeals (ban, message, denied, staff_response) VALUES (1, 'one', false, 'staff response');
INSERT INTO DBPREFIXappeals (ban, message, denied, staff_response) VALUES (2, 'two', true, 'staff response2');

INSERT INTO DBPREFIXbanlist (ip, staff_note, boards) VALUES ('IP 1', 'staff note 1', '*');
INSERT INTO DBPREFIXbanlist (ip, staff_note, boards) VALUES ('IP 2', 'staff note 2', 'board1,  board2,  board3');
INSERT INTO DBPREFIXbanlist (ip, staff_note, boards) VALUES ('IP 3', 'staff note 3', 'board4');

INSERT INTO DBPREFIXboards (dir, title) VALUES ('a', 'anime');
INSERT INTO DBPREFIXboards (dir, title) VALUES ('d', 'fucked up hentai');
INSERT INTO DBPREFIXboards (dir, title) VALUES ('k', 'guns');

INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp)
VALUES (1, 1, 0, 'name', 'trip', 'email', 'subject', 'message1', 'message raw', 'password', CURRENT_TIMESTAMP);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp)
VALUES (2, 1, 1, 'name', 'trip', 'email', 'subject', 'message2', 'message raw', 'password', CURRENT_TIMESTAMP);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp)
VALUES (3, 2, 0, 'name', 'trip', 'email', 'subject', 'message3', 'message raw', 'password', CURRENT_TIMESTAMP);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp)
VALUES (4, 2, 3, 'name', 'trip', 'email', 'subject', 'message4', 'message raw', 'password', CURRENT_TIMESTAMP);

INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp, filename, filename_original, file_checksum, filesize, image_w, image_h, thumb_w, thumb_h)
VALUES (5, 1, 0, 'name', 'trip', 'email', 'subject', 'message5', 'message raw', 'password', CURRENT_TIMESTAMP, 'Filename', 'original filename', 'checksum', 11, 1,2,3,4);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp, filename, filename_original, file_checksum, filesize, image_w, image_h, thumb_w, thumb_h)
VALUES (6, 1, 5, 'name', 'trip', 'email', 'subject', 'message6', 'message raw', 'password', CURRENT_TIMESTAMP, 'Filename2', 'original filename2', 'checksum2', 12, 5,6,7,8);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp, filename, filename_original, file_checksum, filesize, image_w, image_h, thumb_w, thumb_h)
VALUES (7, 2, 0, 'name', 'trip', 'email', 'subject', 'message7', 'message raw', 'password', CURRENT_TIMESTAMP, 'Filename3', 'original filename3', 'checksum3', 13, 9,10,11,12);
INSERT INTO DBPREFIXposts (id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, deleted_timestamp, filename, filename_original, file_checksum, filesize, image_w, image_h, thumb_w, thumb_h)
VALUES (8, 2, 7, 'name', 'trip', 'email', 'subject', 'message8', 'message raw', 'password', CURRENT_TIMESTAMP, 'Filename4', 'original filename4', 'checksum4', 14, 13,14,15,16);

INSERT INTO DBPREFIXreports (board, postid, ip, reason) VALUES ('a', 2, 'ip', 'reason');
INSERT INTO DBPREFIXreports (board, postid, ip, reason) VALUES ('k', 1, 'ip', 'reason');

INSERT INTO DBPREFIXsections (name, abbreviation, hidden) VALUES ('section 1', '1', 5);
INSERT INTO DBPREFIXsections (name, abbreviation, hidden) VALUES ('section 2', '2', 1);
INSERT INTO DBPREFIXsections (name, abbreviation, hidden) VALUES ('section 3', '3', 0);

INSERT INTO DBPREFIXsessions (name, sessiondata) VALUES ('staff1', 'sfjdsfjg');
INSERT INTO DBPREFIXsessions (name, sessiondata) VALUES ('staff2', 'afddsgfdfs');

INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES ('staff1', 'sdfdsf', 1);
INSERT INTO DBPREFIXstaff (username, password_checksum, rank) VALUES ('staff2', 'sadffsda', 2);

INSERT INTO DBPREFIXwordfilters (search, change_to, boards) VALUES ('searchterm', 'replace', 'a,k');
INSERT INTO DBPREFIXwordfilters (search, change_to, boards) VALUES ('searchterm2', 'replace2', '*');
