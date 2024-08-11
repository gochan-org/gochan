package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"net"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

// table: DBPREFIXannouncements
type Announcement struct {
	ID        uint      // sql: id
	StaffID   uint      // sql: staff_id
	Subject   string    // sql: subject
	Message   string    // sql: message
	Timestamp time.Time // sql: timestamp
}

// table: DBPREFIXboard_staff
type BoardStaff struct {
	BoardID uint // sql: board_id
	StaffID uint // sql: staff_id
}

// table: DBPREFIXboards
type Board struct {
	ID               int       // sql: id
	SectionID        int       // sql: section_id
	URI              string    // sql: uri
	Dir              string    // sql: dir
	NavbarPosition   int       // sql: navbar_position
	Title            string    // sql: title
	Subtitle         string    // sql: suttitle
	Description      string    // sql: description
	MaxFilesize      int       // sql: max_file_size
	MaxThreads       int       // sql: max_threads
	DefaultStyle     string    // sql: default_style
	Locked           bool      // sql: locked
	CreatedAt        time.Time // sql: created_at
	AnonymousName    string    // sql: anonymous_name
	ForceAnonymous   bool      // sql: force_anonymous
	AutosageAfter    int       // sql: autosage_after
	NoImagesAfter    int       // sql: no_images_after
	MaxMessageLength int       // sql: max_message_length
	MinMessageLength int       // sql: min_message_length
	AllowEmbeds      bool      // sql: allow_embeds
	RedirectToThread bool      // sql: redirect_to_thread
	RequireFile      bool      // sql: require_file
	EnableCatalog    bool      // sql: enable_catalog
}

// Deprecated, use PostFilter instead, with a condition field = "checksum" if Fingerprinter is nil
// or "ahash" otherwise.
// FileBan contains the information associated with a specific file ban.
type FileBan struct {
	ID            int       // sql: id
	BoardID       *int      // sql: board_id
	StaffID       int       // sql: staff_id
	StaffNote     string    // sql: staff_note
	IssuedAt      time.Time // sql: issued_at
	Checksum      string    // sql: checksum
	Fingerprinter *string   // sql: fingerprinter
	BanIP         bool      // sql: ban_ip
	BanIPMessage  *string   // sql: ban_ip_message
}

// ApplyIPBan bans the given IP if it posted a banned image
// If BanIP is false, it returns with no error
func (fb *FileBan) ApplyIPBan(postIP string) error {
	if !fb.BanIP {
		return nil
	}
	now := time.Now()
	ipBan := &IPBan{
		RangeStart: postIP,
		RangeEnd:   postIP,
		IssuedAt:   now,
	}
	ipBan.IsActive = true
	ipBan.CanAppeal = true
	ipBan.AppealAt = now
	ipBan.StaffID = fb.StaffID
	ipBan.Permanent = true
	if fb.BoardID != nil {
		ipBan.BoardID = new(int)
		*ipBan.BoardID = *fb.BoardID
	}
	if fb.BanIPMessage == nil {
		ipBan.Message = "posting disallowed image, resulting in ban"
	} else {
		ipBan.Message = *fb.BanIPMessage
	}
	if fb.StaffNote == "" {
		ipBan.StaffNote = "fingerprint"
	}

	return NewIPBan(ipBan)
}

// Deprecated, use PostFilter instead
type filenameOrUsernameBanBase struct {
	ID        int       // sql: id
	BoardID   *int      // sql: board_id
	StaffID   int       // sql: staff_id
	StaffNote string    // sql: staff_note
	IssuedAt  time.Time // sql: issued_at
	check     string    // replaced with username or filename
	IsRegex   bool      // sql: is_regex
}

// Deprecated, use PostFilter instead, with a condition field = "filename"
// FilenameBan represents a ban on a specific filename or filename regular expression.
type FilenameBan struct {
	filenameOrUsernameBanBase
	Filename string // sql: `filename`
	IsRegex  bool   // sql: `is_regex`
}

// Filter represents an entry in gochan's new filter system which merges username bans, file bans, and filename bans,
// and will allow moderators to block posts based on the user's name, email, subject, message content, and other fields.
// table: DBPREFIXfilters
type Filter struct {
	ID          int       `json:"id"`         // sql: id
	StaffID     *int      `json:"staff_id"`   // sql: staff_id
	StaffNote   string    `json:"staff_note"` // sql: staff_note
	IssuedAt    time.Time `json:"issued_at"`  // sql: issued_at
	MatchAction string    // sql: match_action
	MatchDetail string    // sql: match_detail
	IsActive    bool      // sql: is_active
	conditions  []FilterCondition
}

// FilterCondition represents a condition to be checked against when a post is submitted
// table: DBPREFIXfilter_conditions
type FilterCondition struct {
	ID       int    // sql: id
	FilterID int    // sql: filter_id
	IsRegex  bool   // sql: is_regex
	Search   string // sql: search
	Field    string // sql: field
}

func (fc *FilterCondition) insert(ctx context.Context, tx *sql.Tx) error {
	_, err := ExecContextSQL(ctx, tx,
		`INSERT INTO DBPREFIXfilter_conditions(filter_id, is_regex, search, field) VALUES(?,?,?,?)`,
		fc.FilterID, fc.IsRegex, fc.Search, fc.Field,
	)
	return err
}

// Upload represents a file attached to a post.
// table: DBPREFIXfiles
type Upload struct {
	ID               int    // sql: id
	PostID           int    // sql: post_id
	FileOrder        int    // sql: file_order
	OriginalFilename string // sql: original_filename
	Filename         string // sql: filename
	Checksum         string // sql: checksum
	FileSize         int    // sql: file_size
	IsSpoilered      bool   // sql: is_spoilered
	ThumbnailWidth   int    // sql: thumbnail_width
	ThumbnailHeight  int    // sql: thumbnail_height
	Width            int    // sql: width
	Height           int    // sql: height
}

// IPBanBase used to composition IPBan and IPBanAudit. It does not represent a SQL table by itself
type IPBanBase struct {
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
	RangeStart      string
	RangeEnd        string
	IssuedAt        time.Time
	IPBanBase
}

// Deprecated: Use the RangeStart and RangeEnd fields or gcutil.GetIPRangeSubnet.
// IP was previously a field in the IPBan struct before range bans were
// implemented. This is here as a fallback for templates
func (ipb *IPBan) IP() string {
	if ipb.RangeStart == ipb.RangeEnd {
		return ipb.RangeStart
	}
	inet, err := gcutil.GetIPRangeSubnet(ipb.RangeStart, ipb.RangeEnd)
	if err != nil {
		return "?"
	}
	return inet.String()
}

func (ipb *IPBan) IsBanned(ipStr string) (bool, error) {
	ipn, err := gcutil.GetIPRangeSubnet(ipb.RangeStart, ipb.RangeEnd)
	if err != nil {
		return false, err
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false, errors.New("invalid IP address")
	}
	return ipn.Contains(ip), nil
}

// table: DBPREFIXip_ban_audit
type IPBanAudit struct {
	IPBanID   int       // sql: `ip_ban_id`
	Timestamp time.Time // sql: `timestamp`
	IPBanBase
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
	Password        string        `json:"-"` // sql: `password`
	DeletedAt       time.Time     // sql: `deleted_at`
	IsDeleted       bool          // sql: `is_deleted`
	BannedMessage   string        // sql: `banned_message`
	Flag            string        // sql: `flag`
	Country         string        // sql: `country`
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

// table: DBPREFIXstaff
type Staff struct {
	ID               int       // sql: `id`
	Username         string    // sql: `username`
	PasswordChecksum string    `json:"-"` // sql: `password_checksum`
	Rank             int       // sql: `global_rank`
	AddedOn          time.Time `json:"-"` // sql: `added_on`
	LastLogin        time.Time `json:"-"` // sql: `last_login`
	IsActive         bool      `json:"-"` // sql: `is_active`
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

// Deprecated, use PostFilter instead, and a FilterCondition with Field = "name"
type UsernameBan struct {
	filenameOrUsernameBanBase
	Username string // sql: `username`
}

// Wordfilter is used for filters that are expected to have a single FilterCondition and a "replace" MatchAction
type Wordfilter struct {
	Filter
}
