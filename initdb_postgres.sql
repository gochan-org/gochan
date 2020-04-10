-- Gochan postgresql new database script
-- Versioning numbering goes by whole numbers. Upgrade script migrate existing databases between versions
-- Database version: 1

CREATE TABLE database_version(
	version int NOT NULL
);

INSERT INTO database_version(version)
VALUES(1);

CREATE TABLE sections(
	id serial,
	name TEXT NOT NULL,
	abbreviation TEXT NOT NULL,
	position SMALLINT NOT NULL,
	hidden BOOL NOT NULL,
	PRIMARY KEY(id),
	UNIQUE(position)
);

create table boards(
	id serial,
	section_id int NOT NULL,
	uri text NOT NULL,
	dir varchar(45) NOT NULL,
	navbar_position SMALLINT NOT NULL,
	title VARCHAR(45) NOT NULL,
	subtitle VARCHAR(64) NOT NULL,
	description VARCHAR(64) NOT NULL,
	max_file_size SMALLINT NOT NULL,
	max_threads SMALLINT NOT NULL,
	default_style VARCHAR(45) NOT NULL,
	locked bool NOT NULL,
	created_at timestamp NOT NULL,
	anonymous_name VARCHAR(45) NOT NULL DEFAULT 'Anonymous',
	force_anonymous bool NOT NULL,
	autosage_after SMALLINT NOT NULL,
	no_images_after SMALLINT NOT NULL,
	max_message_length SMALLINT NOT NULL,
	min_message_length SMALLINT NOT NULL,
	allow_embeds bool NOT NULL,
	redictect_to_thread bool NOT NULL,
	require_file bool NOT NULL,
	enable_catalog bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(section_id) REFERENCES sections(id),
	UNIQUE(dir),
	UNIQUE(uri),
	UNIQUE(navbar_position)
);

create table threads(
	id serial,
	board_id int NOT NULL,
	locked bool NOT NULL,
	stickied bool NOT NULL,
	anchored bool NOT NULL,
	cyclical bool NOT NULL,
	last_bump timestamp NOT NULL,
	deleted_at timestamp NOT NULL,
	is_deleted bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id)
);

CREATE INDEX thread_deleted_index ON threads(is_deleted);

create table posts(
	id serial,
	thread_id int NOT NULL,
	is_top_post bool NOT NULL,
	ip int NOT NULL,
	created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name VARCHAR(50) NOT NULL,
	tripcode VARCHAR(10) NOT NULL,
	is_role_signature bool NOT NULL DEFAULT FALSE,
	email VARCHAR(50) NOT NULL,
	subject VARCHAR(100) NOT NULL,
	message text NOT NULL,
	message_raw text NOT NULL,
	password text NOT NULL,
	deleted_at timestamp NOT NULL,
	is_deleted bool NOT NULL,
	banned_message text,
	PRIMARY KEY(id),
	FOREIGN KEY(thread_id) REFERENCES threads(id)
);

CREATE INDEX top_post_index ON posts(is_top_post);

create table files(
	id serial,
	post_id int NOT NULL,
	file_order int NOT NULL,
	original_filename VARCHAR(255) NOT NULL,
	filename VARCHAR(45) NOT NULL,
	checksum int NOT NULL,
	file_size int NOT NULL,
	is_spoilered bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(post_id) REFERENCES posts(id),
	UNIQUE(post_id, file_order)
);

create table staff(
	id serial,
	username VARCHAR(45) NOT NULL,
	password_checksum VARCHAR(120) NOT NULL,
	global_rank int,
	added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP NOT NULL,
	is_active bool NOT NULL DEFAULT TRUE,
	PRIMARY KEY(id),
	UNIQUE(username)
);

create table sessions(
	id serial,
	staff_id int NOT NULL,
	expires TIMESTAMP NOT NULL,
	data varchar(45) NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table board_staff(
	board_id int NOT NULL,
	staff_id int NOT NULL,
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table announcements(
	id serial,
	staff_id int NOT NULL,
	subject VARCHAR(45) NOT NULL,
	message text NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table ip_ban(
	id serial,
	staff_id int NOT NULL,
	board_id int NOT NULL,
	banned_for_post_id int NOT NULL,
	copy_post_text text NOT NULL,
	is_active bool NOT NULL,
	ip int NOT NULL,
	issued_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	permanent bool NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message text NOT NULL,
	can_appeal bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id),
	FOREIGN KEY(banned_for_post_id) REFERENCES posts(id)
);

create table ip_ban_audit(
	ip_ban_id int NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id int NOT NULL,
	board_id int NOT NULL,
	is_active bool NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	permanent bool NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message text NOT NULL,
	can_appeal bool NOT NULL,
	PRIMARY KEY(ip_ban_id, timestamp),
	FOREIGN KEY(ip_ban_id) REFERENCES ip_ban(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table ip_ban_appeals(
	id serial,
	staff_id int,
	ip_ban_id int NOT NULL,
	appeal_text text NOT NULL,
	staff_response text,
	is_denied bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id),
	FOREIGN KEY(ip_ban_id) REFERENCES ip_ban(id)
);

create table ip_ban_appeals_audit(
	appeal_id int NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id int,
	appeal_text text NOT NULL,
	staff_response text,
	is_denied bool NOT NULL,
	PRIMARY KEY(appeal_id, timestamp),
	FOREIGN KEY(staff_id) REFERENCES staff(id),
	FOREIGN KEY(appeal_id) REFERENCES ip_ban_appeals(id)
);

create table reports(
	id serial, 
	handled_by_staff_id int,
	post_id int NOT NULL,
	ip int NOT NULL,
	reason text NOT NULL,
	is_cleared bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(handled_by_staff_id) REFERENCES staff(id),
	FOREIGN KEY(post_id) REFERENCES posts(id)
);

create table reports_audit(
	report_id int NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	handled_by_staff_id int,
	is_cleared bool NOT NULL,
	FOREIGN KEY(handled_by_staff_id) REFERENCES staff(id),
	FOREIGN KEY(report_id) REFERENCES reports(id)
);

create table filename_ban(
	id serial,
	board_id int,
	staff_id int NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	filename VARCHAR(255) NOT NULL,
	is_regex bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table username_ban(
	id serial,
	board_id int,
	staff_id int NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	username VARCHAR(255) NOT NULL,
	is_regex bool NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table file_ban(
	id serial,
	board_id int,
	staff_id int NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	checksum int NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);

create table wordfilters(
	id serial,
	board_id int,
	staff_id int NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	search VARCHAR(75) NOT NULL CHECK (search <> ''),
	is_regex bool NOT NULL,
	change_to VARCHAR(75) NOT NULL,
	PRIMARY KEY(id),
	FOREIGN KEY(board_id) REFERENCES boards(id),
	FOREIGN KEY(staff_id) REFERENCES staff(id)
);
