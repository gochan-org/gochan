package pre2021

const (
	sectionsQuery = "SELECT id, list_order, hidden, name, abbreviation FROM DBPREFIXsections"

	boardsQuery = `SELECT id, list_order, dir, title, subtitle, description, section, max_file_size, max_pages,
default_style, locked, created_on, anonymous, forced_anon, autosage_after, no_images_after, max_message_length, embeds_allowed,
redirect_to_thread, require_file, enable_catalog
FROM DBPREFIXboards`

	postsQuery = `SELECT id, boardid, parentid, name, tripcode, email, subject, message, message_raw, password, filename,
filename_original, file_checksum, filesize, image_w, image_h, thumb_w, thumb_h, ip, timestamp, autosage,
bumped, stickied, locked FROM DBPREFIXposts WHERE deleted_timestamp IS NULL`

	threadsQuery = postsQuery + " AND parentid = 0"

	staffQuery = `SELECT id, username, rank, boards, added_on, last_active FROM DBPREFIXstaff`

	bansQuery = `SELECT  id, allow_read, COALESCE(ip, '') as ip, name, name_is_regex, filename, file_checksum, boards, staff,
timestamp, expires, permaban, reason, type, staff_note, appeal_at, can_appeal FROM DBPREFIXbanlist`

	announcementsQuery = "SELECT id, subject, message, poster, timestamp FROM DBPREFIXannouncements"

	renameTableStatementTemplate = "ALTER TABLE %s RENAME TO _tmp_%s"
)

var (
	boardAlterStatements = []string{
		"ALTER TABLE DBPREFIXboards RENAME COLUMN section TO section_id",
		"ALTER TABLE DBPREFIXboards RENAME COLUMN list_order TO navbar_position",
		"ALTER TABLE DBPREFIXboards RENAME COLUMN created_on TO created_at",
		"ALTER TABLE DBPREFIXboards RENAME COLUMN anonymous TO anonymous_name",
		"ALTER TABLE DBPREFIXboards RENAME COLUMN forced_anon TO force_anonymous",
		"ALTER TABLE DBPREFIXboards RENAME COLUMN embeds_allowed TO allow_embeds",
		"ALTER TABLE DBPREFIXboards ADD COLUMN uri VARCHAR(45) NOT NULL DEFAULT ''",
		"ALTER TABLE DBPREFIXboards ADD COLUMN min_message_length SMALLINT NOT NULL DEFAULT 0",
		"ALTER TABLE DBPREFIXboards ADD COLUMN max_threads SMALLINT NOT NULL DEFAULT 65535",
		// the following statements don't work in SQLite since it doesn't support adding foreign keys after table creation.
		// "in-place" migration support for SQLite may be removed
		"ALTER TABLE DBPREFIXboards ADD CONSTRAINT boards_section_id_fk FOREIGN KEY (section_id) REFERENCES DBPREFIXsections(id)",
		"ALTER TABLE DBPREFIXboards ADD CONSTRAINT boards_dir_unique UNIQUE (dir)",
		"ALTER TABLE DBPREFIXboards ADD CONSTRAINT boards_uri_unique UNIQUE (uri)",
	}

	// tables to be renamed to _tmp_DBPREFIX* to work around SQLite's lack of support for changing/removing columns
	renameTables = []string{
		"DBPREFIXannouncements", "DBPREFIXappeals", "DBPREFIXbanlist", "DBPREFIXboards", "DBPREFIXembeds", "DBPREFIXlinks",
		"DBPREFIXposts", "DBPREFIXreports", "DBPREFIXsections", "DBPREFIXsessions", "DBPREFIXstaff", "DBPREFIXwordfilters",
	}
)
