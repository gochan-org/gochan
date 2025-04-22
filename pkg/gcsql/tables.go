package gcsql

import (
	"context"
	"database/sql"
	"errors"
	"html/template"
	"net"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

// table: DBPREFIXannouncements
type Announcement struct {
	ID        int       // sql: id
	StaffID   int       // sql: staff_id
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
	ID               int       `json:"-"`                 // sql: id
	SectionID        int       `json:"-"`                 // sql: section_id
	URI              string    `json:"-"`                 // sql: uri
	Dir              string    `json:"board"`             // sql: dir
	NavbarPosition   int       `json:"-"`                 // sql: navbar_position
	Title            string    `json:"title"`             // sql: title
	Subtitle         string    `json:"meta_description"`  // sql: suttitle
	Description      string    `json:"-"`                 // sql: description
	MaxFilesize      int       `json:"max_filesize"`      // sql: max_file_size
	MaxThreads       int       `json:"-"`                 // sql: max_threads
	DefaultStyle     string    `json:"-"`                 // sql: default_style
	Locked           bool      `json:"is_archived"`       // sql: locked
	CreatedAt        time.Time `json:"-"`                 // sql: created_at
	AnonymousName    string    `json:"-"`                 // sql: anonymous_name
	ForceAnonymous   bool      `json:"-"`                 // sql: force_anonymous
	AutosageAfter    int       `json:"-"`                 // sql: autosage_after
	NoImagesAfter    int       `json:"image_limit"`       // sql: no_images_after
	MaxMessageLength int       `json:"max_comment_chars"` // sql: max_message_length
	MinMessageLength int       `json:"min_comment_chars"` // sql: min_message_length
	AllowEmbeds      bool      `json:"-"`                 // sql: allow_embeds
	RedirectToThread bool      `json:"-"`                 // sql: redirect_to_thread
	RequireFile      bool      `json:"-"`                 // sql: require_file
	EnableCatalog    bool      `json:"-"`                 // sql: enable_catalog
}

// Filter represents an entry in gochan's new filter system which merges username bans, file bans, and filename bans,
// and will allow moderators to block posts based on the user's name, email, subject, message content, and other fields.
// table: DBPREFIXfilters
type Filter struct {
	ID          int       // sql: id
	StaffID     *int      // sql: staff_id
	StaffNote   string    // sql: staff_note
	IssuedAt    time.Time // sql: issued_at
	MatchAction string    // sql: match_action
	MatchDetail string    // sql: match_detail
	HandleIfAny bool      // sql: handle_if_any
	IsActive    bool      // sql: is_active
	conditions  []FilterCondition
}

// FilterCondition represents a condition to be checked against when a post is submitted
// table: DBPREFIXfilter_conditions
type FilterCondition struct {
	ID        int             // sql: id
	FilterID  int             // sql: filter_id
	MatchMode StringMatchMode // sql: match_mode
	Search    string          // sql: search
	Field     string          // sql: field
}

func (fc FilterCondition) insert(ctx context.Context, tx *sql.Tx) error {
	_, err := ExecContextSQL(ctx, tx,
		`INSERT INTO DBPREFIXfilter_conditions(filter_id, match_mode, search, field) VALUES(?,?,?,?)`,
		fc.FilterID, fc.MatchMode, fc.Search, fc.Field,
	)
	return err
}

// FilterHit represents a match from a post filter to an attempted post
// table: DBPREFIXfilter_hits
type FilterHit struct {
	ID        int       // sql: id
	FilterID  int       // sql: filter_id
	PostData  string    // sql: post_data
	MatchTime time.Time // sql: match_time
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

// IsEmbed returns true if the upload is an embed
func (u *Upload) IsEmbed() bool {
	return strings.HasPrefix(u.Filename, "embed:")
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
	IPBanID   int       // sql: ip_ban_id
	Timestamp time.Time // sql: timestamp
	IPBanBase
}

// used to composition IPBanAppeal and IPBanAppealAudit
type ipBanAppealBase struct {
	StaffID       int    // sql: staff_id
	AppealText    string // sql: appeal_text
	StaffResponse string // sql: staff_response
	IsDenied      bool   // sql: is_denied
}

// table: DBPREFIXip_ban_appeals
type IPBanAppeal struct {
	ID      int // sql: id
	IPBanID int // sql: ip_ban_id
	ipBanAppealBase
}

// table: DBPREFIXip_ban_appeals_audit
type IPBanAppealAudit struct {
	AppealID  int       // sql: appeal_id
	Timestamp time.Time // sql: timestamp
	ipBanAppealBase
}

// table: DBPREFIXposts
type Post struct {
	ID               int           `json:"no"`    // sql: id
	ThreadID         int           `json:"-"`     // sql: thread_id
	IsTopPost        bool          `json:"-"`     // sql: is_top_post
	IP               string        `json:"-"`     // sql: ip
	CreatedOn        time.Time     `json:"time"`  // sql: created_on
	Name             string        `json:"name"`  // sql: name
	Tripcode         string        `json:"trip"`  // sql: tripcode
	IsSecureTripcode bool          `json:"-"`     // sql: is_secure_tripcode
	IsRoleSignature  bool          `json:"-"`     // sql: is_role_signature
	Email            string        `json:"email"` // sql: email
	Subject          string        `json:"sub"`   // sql: subject
	Message          template.HTML `json:"-"`     // sql: message
	MessageRaw       string        `json:"com"`   // sql: message_raw
	Password         string        `json:"-"`     // sql: `password`
	DeletedAt        time.Time     `json:"-"`     // sql: deleted_at
	IsDeleted        bool          `json:"-"`     // sql: is_deleted
	BannedMessage    string        `json:"-"`     // sql: banned_message
	Flag             string        `json:"-"`     // sql: flag
	Country          string        `json:"-"`     // sql: country

	// used for convenience to avoid needing to do multiple queries
	opID     int
	boardDir string
}

// table: DBPREFIXreports
type Report struct {
	ID               int    `json:"id"`         // sql: id
	HandledByStaffID *int   `json:"staff_id"`   // sql: handled_by_staff_id
	PostID           int    `json:"post_id"`    // sql: post_id
	IP               string `json:"ip"`         // sql: ip
	Reason           string `json:"reason"`     // sql: reason
	IsCleared        bool   `json:"is_cleared"` // sql: is_cleared
}

// table: DBPREFIXreports_audit
type ReportAudit struct {
	Report           int       // sql: report_id
	Timestamp        time.Time // sql: timestamp
	HandledByStaffID int       // sql: handled_by_staff_id
	IsCleared        bool      // sql: is_cleared
}

// table: DBPREFIXsections
type Section struct {
	ID           int    // sql: id
	Name         string // sql: name
	Abbreviation string // sql: abbreviation
	Position     int    // sql: position
	Hidden       bool   // sql: hidden
}

// table: DBPREFIXsessions
type LoginSession struct {
	ID      int       // sql: id
	StaffID int       // sql: staff_id
	Expires time.Time // sql: expires
	Data    string    // sql: data
}

// table: DBPREFIXstaff
type Staff struct {
	ID               int       // sql: id
	Username         string    // sql: username
	PasswordChecksum string    `json:"-"` // sql: password_checksum
	Rank             int       // sql: global_rank
	AddedOn          time.Time `json:"-"` // sql: added_on
	LastLogin        time.Time `json:"-"` // sql: last_login
	IsActive         bool      `json:"-"` // sql: is_active
}

// table: DBPREFIXthreads
type Thread struct {
	ID          int       // sql: id
	BoardID     int       // sql: board_id
	Locked      bool      // sql: locked
	Stickied    bool      // sql: stickied
	Anchored    bool      // sql: anchored
	Cyclical    bool      // sql: cyclical
	IsSpoilered bool      // sql: is_spoilered
	LastBump    time.Time // sql: last_bump
	DeletedAt   time.Time // sql: deleted_at
	IsDeleted   bool      // sql: is_deleted
}

// Wordfilter is used for filters that are expected to have a single FilterCondition and a "replace" MatchAction
type Wordfilter struct {
	Filter
}
