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

	bansQuery = `SELECT  id, allow_read, COALESCE(ip, '') as ip, name, name_is_regex, filename, file_checksum, boards, staff,
timestamp, expires, permaban, reason, type, staff_note, appeal_at, can_appeal FROM DBPREFIXbanlist`
)
