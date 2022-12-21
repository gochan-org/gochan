package gcsql

import (
	"html/template"
	"time"
)

// table: DBPREFIXannouncements
type Announcement struct {
	ID        uint      `json:"no"`   // sql: `id`
	StaffID   string    `json:"name"` // sql: `staff_id`
	Subject   string    `json:"sub"`  // sql: `subject`
	Message   string    `json:"com"`  // sql: `message`
	Timestamp time.Time `json:"-"`    // sql: `timestamp`
}

// table: DBPREFIXboard_staff
type BoardStaff struct {
	BoardID uint // sql: `board_id`
	StaffID uint // sql: `staff_id`
}

// table: DBPREFIXboards
type Board struct {
	ID               int       // sql: `id`
	SectionID        int       // sql: `section_id`
	URI              string    // sql: `uri`
	Dir              string    // sql: `dir`
	NavbarPosition   int       // sql: `navbar_position`
	Title            string    // sql: `title`
	Subtitle         string    // sql: `suttitle`
	Description      string    // sql: `description`
	MaxFilesize      int       // sql: `max_file_size`
	MaxThreads       int       // sql: `max_threads`
	DefaultStyle     string    // sql: `default_style`
	Locked           bool      // sql: `locked`
	CreatedAt        time.Time // sql: `created_at`
	AnonymousName    string    // sql: `anonymous_name`
	ForceAnonymous   bool      // sql: `force_anonymous`
	AutosageAfter    int       // sql: `autosage_after`
	NoImagesAfter    int       // sql: `no_images_after`
	MaxMessageLength int       // sql: `max_message_length`
	MinMessageLength int       // sql: `min_message_length`
	AllowEmbeds      bool      // sql: `allow_embeds`
	RedirectToThread bool      // sql: `redirect_to_thread`
	RequireFile      bool      // sql: `require_file`
	EnableCatalog    bool      // sql: `enable_catalog`
}

// FileBan contains the information associated with a specific file ban.
// table: DBPREFIXfile_ban
type FileBan struct {
	ID        int       // sql: `id`
	BoardID   *int      // sql: `board_id`
	StaffID   int       // sql: `staff_id`
	StaffNote string    // sql: `staff_note`
	IssuedAt  time.Time // sql: `issued_at`
	Checksum  string    // sql: `checksum`
}

type filenameOrUsernameBanBase struct {
	ID        int       // sql: id
	BoardID   *int      // sql: board_id
	StaffID   int       // sql: staff_id
	StaffNote string    // sql: staff_note
	IssuedAt  time.Time // sql: issued_at
	check     string    // replaced with username or filename
	IsRegex   bool      // sql: is_regex
}

// FilenameBan represents a ban on a specific filename or filename regular expression.
// table: DBPREFIXfilename_ban
type FilenameBan struct {
	filenameOrUsernameBanBase
	Filename string // sql: `filename`
	IsRegex  bool   // sql: `is_regex`
}

// Upload represents a file attached to a post.
// table: DBPREFIXfiles
type Upload struct {
	ID               int    // sql: `id`
	PostID           int    // sql: `post_id`
	FileOrder        int    // sql: `file_order`
	OriginalFilename string // sql: `original_filename`
	Filename         string // sql: `filename`
	Checksum         string // sql: `checksum`
	FileSize         int    // sql: `file_size`
	IsSpoilered      bool   // sql: `is_spoilered`
	ThumbnailWidth   int    // sql: `thumbnail_width`
	ThumbnailHeight  int    // sql: `thumbnail_height`
	Width            int    // sql: `width`
	Height           int    // sql: `height`
}

// used to composition IPBan and IPBanAudit
type ipBanBase struct {
	IsActive    bool
	IsThreadBan bool
	ExpiresAt   time.Time
	StaffID     int
	AppealAt    time.Time
	Permanent   bool
	StaffNote   string
	Message     string
	CanAppeal   bool
}

// IPBan contains the information association with a specific ip ban.
// table: DBPREFIXip_ban
type IPBan struct {
	ID              int
	BoardID         *int
	BannedForPostID *int
	CopyPostText    template.HTML
	IP              string
	IssuedAt        time.Time
	ipBanBase
}

// table: DBPREFIXip_ban_audit
type IPBanAudit struct {
	IPBanID   int       // sql: `ip_ban_id`
	Timestamp time.Time // sql: `timestamp`
	ipBanBase
}

// used to composition IPBanAppeal and IPBanAppealAudit
type ipBanAppealBase struct {
	StaffID       int    // sql: `staff_id`
	AppealText    string // sql: `appeal_text`
	StaffResponse string // sql: `staff_response`
	IsDenied      bool   // sql: `is_denied`
}

// table: DBPREFIXip_ban_appeals
type IPBanAppeal struct {
	ID      int // sql: `id`
	IPBanID int // sql: `ip_ban_id`
	ipBanAppealBase
}

// table: DBPREFIXip_ban_appeals_audit
type IPBanAppealAudit struct {
	AppealID  int       // sql: `appeal_id`
	Timestamp time.Time // sql: `timestamp`
	ipBanAppealBase
}

// table: DBPREFIXposts
type Post struct {
	ID              int           // sql: `id`
	ThreadID        int           // sql: `thread_id`
	IsTopPost       bool          // sql: `is_top_post`
	IP              string        // sql: `ip`
	CreatedOn       time.Time     // sql: `created_on`
	Name            string        // sql: `name`
	Tripcode        string        // sql: `tripcode`
	IsRoleSignature bool          // sql: `is_role_signature`
	Email           string        // sql: `email`
	Subject         string        // sql: `subject`
	Message         template.HTML // sql: `message`
	MessageRaw      string        // sql: `message_raw`
	Password        string        // sql: `password`
	DeletedAt       time.Time     // sql: `deleted_at`
	IsDeleted       bool          // sql: `is_deleted`
	BannedMessage   string        // sql: `banned_message`

	sanitized bool
}

// table: DBPREFIXreports
type Report struct {
	ID               int    // sql: `id`
	HandledByStaffID int    // sql: `handled_by_staff_id`
	PostID           int    // sql: `post_id`
	IP               string // sql: `ip`
	Reason           string // sql: `reason`
	IsCleared        bool   // sql: `is_cleared`
}

// table: DBPREFIXreports_audit
type ReportAudit struct {
	Report           int       // sql: `report_id`
	Timestamp        time.Time // sql: `timestamp`
	HandledByStaffID int       // sql: `handled_by_staff_id`
	IsCleared        bool      // sql: `is_cleared`
}

// table: DBPREFIXsections
type Section struct {
	ID           int    // sql: `id`
	Name         string // sql: `name`
	Abbreviation string // sql: `abbreviation`
	Position     int    // sql: `position`
	Hidden       bool   // sql: `hidden`
}

// table: DBPREFIXsessions
type LoginSession struct {
	ID      int       // sql: `id`
	StaffID int       // sql: `staff_id`
	Expires time.Time // sql: `expires`
	Data    string    // sql: `data`
}

// DBPREFIXstaff
type Staff struct {
	ID               int       // sql: `id`
	Username         string    // sql: `username`
	PasswordChecksum string    // sql: `password_checksum`
	Rank             int       // sql: `global_rank`
	AddedOn          time.Time // sql: `added_on`
	LastLogin        time.Time // sql: `last_login`
	IsActive         bool      // sql: `is_active`
}

// table: DBPREFIXthreads
type Thread struct {
	ID        int       // sql: `id`
	BoardID   int       // sql: `board_id`
	Locked    bool      // sql: `locked`
	Stickied  bool      // sql: `stickied`
	Anchored  bool      // sql: `anchored`
	Cyclical  bool      // sql: `cyclical`
	LastBump  time.Time // sql: `last_bump`
	DeletedAt time.Time // sql: `deleted_at`
	IsDeleted bool      // sql: `is_deleted`
}

// table: DBPREFIXusername_ban
type UsernameBan struct {
	filenameOrUsernameBanBase
	Username string // sql: `username`
}

// table DBPREFIXwordfilters
type Wordfilter struct {
	ID        int       `json:"id"`         // sql: `id`
	BoardDirs *string   `json:"boards"`     // sql: `board_dirs`
	StaffID   int       `json:"staff_id"`   // sql: `staff_id`
	StaffNote string    `json:"staff_note"` // sql: `staff_note`
	IssuedAt  time.Time `json:"issued_at"`  // sql: `issued_at`
	Search    string    `json:"search"`     // sql: `search`
	IsRegex   bool      `json:"is_regex"`   // sql: `is_regex`
	ChangeTo  string    `json:"change_to"`  // sql: `change_to`
}
