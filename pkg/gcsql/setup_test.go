// this source file contains helper functions for gcsql

package gcsql

import (
	"database/sql"
	"database/sql/driver"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

var (
	testInitDBMySQLStatements = []string{
		`CREATE TABLE database_version\(\s+component VARCHAR\(40\) NOT NULL PRIMARY KEY,\s+version INT NOT NULL \)`,
		`CREATE TABLE sections\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+name TEXT NOT NULL,\s+abbreviation TEXT NOT NULL,\s+position SMALLINT NOT NULL,\s+hidden BOOL NOT NULL \)`,
		`CREATE TABLE boards\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+section_id BIGINT NOT NULL,\s+uri VARCHAR\(45\) NOT NULL,\s+dir VARCHAR\(45\) NOT NULL,\s+navbar_position SMALLINT NOT NULL,\s+title VARCHAR\(45\) NOT NULL,\s+subtitle VARCHAR\(64\) NOT NULL,\s+description VARCHAR\(64\) NOT NULL,\s+max_file_size INT NOT NULL,\s+max_threads SMALLINT NOT NULL,  default_style VARCHAR\(45\) NOT NULL,\s+locked BOOL NOT NULL,\s+created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+anonymous_name VARCHAR\(45\) NOT NULL DEFAULT 'Anonymous',\s+force_anonymous BOOL NOT NULL,\s+autosage_after SMALLINT NOT NULL,\s+no_images_after SMALLINT NOT NULL,\s+max_message_length SMALLINT NOT NULL,\s+min_message_length SMALLINT NOT NULL,\s+allow_embeds BOOL NOT NULL,\s+redirect_to_thread BOOL NOT NULL,\s+require_file BOOL NOT NULL,\s+enable_catalog BOOL NOT NULL,\s+CONSTRAINT boards_section_id_fk\s+FOREIGN KEY\(section_id\) REFERENCES sections\(id\),\s+CONSTRAINT boards_dir_unique UNIQUE\(dir\),\s+CONSTRAINT boards_uri_unique UNIQUE\(uri\)\s*\)`,
		`CREATE TABLE threads\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+board_id BIGINT NOT NULL,\s+locked BOOL NOT NULL DEFAULT FALSE,\s+stickied BOOL NOT NULL DEFAULT FALSE,\s+anchored BOOL NOT NULL DEFAULT FALSE,\s+cyclical BOOL NOT NULL DEFAULT FALSE,\s+spoilered BOOL NOT NULL DEFAULT FALSE,\s+last_bump TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+CONSTRAINT threads_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE\s*\)`,
		`CREATE INDEX thread_deleted_index ON threads\(is_deleted\)`,
		`CREATE TABLE posts\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+thread_id BIGINT NOT NULL,\s+is_top_post BOOL NOT NULL DEFAULT FALSE,\s+ip VARBINARY\(16\) NOT NULL,\s+created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+name VARCHAR\(50\) NOT NULL DEFAULT '',\s+tripcode VARCHAR\(10\) NOT NULL DEFAULT '',\s+is_secure_tripcode BOOL NOT NULL DEFAULT FALSE,\s+is_role_signature BOOL NOT NULL DEFAULT FALSE,  email VARCHAR\(50\) NOT NULL DEFAULT '',\s+subject VARCHAR\(100\) NOT NULL DEFAULT '',\s+message TEXT NOT NULL,\s+message_raw TEXT NOT NULL,\s+password TEXT NOT NULL,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+banned_message TEXT,\s+flag VARCHAR\(45\) NOT NULL DEFAULT '',\s+country VARCHAR\(80\) NOT NULL DEFAULT '',\s+CONSTRAINT posts_thread_id_fk\s+FOREIGN KEY\(thread_id\) REFERENCES threads\(id\) ON DELETE CASCADE \)`,
		`CREATE INDEX top_post_index ON posts\(is_top_post\)`,
		`CREATE TABLE files\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+post_id BIGINT NOT NULL,\s+file_order INT NOT NULL,\s+original_filename VARCHAR\(255\) NOT NULL,\s+filename VARCHAR\(45\) NOT NULL,\s+checksum TEXT NOT NULL,\s+file_size INT NOT NULL,\s+is_spoilered BOOL NOT NULL,\s+thumbnail_width INT NOT NULL,\s+thumbnail_height INT NOT NULL,\s+width INT NOT NULL,\s+height INT NOT NULL,\s+CONSTRAINT files_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE,\s+CONSTRAINT files_post_id_file_order_unique UNIQUE\(post_id, file_order\) \)`,
		`CREATE TABLE staff\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+username VARCHAR\(45\) NOT NULL,\s+password_checksum VARCHAR\(120\) NOT NULL,\s+global_rank INT,\s+added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+last_login TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_active BOOL NOT NULL DEFAULT TRUE,\s+CONSTRAINT staff_username_unique UNIQUE\(username\) \)`,
		`CREATE TABLE sessions\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+staff_id BIGINT NOT NULL,\s+expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+data VARCHAR\(45\) NOT NULL,\s+CONSTRAINT sessions_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE board_staff\(\s+board_id BIGINT NOT NULL,\s+staff_id BIGINT NOT NULL,  CONSTRAINT board_staff_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_pk PRIMARY KEY \(board_id,staff_id\) \)`,
		`CREATE TABLE announcements\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+staff_id BIGINT NOT NULL,\s+subject VARCHAR\(45\) NOT NULL,\s+message TEXT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+CONSTRAINT announcements_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) \)`,
		`CREATE TABLE ip_ban\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+staff_id BIGINT NOT NULL, board_id BIGINT, banned_for_post_id BIGINT, copy_post_text TEXT NOT NULL, is_thread_ban BOOL NOT NULL, is_active BOOL NOT NULL, range_start VARBINARY\(16\) NOT NULL, range_end VARBINARY\(16\) NOT NULL, issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, permanent BOOL NOT NULL, staff_note VARCHAR\(255\) NOT NULL, message TEXT NOT NULL, can_appeal BOOL NOT NULL, CONSTRAINT ip_ban_board_id_fk FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE, CONSTRAINT ip_ban_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\), CONSTRAINT ip_ban_banned_for_post_id_fk FOREIGN KEY\(banned_for_post_id\) REFERENCES posts\(id\) ON DELETE SET NULL \)`,
		`CREATE TABLE ip_ban_audit\(\s+ip_ban_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT NOT NULL,\s+is_active BOOL NOT NULL,\s+is_thread_ban BOOL NOT NULL,\s+expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+permanent BOOL NOT NULL,\s+staff_note VARCHAR\(255\) NOT NULL,\s+message TEXT NOT NULL,\s+can_appeal BOOL NOT NULL,\s+PRIMARY KEY\(ip_ban_id, timestamp\),\s+CONSTRAINT ip_ban_audit_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE,\s+CONSTRAINT ip_ban_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s+\)`,
		`CREATE TABLE ip_ban_appeals\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+staff_id BIGINT,\s+ip_ban_id BIGINT NOT NULL,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+CONSTRAINT ip_ban_appeals_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE ip_ban_appeals_audit\(\s+appeal_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+PRIMARY KEY\(appeal_id, timestamp\),\s+CONSTRAINT ip_ban_appeals_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_audit_appeal_id_fk\s+FOREIGN KEY\(appeal_id\) REFERENCES ip_ban_appeals\(id\)\s+ON DELETE CASCADE \)`,
		`CREATE TABLE reports\(\s+id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s+handled_by_staff_id BIGINT,\s+post_id BIGINT NOT NULL,\s+ip VARBINARY\(16\) NOT NULL,\s+reason TEXT NOT NULL,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),  CONSTRAINT reports_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE reports_audit\(\s+report_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+handled_by_staff_id BIGINT,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_audit_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT reports_audit_report_id_fk\s+FOREIGN KEY\(report_id\) REFERENCES reports\(id\) ON DELETE CASCADE\s+\)`,
		`CREATE TABLE filters\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s*staff_id BIGINT,\s*staff_note VARCHAR\(255\) NOT NULL,\s*issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*match_action VARCHAR\(45\) NOT NULL DEFAULT 'replace',\s*match_detail TEXT NOT NULL,\s*handle_if_any BOOL NOT NULL DEFAULT FALSE,\s*is_active BOOL NOT NULL,\s*CONSTRAINT filters_staff_id_fk\s*FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s*ON DELETE SET NULL\s*\)`,
		`CREATE TABLE filter_boards\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*board_id BIGINT NOT NULL,\s*CONSTRAINT filter_boards_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_boards_board_id_fk\s*FOREIGN KEY\(board_id\)\s*REFERENCES boards\(id\)\s*ON DELETE CASCADE\s*\)`,
		`CREATE TABLE filter_conditions\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*match_mode SMALLINT NOT NULL,\s*search VARCHAR\(75\) NOT NULL,\s*field VARCHAR\(75\) NOT NULL,\s*CONSTRAINT filter_conditions_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_conditions_search_check CHECK \(search <> '' OR match_mode = 3\)\s*\)`,
		`CREATE TABLE filter_hits\(\s*id BIGINT NOT NULL AUTO_INCREMENT UNIQUE PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*post_data TEXT NOT NULL,\s*match_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*CONSTRAINT filter_hits_filter_id_fk\s*FOREIGN KEY\(filter_id\)\s*REFERENCES filters\(id\)\s*ON DELETE CASCADE\s*\)`,
		`INSERT INTO database_version\(component, version\)\s+VALUES\('gochan', 4\)`,
	}
	testInitDBPostgresStatements = []string{
		`CREATE TABLE database_version\(\s+component VARCHAR\(40\) NOT NULL PRIMARY KEY,\s+version INT NOT NULL \)`,
		`CREATE TABLE sections\(\s+id BIGSERIAL PRIMARY KEY,\s+name TEXT NOT NULL,\s+abbreviation TEXT NOT NULL,\s+position SMALLINT NOT NULL,\s+hidden BOOL NOT NULL \)`,
		`CREATE TABLE boards\(\s*id BIGSERIAL PRIMARY KEY,\s+section_id BIGINT NOT NULL,\s+uri VARCHAR\(45\) NOT NULL,\s+dir VARCHAR\(45\) NOT NULL,\s+navbar_position SMALLINT NOT NULL,\s+title VARCHAR\(45\) NOT NULL,\s+subtitle VARCHAR\(64\) NOT NULL,\s+description VARCHAR\(64\) NOT NULL,\s+max_file_size INT NOT NULL,\s+max_threads SMALLINT NOT NULL,  default_style VARCHAR\(45\) NOT NULL,\s+locked BOOL NOT NULL,\s+created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+anonymous_name VARCHAR\(45\) NOT NULL DEFAULT 'Anonymous',\s+force_anonymous BOOL NOT NULL,\s+autosage_after SMALLINT NOT NULL,\s+no_images_after SMALLINT NOT NULL,\s+max_message_length SMALLINT NOT NULL,\s+min_message_length SMALLINT NOT NULL,\s+allow_embeds BOOL NOT NULL,\s+redirect_to_thread BOOL NOT NULL,\s+require_file BOOL NOT NULL,\s+enable_catalog BOOL NOT NULL,\s+CONSTRAINT boards_section_id_fk\s+FOREIGN KEY\(section_id\) REFERENCES sections\(id\),\s+CONSTRAINT boards_dir_unique UNIQUE\(dir\),\s+CONSTRAINT boards_uri_unique UNIQUE\(uri\)\s*\)`,
		`CREATE TABLE threads\(\s*id BIGSERIAL PRIMARY KEY,\s+board_id BIGINT NOT NULL,\s+locked BOOL NOT NULL DEFAULT FALSE,\s+stickied BOOL NOT NULL DEFAULT FALSE,\s+anchored BOOL NOT NULL DEFAULT FALSE,\s+cyclical BOOL NOT NULL DEFAULT FALSE,\s+spoilered BOOL NOT NULL DEFAULT FALSE,\s+last_bump TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+CONSTRAINT threads_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE\s*\)`,
		`CREATE INDEX thread_deleted_index ON threads\(is_deleted\)`,
		`CREATE TABLE posts\(\s+id BIGSERIAL PRIMARY KEY,\s+thread_id BIGINT NOT NULL,\s+is_top_post BOOL NOT NULL DEFAULT FALSE,\s+ip INET NOT NULL,\s+created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+name VARCHAR\(50\) NOT NULL DEFAULT '',\s+tripcode VARCHAR\(10\) NOT NULL DEFAULT '',\s+is_secure_tripcode BOOL NOT NULL DEFAULT FALSE,\s+is_role_signature BOOL NOT NULL DEFAULT FALSE,  email VARCHAR\(50\) NOT NULL DEFAULT '',\s+subject VARCHAR\(100\) NOT NULL DEFAULT '',\s+message TEXT NOT NULL,\s+message_raw TEXT NOT NULL,\s+password TEXT NOT NULL,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+banned_message TEXT,\s+flag VARCHAR\(45\) NOT NULL DEFAULT '',\s+country VARCHAR\(80\) NOT NULL DEFAULT '',\s+CONSTRAINT posts_thread_id_fk\s+FOREIGN KEY\(thread_id\) REFERENCES threads\(id\) ON DELETE CASCADE \)`,
		`CREATE INDEX top_post_index ON posts\(is_top_post\)`,
		`CREATE TABLE files\(\s+id BIGSERIAL PRIMARY KEY,\s+post_id BIGINT NOT NULL,\s+file_order INT NOT NULL,\s+original_filename VARCHAR\(255\) NOT NULL,\s+filename VARCHAR\(45\) NOT NULL,\s+checksum TEXT NOT NULL,\s+file_size INT NOT NULL,\s+is_spoilered BOOL NOT NULL,\s+thumbnail_width INT NOT NULL,\s+thumbnail_height INT NOT NULL,\s+width INT NOT NULL,\s+height INT NOT NULL,\s+CONSTRAINT files_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE,\s+CONSTRAINT files_post_id_file_order_unique UNIQUE\(post_id, file_order\) \)`,
		`CREATE TABLE staff\(\s+id BIGSERIAL PRIMARY KEY,\s+username VARCHAR\(45\) NOT NULL,\s+password_checksum VARCHAR\(120\) NOT NULL,\s+global_rank INT,\s+added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+last_login TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_active BOOL NOT NULL DEFAULT TRUE,\s+CONSTRAINT staff_username_unique UNIQUE\(username\) \)`,
		`CREATE TABLE sessions\(\s+id BIGSERIAL PRIMARY KEY,\s+staff_id BIGINT NOT NULL,\s+expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+data VARCHAR\(45\) NOT NULL,\s+CONSTRAINT sessions_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE board_staff\(\s+board_id BIGINT NOT NULL,\s+staff_id BIGINT NOT NULL,  CONSTRAINT board_staff_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_pk PRIMARY KEY \(board_id,staff_id\) \)`,
		`CREATE TABLE announcements\(\s+id BIGSERIAL PRIMARY KEY,\s+staff_id BIGINT NOT NULL,\s+subject VARCHAR\(45\) NOT NULL,\s+message TEXT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+CONSTRAINT announcements_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) \)`,
		`CREATE TABLE ip_ban\(\s+id BIGSERIAL PRIMARY KEY,\s+staff_id BIGINT NOT NULL, board_id BIGINT, banned_for_post_id BIGINT, copy_post_text TEXT NOT NULL, is_thread_ban BOOL NOT NULL, is_active BOOL NOT NULL, range_start INET NOT NULL, range_end INET NOT NULL, issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, permanent BOOL NOT NULL, staff_note VARCHAR\(255\) NOT NULL, message TEXT NOT NULL, can_appeal BOOL NOT NULL, CONSTRAINT ip_ban_board_id_fk FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE, CONSTRAINT ip_ban_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\), CONSTRAINT ip_ban_banned_for_post_id_fk FOREIGN KEY\(banned_for_post_id\) REFERENCES posts\(id\) ON DELETE SET NULL \)`,
		`CREATE TABLE ip_ban_audit\(\s+ip_ban_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT NOT NULL,\s+is_active BOOL NOT NULL,\s+is_thread_ban BOOL NOT NULL,\s+expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+permanent BOOL NOT NULL,\s+staff_note VARCHAR\(255\) NOT NULL,\s+message TEXT NOT NULL,\s+can_appeal BOOL NOT NULL,\s+PRIMARY KEY\(ip_ban_id, timestamp\),\s+CONSTRAINT ip_ban_audit_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE,\s+CONSTRAINT ip_ban_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s+\)`,
		`CREATE TABLE ip_ban_appeals\(\s+id BIGSERIAL PRIMARY KEY,\s+staff_id BIGINT,\s+ip_ban_id BIGINT NOT NULL,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+CONSTRAINT ip_ban_appeals_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE ip_ban_appeals_audit\(\s+appeal_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+PRIMARY KEY\(appeal_id, timestamp\),\s+CONSTRAINT ip_ban_appeals_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_audit_appeal_id_fk\s+FOREIGN KEY\(appeal_id\) REFERENCES ip_ban_appeals\(id\)\s+ON DELETE CASCADE \)`,
		`CREATE TABLE reports\(\s+id BIGSERIAL PRIMARY KEY,\s+handled_by_staff_id BIGINT,\s+post_id BIGINT NOT NULL,\s+ip INET NOT NULL,\s+reason TEXT NOT NULL,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),  CONSTRAINT reports_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE reports_audit\(\s+report_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+handled_by_staff_id BIGINT,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_audit_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT reports_audit_report_id_fk\s+FOREIGN KEY\(report_id\) REFERENCES reports\(id\) ON DELETE CASCADE\s+\)`,
		`CREATE TABLE filters\(\s*id BIGSERIAL PRIMARY KEY,\s*staff_id BIGINT,\s*staff_note VARCHAR\(255\) NOT NULL,\s*issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*match_action VARCHAR\(45\) NOT NULL DEFAULT 'replace',\s*match_detail TEXT NOT NULL,\s*handle_if_any BOOL NOT NULL DEFAULT FALSE,\s*is_active BOOL NOT NULL,\s*CONSTRAINT filters_staff_id_fk\s*FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s*ON DELETE SET NULL\s*\)`,
		`CREATE TABLE filter_boards\(\s*id BIGSERIAL PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*board_id BIGINT NOT NULL,\s*CONSTRAINT filter_boards_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_boards_board_id_fk\s*FOREIGN KEY\(board_id\) REFERENCES boards\(id\)\s*ON DELETE CASCADE\s*\)`,
		`CREATE TABLE filter_conditions\(\s*id BIGSERIAL PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*match_mode SMALLINT NOT NULL,\s*search VARCHAR\(75\) NOT NULL,\s*field VARCHAR\(75\) NOT NULL,\s*CONSTRAINT filter_conditions_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_conditions_search_check CHECK \(search <> '' OR match_mode = 3\)\s*\)`,
		`CREATE TABLE filter_hits\(\s*id BIGSERIAL PRIMARY KEY,\s*filter_id BIGINT NOT NULL,\s*post_data TEXT NOT NULL,\s*match_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*CONSTRAINT filter_hits_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE\s*\)`,
		`INSERT INTO database_version\(component, version\)\s+VALUES\('gochan', 4\)`,
	}
	testInitDBSQLite3Statements = []string{
		`CREATE TABLE database_version\(\s+component VARCHAR\(40\) NOT NULL PRIMARY KEY,\s+version INT NOT NULL \)`,
		`CREATE TABLE sections\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+name TEXT NOT NULL,\s+abbreviation TEXT NOT NULL,\s+position SMALLINT NOT NULL,\s+hidden BOOL NOT NULL \)`,
		`CREATE TABLE boards\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+section_id BIGINT NOT NULL,\s+uri VARCHAR\(45\) NOT NULL,\s+dir VARCHAR\(45\) NOT NULL,\s+navbar_position SMALLINT NOT NULL,\s+title VARCHAR\(45\) NOT NULL,\s+subtitle VARCHAR\(64\) NOT NULL,\s+description VARCHAR\(64\) NOT NULL,\s+max_file_size INT NOT NULL,\s+max_threads SMALLINT NOT NULL,  default_style VARCHAR\(45\) NOT NULL,\s+locked BOOL NOT NULL,\s+created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+anonymous_name VARCHAR\(45\) NOT NULL DEFAULT 'Anonymous',\s+force_anonymous BOOL NOT NULL,\s+autosage_after SMALLINT NOT NULL,\s+no_images_after SMALLINT NOT NULL,\s+max_message_length SMALLINT NOT NULL,\s+min_message_length SMALLINT NOT NULL,\s+allow_embeds BOOL NOT NULL,\s+redirect_to_thread BOOL NOT NULL,\s+require_file BOOL NOT NULL,\s+enable_catalog BOOL NOT NULL,\s+CONSTRAINT boards_section_id_fk\s+FOREIGN KEY\(section_id\) REFERENCES sections\(id\),\s+CONSTRAINT boards_dir_unique UNIQUE\(dir\),\s+CONSTRAINT boards_uri_unique UNIQUE\(uri\)\s*\)`,
		`CREATE TABLE threads\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+board_id BIGINT NOT NULL,\s+locked BOOL NOT NULL DEFAULT FALSE,\s+stickied BOOL NOT NULL DEFAULT FALSE,\s+anchored BOOL NOT NULL DEFAULT FALSE,\s+cyclical BOOL NOT NULL DEFAULT FALSE,\s+spoilered BOOL NOT NULL DEFAULT FALSE,\s+last_bump TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+CONSTRAINT threads_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE\s*\)`,
		`CREATE INDEX thread_deleted_index ON threads\(is_deleted\)`,
		`CREATE TABLE posts\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+thread_id BIGINT NOT NULL,\s+is_top_post BOOL NOT NULL DEFAULT FALSE,\s+ip VARCHAR\(45\) NOT NULL,\s+created_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+name VARCHAR\(50\) NOT NULL DEFAULT '',\s+tripcode VARCHAR\(10\) NOT NULL DEFAULT '',\s+is_secure_tripcode BOOL NOT NULL DEFAULT FALSE,\s+is_role_signature BOOL NOT NULL DEFAULT FALSE,  email VARCHAR\(50\) NOT NULL DEFAULT '',\s+subject VARCHAR\(100\) NOT NULL DEFAULT '',\s+message TEXT NOT NULL,\s+message_raw TEXT NOT NULL,\s+password TEXT NOT NULL,\s+deleted_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_deleted BOOL NOT NULL DEFAULT FALSE,\s+banned_message TEXT,\s+flag VARCHAR\(45\) NOT NULL DEFAULT '',\s+country VARCHAR\(80\) NOT NULL DEFAULT '',\s+CONSTRAINT posts_thread_id_fk\s+FOREIGN KEY\(thread_id\) REFERENCES threads\(id\) ON DELETE CASCADE \)`,
		`CREATE INDEX top_post_index ON posts\(is_top_post\)`,
		`CREATE TABLE files\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+post_id BIGINT NOT NULL,\s+file_order INT NOT NULL,\s+original_filename VARCHAR\(255\) NOT NULL,\s+filename VARCHAR\(45\) NOT NULL,\s+checksum TEXT NOT NULL,\s+file_size INT NOT NULL,\s+is_spoilered BOOL NOT NULL,\s+thumbnail_width INT NOT NULL,\s+thumbnail_height INT NOT NULL,\s+width INT NOT NULL,\s+height INT NOT NULL,\s+CONSTRAINT files_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE,\s+CONSTRAINT files_post_id_file_order_unique UNIQUE\(post_id, file_order\) \)`,
		`CREATE TABLE staff\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+username VARCHAR\(45\) NOT NULL,\s+password_checksum VARCHAR\(120\) NOT NULL,\s+global_rank INT,\s+added_on TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+last_login TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+is_active BOOL NOT NULL DEFAULT TRUE,\s+CONSTRAINT staff_username_unique UNIQUE\(username\) \)`,
		`CREATE TABLE sessions\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+staff_id BIGINT NOT NULL,\s+expires TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+data VARCHAR\(45\) NOT NULL,\s+CONSTRAINT sessions_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE board_staff\(\s+board_id BIGINT NOT NULL,\s+staff_id BIGINT NOT NULL,  CONSTRAINT board_staff_board_id_fk\s+FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) ON DELETE CASCADE,\s+CONSTRAINT board_staff_pk PRIMARY KEY \(board_id,staff_id\) \)`,
		`CREATE TABLE announcements\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+staff_id BIGINT NOT NULL,\s+subject VARCHAR\(45\) NOT NULL,\s+message TEXT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+CONSTRAINT announcements_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\) \)`,
		`CREATE TABLE ip_ban\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+staff_id BIGINT NOT NULL, board_id BIGINT, banned_for_post_id BIGINT, copy_post_text TEXT NOT NULL, is_thread_ban BOOL NOT NULL, is_active BOOL NOT NULL, range_start VARCHAR\(45\) NOT NULL, range_end VARCHAR\(45\) NOT NULL, issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, permanent BOOL NOT NULL, staff_note VARCHAR\(255\) NOT NULL, message TEXT NOT NULL, can_appeal BOOL NOT NULL, CONSTRAINT ip_ban_board_id_fk FOREIGN KEY\(board_id\) REFERENCES boards\(id\) ON DELETE CASCADE, CONSTRAINT ip_ban_staff_id_fk FOREIGN KEY\(staff_id\) REFERENCES staff\(id\), CONSTRAINT ip_ban_banned_for_post_id_fk FOREIGN KEY\(banned_for_post_id\) REFERENCES posts\(id\) ON DELETE SET NULL \)`,
		`CREATE TABLE ip_ban_audit\(\s+ip_ban_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT NOT NULL,\s+is_active BOOL NOT NULL,\s+is_thread_ban BOOL NOT NULL,\s+expires_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+appeal_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+permanent BOOL NOT NULL,\s+staff_note VARCHAR\(255\) NOT NULL,\s+message TEXT NOT NULL,\s+can_appeal BOOL NOT NULL,\s+PRIMARY KEY\(ip_ban_id, timestamp\),\s+CONSTRAINT ip_ban_audit_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE,\s+CONSTRAINT ip_ban_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s+\)`,
		`CREATE TABLE ip_ban_appeals\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+staff_id BIGINT,\s+ip_ban_id BIGINT NOT NULL,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+CONSTRAINT ip_ban_appeals_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_ip_ban_id_fk\s+FOREIGN KEY\(ip_ban_id\) REFERENCES ip_ban\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE ip_ban_appeals_audit\(\s+appeal_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+staff_id BIGINT,\s+appeal_text TEXT NOT NULL,\s+staff_response TEXT,\s+is_denied BOOL NOT NULL,\s+PRIMARY KEY\(appeal_id, timestamp\),\s+CONSTRAINT ip_ban_appeals_audit_staff_id_fk\s+FOREIGN KEY\(staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT ip_ban_appeals_audit_appeal_id_fk\s+FOREIGN KEY\(appeal_id\) REFERENCES ip_ban_appeals\(id\)\s+ON DELETE CASCADE \)`,
		`CREATE TABLE reports\(\s+id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s+handled_by_staff_id BIGINT,\s+post_id BIGINT NOT NULL,\s+ip VARCHAR\(45\) NOT NULL,\s+reason TEXT NOT NULL,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),  CONSTRAINT reports_post_id_fk\s+FOREIGN KEY\(post_id\) REFERENCES posts\(id\) ON DELETE CASCADE \)`,
		`CREATE TABLE reports_audit\(\s+report_id BIGINT NOT NULL,\s+timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s+handled_by_staff_id BIGINT,\s+is_cleared BOOL NOT NULL,\s+CONSTRAINT reports_audit_handled_by_staff_id_fk\s+FOREIGN KEY\(handled_by_staff_id\) REFERENCES staff\(id\),\s+CONSTRAINT reports_audit_report_id_fk\s+FOREIGN KEY\(report_id\) REFERENCES reports\(id\) ON DELETE CASCADE\s+\)`,
		`CREATE TABLE filters\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s*staff_id BIGINT,\s*staff_note VARCHAR\(255\) NOT NULL,\s*issued_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*match_action VARCHAR\(45\) NOT NULL DEFAULT 'replace',\s*match_detail TEXT NOT NULL,\s*handle_if_any BOOL NOT NULL DEFAULT FALSE,\s*is_active BOOL NOT NULL,\s*CONSTRAINT filters_staff_id_fk\s*FOREIGN KEY\(staff_id\) REFERENCES staff\(id\)\s*ON DELETE SET NULL\s*\)`,
		`CREATE TABLE filter_boards\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s*filter_id BIGINT NOT NULL,\s*board_id BIGINT NOT NULL,\s*CONSTRAINT filter_boards_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_boards_board_id_fk\s*FOREIGN KEY\(board_id\) REFERENCES boards\(id\)\s*ON DELETE CASCADE\s*\)`,
		`CREATE TABLE filter_conditions\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s*filter_id BIGINT NOT NULL,\s*match_mode SMALLINT NOT NULL,\s*search VARCHAR\(75\) NOT NULL,\s*field VARCHAR\(75\) NOT NULL,\s*CONSTRAINT filter_conditions_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE,\s*CONSTRAINT filter_conditions_search_check CHECK \(search <> '' OR match_mode = 3\)\s*\)`,
		`CREATE TABLE filter_hits\(\s*id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,\s*filter_id BIGINT NOT NULL,\s*post_data TEXT NOT NULL,\s*match_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,\s*CONSTRAINT filter_hits_filter_id_fk\s*FOREIGN KEY\(filter_id\) REFERENCES filters\(id\)\s*ON DELETE CASCADE\s*\)`,
		`INSERT INTO database_version\(component, version\)\s+VALUES\('gochan', 4\)`,
	}
)

func setupAndProvisionMockDB(t *testing.T, mock sqlmock.Sqlmock, dbType string, dbName string) error {
	t.Helper()
	if gcdb == nil || gcdb.db == nil {
		return ErrNotConnected
	}

	mock.ExpectPrepare("CREATE DATABASE " + dbName).
		ExpectExec().WillReturnResult(driver.ResultNoRows)
	mock.ExpectBegin()
	var statements []string
	staffInsert := `INSERT INTO staff\s+\(username, password_checksum, global_rank\)\s+VALUES\(`
	sectionsInsert := `INSERT INTO sections \(name, abbreviation, hidden, position\) VALUES \(`
	boardsInsert := `INSERT INTO boards\s*\(` +
		`section_id,\s*uri,\s*dir,\s*navbar_position,\s*title,\s*subtitle,\s*description,\s*max_file_size,\s*` +
		`max_threads,\s*default_style,\s*locked,\s*anonymous_name,\s*force_anonymous,\s*autosage_after,\s*no_images_after,\s*` +
		`max_message_length,\s*min_message_length,\s*allow_embeds,\s*redirect_to_thread,\s*require_file,\s*enable_catalog\)\s+VALUES\(`

	switch dbType {
	case "mysql":
		statements = testInitDBMySQLStatements
	case "postgres":
		statements = testInitDBPostgresStatements
	case "sqlite3":
		statements = testInitDBSQLite3Statements
	default:
		return ErrUnsupportedDB
	}

	// set up parameterized statements, with sqlite3 and postgres both having $# and mysql having ?
	switch dbType {
	case "mysql":
		staffInsert += `\?,\?,\?\)`
		sectionsInsert += `\?,\?,\?,\?\)`
		boardsInsert += `\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?,\?\)`
	case "sqlite3":
		fallthrough
	case "postgresql":
		staffInsert += `\$1,\$2,\$3\)`
		sectionsInsert += `\$1,\$2,\$3,\$4\)`
		boardsInsert += `\$1,\$2,\$3,\$4,\$5,\$6,\$7,\$8,\$9,\$10,\$11,\$12,\$13,\$14,\$15,\$16,\$17,\$18,\$19,\$20,\$21\)`
	}

	for _, stmtStr := range statements {
		mock.ExpectPrepare(stmtStr).
			ExpectExec().WithoutArgs().
			WillReturnResult(driver.ResultNoRows)
	}

	mock.ExpectCommit()
	mock.ExpectPrepare(`SELECT COUNT\(id\) FROM staff`).
		ExpectQuery().
		WillReturnRows(sqlmock.NewRows([]string{}))

	mock.ExpectPrepare(staffInsert).
		ExpectExec().
		WithArgs("admin", sqlmock.AnyArg(), 3).
		WillReturnResult(driver.ResultNoRows)

	mock.ExpectPrepare(`SELECT id FROM sections WHERE name = 'Main'`).
		ExpectQuery().WithoutArgs().WillReturnError(sql.ErrNoRows)

	mock.ExpectBegin()

	mock.ExpectPrepare(`SELECT COALESCE\(MAX\(position\) \+ 1, 1\) FROM sections`).
		ExpectQuery().WithoutArgs().WillReturnError(sql.ErrNoRows)

	mock.ExpectPrepare(sectionsInsert).
		ExpectExec().WithArgs("Main", "main", false, 1).WillReturnResult(driver.ResultNoRows)

	mock.ExpectPrepare(`SELECT MAX\(id\) FROM sections`).
		ExpectQuery().WithoutArgs().
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectCommit()

	mock.ExpectPrepare(boardsInsert).ExpectExec().WithArgs(
		1, "test", "test", 3, "Testing Board", "Board for testing stuff", "Board for testing stuff", 15000,
		300, "pipes.css", false, "Anonymous", false, 500, -1, 1500, 0, false, false, false, true,
	).WillReturnResult(driver.ResultNoRows)

	mock.ExpectPrepare("SELECT id FROM boards WHERE dir = ?").
		ExpectQuery().WithArgs("test").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))

	_, err := Exec(nil, "CREATE DATABASE gochan")
	if err != nil {
		return err
	}

	if err = buildNewDatabase(dbType); err != nil {
		return err
	}

	mock.ExpectationsWereMet()
	return nil
}
