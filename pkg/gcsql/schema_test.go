package gcsql

import (
	"regexp"

	"github.com/DATA-DOG/go-sqlmock"
)

func createMockSchema() (err error) {
	if gcdb == nil {
		return ErrNotConnected
	}

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_database_version(
	component VARCHAR(40) NOT NULL PRIMARY KEY,
	version INT NOT NULL
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

	sqm.ExpectPrepare(regexp.QuoteMeta(`CREATE TABLE gc_sections(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	name TEXT NOT NULL,
	abbreviation TEXT NOT NULL,
	position SMALLINT NOT NULL,
	hidden BOOL NOT NULL
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(2, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_boards(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	section_id BIGINT NOT NULL,
	uri VARCHAR(45) NOT NULL,
	dir VARCHAR(45) NOT NULL,
	navbar_position SMALLINT NOT NULL,
	title VARCHAR(45) NOT NULL,
	subtitle VARCHAR(64) NOT NULL,
	description VARCHAR(64) NOT NULL,
	max_file_size INT NOT NULL,
	max_threads SMALLINT NOT NULL,
	default_style VARCHAR(45) NOT NULL,
	locked BOOL NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	anonymous_name VARCHAR(45) NOT NULL DEFAULT 'Anonymous',
	force_anonymous BOOL NOT NULL,
	autosage_after SMALLINT NOT NULL,
	no_images_after SMALLINT NOT NULL,
	max_message_length SMALLINT NOT NULL,
	min_message_length SMALLINT NOT NULL,
	allow_embeds BOOL NOT NULL,
	redirect_to_thread BOOL NOT NULL,
	require_file BOOL NOT NULL,
	enable_catalog BOOL NOT NULL,
	CONSTRAINT boards_section_id_fk FOREIGN KEY(section_id) REFERENCES gc_sections(id),
	CONSTRAINT boards_dir_unique UNIQUE(dir),
	CONSTRAINT boards_uri_unique UNIQUE(uri)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(3, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_threads(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_id BIGINT NOT NULL,
	locked BOOL NOT NULL DEFAULT FALSE,
	stickied BOOL NOT NULL DEFAULT FALSE,
	anchored BOOL NOT NULL DEFAULT FALSE,
	cyclical BOOL NOT NULL DEFAULT FALSE,
	last_bump TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_deleted BOOL NOT NULL DEFAULT FALSE,
	CONSTRAINT threads_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(4, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE INDEX thread_deleted_index ON gc_threads(is_deleted);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(5, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_posts(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	thread_id BIGINT NOT NULL,
	is_top_post BOOL NOT NULL DEFAULT FALSE,
	ip VARCHAR(45) NOT NULL,
	created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name VARCHAR(50) NOT NULL DEFAULT '',
	tripcode VARCHAR(10) NOT NULL DEFAULT '',
	is_role_signature BOOL NOT NULL DEFAULT FALSE,
	email VARCHAR(50) NOT NULL DEFAULT '',
	subject VARCHAR(100) NOT NULL DEFAULT '',
	message TEXT NOT NULL,
	message_raw TEXT NOT NULL,
	password TEXT NOT NULL,
	deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_deleted BOOL NOT NULL DEFAULT FALSE,
	banned_message TEXT,
	CONSTRAINT posts_thread_id_fk FOREIGN KEY(thread_id) REFERENCES gc_threads(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(5, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_files(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	post_id BIGINT NOT NULL,
	file_order INT NOT NULL,
	original_filename VARCHAR(255) NOT NULL,
	filename VARCHAR(45) NOT NULL,
	checksum TEXT NOT NULL,
	file_size INT NOT NULL,
	is_spoilered BOOL NOT NULL,
	thumbnail_width INT NOT NULL,
	thumbnail_height INT NOT NULL,
	width INT NOT NULL,
	height INT NOT NULL,
	CONSTRAINT files_post_id_fk FOREIGN KEY(post_id) REFERENCES gc_posts(id) ON DELETE CASCADE,
	CONSTRAINT files_post_id_file_order_unique UNIQUE(post_id, file_order)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(6, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_staff(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	username VARCHAR(45) NOT NULL,
	password_checksum VARCHAR(120) NOT NULL,
	global_rank INT,
	added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_active BOOL NOT NULL DEFAULT TRUE,
	CONSTRAINT staff_username_unique UNIQUE(username)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(7, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_sessions(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	expires TIMESTAMP NOT NULL,
	data VARCHAR(45) NOT NULL,
	CONSTRAINT sessions_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_board_staff(
	board_id BIGINT NOT NULL,
	staff_id BIGINT NOT NULL,
	CONSTRAINT board_staff_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE,
	CONSTRAINT board_staff_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id) ON DELETE CASCADE,
	CONSTRAINT board_staff_pk PRIMARY KEY (board_id,staff_id)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_announcements(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	subject VARCHAR(45) NOT NULL,
	message TEXT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT announcements_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_ip_ban(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	board_id BIGINT,
	banned_for_post_id BIGINT,
	copy_post_text TEXT NOT NULL,
	is_thread_ban BOOL NOT NULL,
	is_active BOOL NOT NULL,
	ip VARCHAR(45) NOT NULL,
	issued_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	permanent BOOL NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message TEXT NOT NULL,
	can_appeal BOOL NOT NULL,
	CONSTRAINT ip_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE,
	CONSTRAINT ip_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id),
	CONSTRAINT ip_ban_banned_for_post_id_fk FOREIGN KEY(banned_for_post_id) REFERENCES gc_posts(id) ON DELETE SET NULL
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_ip_ban_audit(
	ip_ban_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id BIGINT NOT NULL,
	is_active BOOL NOT NULL,
	is_thread_ban BOOL NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	permanent BOOL NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message TEXT NOT NULL,
	can_appeal BOOL NOT NULL,
	PRIMARY KEY(ip_ban_id, timestamp),
	CONSTRAINT ip_ban_audit_ip_ban_id_fk FOREIGN KEY(ip_ban_id) REFERENCES gc_ip_ban(id) ON DELETE CASCADE,
	CONSTRAINT ip_ban_audit_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id)
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_ip_ban_appeals(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT,
	ip_ban_id BIGINT NOT NULL,
	appeal_text TEXT NOT NULL,
	staff_response TEXT,
	is_denied BOOL NOT NULL,
	CONSTRAINT ip_ban_appeals_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id),
	CONSTRAINT ip_ban_appeals_ip_ban_id_fk FOREIGN KEY(ip_ban_id) REFERENCES gc_ip_ban(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_ip_ban_appeals_audit(
	appeal_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id BIGINT,
	appeal_text TEXT NOT NULL,
	staff_response TEXT,
	is_denied BOOL NOT NULL,
	PRIMARY KEY(appeal_id, timestamp),
	CONSTRAINT ip_ban_appeals_audit_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id),
	CONSTRAINT ip_ban_appeals_audit_appeal_id_fk FOREIGN KEY(appeal_id) REFERENCES gc_ip_ban_appeals(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_reports(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY, 
	handled_by_staff_id BIGINT,
	post_id BIGINT NOT NULL,
	ip VARCHAR(45) NOT NULL,
	reason TEXT NOT NULL,
	is_cleared BOOL NOT NULL,
	CONSTRAINT reports_handled_by_staff_id_fk FOREIGN KEY(handled_by_staff_id) REFERENCES gc_staff(id),
	CONSTRAINT reports_post_id_fk FOREIGN KEY(post_id) REFERENCES gc_posts(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_reports_audit(
	report_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	handled_by_staff_id BIGINT,
	is_cleared BOOL NOT NULL,
	CONSTRAINT reports_audit_handled_by_staff_id_fk FOREIGN KEY(handled_by_staff_id) REFERENCES gc_staff(id),
	CONSTRAINT reports_audit_report_id_fk FOREIGN KEY(report_id) REFERENCES gc_reports(id) ON DELETE CASCADE
);`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_filename_ban(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_id BIGINT,
	staff_id BIGINT NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	filename VARCHAR(255) NOT NULL,
	is_regex BOOL NOT NULL,
	CONSTRAINT filename_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE,
	CONSTRAINT filename_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id)
)`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_username_ban(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_id BIGINT,
	staff_id BIGINT NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	username VARCHAR(255) NOT NULL,
	is_regex BOOL NOT NULL,
	CONSTRAINT username_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE,
	CONSTRAINT username_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id)
)`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_file_ban(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_id BIGINT,
	staff_id BIGINT NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	checksum TEXT NOT NULL,
	CONSTRAINT file_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES gc_boards(id) ON DELETE CASCADE,
	CONSTRAINT file_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id)
)`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	sqm.ExpectPrepare(prepTestQueryString(`CREATE TABLE gc_wordfilters(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_dirs VARCHAR(255) DEFAULT '*',
	staff_id BIGINT NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	search VARCHAR(75) NOT NULL,
	is_regex BOOL NOT NULL,
	change_to VARCHAR(75) NOT NULL,
	CONSTRAINT wordfilters_staff_id_fk FOREIGN KEY(staff_id) REFERENCES gc_staff(id),
	CONSTRAINT wordfilters_search_check CHECK (search <> '')
)`)).ExpectExec().WillReturnResult(sqlmock.NewResult(8, 1))

	// start fulfilling the expected execs

	if _, err = ExecSQL(`CREATE TABLE DBPREFIXdatabase_version(
	component VARCHAR(40) NOT NULL PRIMARY KEY,
	version INT NOT NULL
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXsections(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	name TEXT NOT NULL,
	abbreviation TEXT NOT NULL,
	position SMALLINT NOT NULL,
	hidden BOOL NOT NULL
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXboards(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	section_id BIGINT NOT NULL,
	uri VARCHAR(45) NOT NULL,
	dir VARCHAR(45) NOT NULL,
	navbar_position SMALLINT NOT NULL,
	title VARCHAR(45) NOT NULL,
	subtitle VARCHAR(64) NOT NULL,
	description VARCHAR(64) NOT NULL,
	max_file_size INT NOT NULL,
	max_threads SMALLINT NOT NULL,
	default_style VARCHAR(45) NOT NULL,
	locked BOOL NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	anonymous_name VARCHAR(45) NOT NULL DEFAULT 'Anonymous',
	force_anonymous BOOL NOT NULL,
	autosage_after SMALLINT NOT NULL,
	no_images_after SMALLINT NOT NULL,
	max_message_length SMALLINT NOT NULL,
	min_message_length SMALLINT NOT NULL,
	allow_embeds BOOL NOT NULL,
	redirect_to_thread BOOL NOT NULL,
	require_file BOOL NOT NULL,
	enable_catalog BOOL NOT NULL,
	CONSTRAINT boards_section_id_fk FOREIGN KEY(section_id) REFERENCES DBPREFIXsections(id),
	CONSTRAINT boards_dir_unique UNIQUE(dir),
	CONSTRAINT boards_uri_unique UNIQUE(uri)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXthreads(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	board_id BIGINT NOT NULL,
	locked BOOL NOT NULL DEFAULT FALSE,
	stickied BOOL NOT NULL DEFAULT FALSE,
	anchored BOOL NOT NULL DEFAULT FALSE,
	cyclical BOOL NOT NULL DEFAULT FALSE,
	last_bump TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_deleted BOOL NOT NULL DEFAULT FALSE,
	CONSTRAINT threads_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE INDEX thread_deleted_index ON DBPREFIXthreads(is_deleted);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXposts(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	thread_id BIGINT NOT NULL,
	is_top_post BOOL NOT NULL DEFAULT FALSE,
	ip VARCHAR(45) NOT NULL,
	created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	name VARCHAR(50) NOT NULL DEFAULT '',
	tripcode VARCHAR(10) NOT NULL DEFAULT '',
	is_role_signature BOOL NOT NULL DEFAULT FALSE,
	email VARCHAR(50) NOT NULL DEFAULT '',
	subject VARCHAR(100) NOT NULL DEFAULT '',
	message TEXT NOT NULL,
	message_raw TEXT NOT NULL,
	password TEXT NOT NULL,
	deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_deleted BOOL NOT NULL DEFAULT FALSE,
	banned_message TEXT,
	CONSTRAINT posts_thread_id_fk FOREIGN KEY(thread_id) REFERENCES DBPREFIXthreads(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXfiles(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	post_id BIGINT NOT NULL,
	file_order INT NOT NULL,
	original_filename VARCHAR(255) NOT NULL,
	filename VARCHAR(45) NOT NULL,
	checksum TEXT NOT NULL,
	file_size INT NOT NULL,
	is_spoilered BOOL NOT NULL,
	thumbnail_width INT NOT NULL,
	thumbnail_height INT NOT NULL,
	width INT NOT NULL,
	height INT NOT NULL,
	CONSTRAINT files_post_id_fk FOREIGN KEY(post_id) REFERENCES DBPREFIXposts(id) ON DELETE CASCADE,
	CONSTRAINT files_post_id_file_order_unique UNIQUE(post_id, file_order)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXstaff(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	username VARCHAR(45) NOT NULL,
	password_checksum VARCHAR(120) NOT NULL,
	global_rank INT,
	added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	last_login TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	is_active BOOL NOT NULL DEFAULT TRUE,
	CONSTRAINT staff_username_unique UNIQUE(username)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXsessions(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	expires TIMESTAMP NOT NULL,
	data VARCHAR(45) NOT NULL,
	CONSTRAINT sessions_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXboard_staff(
	board_id BIGINT NOT NULL,
	staff_id BIGINT NOT NULL,
	CONSTRAINT board_staff_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE,
	CONSTRAINT board_staff_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id) ON DELETE CASCADE,
	CONSTRAINT board_staff_pk PRIMARY KEY (board_id,staff_id)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXannouncements(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	subject VARCHAR(45) NOT NULL,
	message TEXT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	CONSTRAINT announcements_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXip_ban(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT NOT NULL,
	board_id BIGINT,
	banned_for_post_id BIGINT,
	copy_post_text TEXT NOT NULL,
	is_thread_ban BOOL NOT NULL,
	is_active BOOL NOT NULL,
	ip VARCHAR(45) NOT NULL,
	issued_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	permanent BOOL NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message TEXT NOT NULL,
	can_appeal BOOL NOT NULL,
	CONSTRAINT ip_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE,
	CONSTRAINT ip_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id),
	CONSTRAINT ip_ban_banned_for_post_id_fk FOREIGN KEY(banned_for_post_id) REFERENCES DBPREFIXposts(id) ON DELETE SET NULL
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXip_ban_audit(
	ip_ban_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id BIGINT NOT NULL,
	is_active BOOL NOT NULL,
	is_thread_ban BOOL NOT NULL,
	expires_at TIMESTAMP NOT NULL,
	appeal_at TIMESTAMP NOT NULL,
	permanent BOOL NOT NULL,
	staff_note VARCHAR(255) NOT NULL,
	message TEXT NOT NULL,
	can_appeal BOOL NOT NULL,
	PRIMARY KEY(ip_ban_id, timestamp),
	CONSTRAINT ip_ban_audit_ip_ban_id_fk FOREIGN KEY(ip_ban_id) REFERENCES DBPREFIXip_ban(id) ON DELETE CASCADE,
	CONSTRAINT ip_ban_audit_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id)
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXip_ban_appeals(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
	staff_id BIGINT,
	ip_ban_id BIGINT NOT NULL,
	appeal_text TEXT NOT NULL,
	staff_response TEXT,
	is_denied BOOL NOT NULL,
	CONSTRAINT ip_ban_appeals_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id),
	CONSTRAINT ip_ban_appeals_ip_ban_id_fk FOREIGN KEY(ip_ban_id) REFERENCES DBPREFIXip_ban(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXip_ban_appeals_audit(
	appeal_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	staff_id BIGINT,
	appeal_text TEXT NOT NULL,
	staff_response TEXT,
	is_denied BOOL NOT NULL,
	PRIMARY KEY(appeal_id, timestamp),
	CONSTRAINT ip_ban_appeals_audit_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id),
	CONSTRAINT ip_ban_appeals_audit_appeal_id_fk FOREIGN KEY(appeal_id) REFERENCES DBPREFIXip_ban_appeals(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXreports(
	id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY, 
	handled_by_staff_id BIGINT,
	post_id BIGINT NOT NULL,
	ip VARCHAR(45) NOT NULL,
	reason TEXT NOT NULL,
	is_cleared BOOL NOT NULL,
	CONSTRAINT reports_handled_by_staff_id_fk FOREIGN KEY(handled_by_staff_id) REFERENCES DBPREFIXstaff(id),
	CONSTRAINT reports_post_id_fk FOREIGN KEY(post_id) REFERENCES DBPREFIXposts(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXreports_audit(
	report_id BIGINT NOT NULL,
	timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	handled_by_staff_id BIGINT,
	is_cleared BOOL NOT NULL,
	CONSTRAINT reports_audit_handled_by_staff_id_fk FOREIGN KEY(handled_by_staff_id) REFERENCES DBPREFIXstaff(id),
	CONSTRAINT reports_audit_report_id_fk FOREIGN KEY(report_id) REFERENCES DBPREFIXreports(id) ON DELETE CASCADE
);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXfilename_ban(
		id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
		board_id BIGINT,
		staff_id BIGINT NOT NULL,
		staff_note VARCHAR(255) NOT NULL,
		issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		filename VARCHAR(255) NOT NULL,
		is_regex BOOL NOT NULL,
		CONSTRAINT filename_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE,
		CONSTRAINT filename_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id)
	);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXusername_ban(
		id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
		board_id BIGINT,
		staff_id BIGINT NOT NULL,
		staff_note VARCHAR(255) NOT NULL,
		issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		username VARCHAR(255) NOT NULL,
		is_regex BOOL NOT NULL,
		CONSTRAINT username_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE,
		CONSTRAINT username_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id)
	);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXfile_ban(
		id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
		board_id BIGINT,
		staff_id BIGINT NOT NULL,
		staff_note VARCHAR(255) NOT NULL,
		issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		checksum TEXT NOT NULL,
		CONSTRAINT file_ban_board_id_fk FOREIGN KEY(board_id) REFERENCES DBPREFIXboards(id) ON DELETE CASCADE,
		CONSTRAINT file_ban_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id)
	);`); err != nil {
		return err
	}
	if _, err = ExecSQL(`CREATE TABLE DBPREFIXwordfilters(
		id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,
		board_dirs VARCHAR(255) DEFAULT '*',
		staff_id BIGINT NOT NULL,
		staff_note VARCHAR(255) NOT NULL,
		issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		search VARCHAR(75) NOT NULL,
		is_regex BOOL NOT NULL,
		change_to VARCHAR(75) NOT NULL,
		CONSTRAINT wordfilters_staff_id_fk FOREIGN KEY(staff_id) REFERENCES DBPREFIXstaff(id),
		CONSTRAINT wordfilters_search_check CHECK (search <> '')
	);`); err != nil {
		return err
	}

	return sqm.ExpectationsWereMet()
}
