package gcsql

import (
	"fmt"
	"html"
	"html/template"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	_ = iota
	threadBan
	imageBan
	fullBan
)

var (
	// AllSections is a cached list of all of the board sections
	AllSections []BoardSection
	// AllBoards is a cached list of all of the boards
	AllBoards []Board
	// TempPosts is a cached list of all of the posts in the temporary posts table
	TempPosts []Post
)

type Announcement struct {
	ID        uint   `json:"no"`
	Subject   string `json:"sub"`
	Message   string `json:"com"`
	Poster    string `json:"name"`
	Timestamp time.Time
}

type BanAppeal struct {
	ID            int
	Ban           int
	Message       string
	Denied        bool
	StaffResponse string
}

type BanInfo struct {
	ID          uint
	IP          string
	Name        string
	NameIsRegex bool
	Boards      string
	Staff       string
	Timestamp   time.Time
	Expires     time.Time
	Permaban    bool
	Reason      string
	Type        int
	StaffNote   string
	AppealAt    time.Time
	CanAppeal   bool
}

// BannedForever returns true if the ban is an unappealable permaban
func (ban *BanInfo) BannedForever() bool {
	return ban.Permaban && !ban.CanAppeal && ban.Type == fullBan && ban.Boards == ""
}

// IsActive returns true if the ban is still active (unexpired or a permaban)
func (ban *BanInfo) IsActive(board string) bool {
	if ban.Boards == "" && (ban.Expires.After(time.Now()) || ban.Permaban) {
		return true
	}
	boardsArr := strings.Split(ban.Boards, ",")
	for _, b := range boardsArr {
		if b == board && (ban.Expires.After(time.Now()) || ban.Permaban) {
			return true
		}
	}

	return false
}

// IsBanned checks to see if the ban applies to the given board
func (ban *BanInfo) IsBanned(board string) bool {
	if ban.Boards == "" && (ban.Expires.After(time.Now()) || ban.Permaban) {
		return true
	}
	boardsArr := strings.Split(ban.Boards, ",")
	for _, b := range boardsArr {
		if b == board && (ban.Expires.After(time.Now()) || ban.Permaban) {
			return true
		}
	}

	return false
}

type BannedHash struct {
	ID          uint
	Checksum    string
	Description string
}

type Board struct {
	ID                     int            `json:"-"`
	CurrentPage            int            `json:"-"`
	NumPages               int            `json:"pages"`
	ListOrder              int            `json:"-"`
	Dir                    string         `json:"board"`
	Type                   int            `json:"-"`
	UploadType             int            `json:"-"`
	Title                  string         `json:"title"`
	Subtitle               string         `json:"meta_description"`
	Description            string         `json:"-"`
	Section                int            `json:"-"`
	MaxFilesize            int            `json:"max_filesize"`
	MaxPages               int            `json:"max_pages"`
	DefaultStyle           string         `json:"-"`
	Locked                 bool           `json:"is_archived"`
	CreatedOn              time.Time      `json:"-"`
	Anonymous              string         `json:"-"`
	ForcedAnon             bool           `json:"-"`
	MaxAge                 int            `json:"-"`
	AutosageAfter          int            `json:"bump_limit"`
	NoImagesAfter          int            `json:"image_limit"`
	MaxMessageLength       int            `json:"max_comment_chars"`
	EmbedsAllowed          bool           `json:"-"`
	RedirectToThread       bool           `json:"-"`
	ShowID                 bool           `json:"-"`
	RequireFile            bool           `json:"-"`
	EnableCatalog          bool           `json:"-"`
	EnableSpoileredImages  bool           `json:"-"`
	EnableSpoileredThreads bool           `json:"-"`
	Worksafe               bool           `json:"ws_board"`
	ThreadPage             int            `json:"-"`
	Cooldowns              BoardCooldowns `json:"cooldowns"`
	ThreadsPerPage         int            `json:"per_page"`
}

// AbsolutePath returns the full filepath of the board directory
func (board *Board) AbsolutePath(subpath ...string) string {
	return path.Join(config.GetSystemCriticalConfig().DocumentRoot, board.Dir, path.Join(subpath...))
}

// WebPath returns a string that represents the file's path as accessible by a browser
// fileType should be "boardPage", "threadPage", "upload", or "thumb"
func (board *Board) WebPath(fileName, fileType string) string {
	var filePath string
	systemCritical := config.GetSystemCriticalConfig()

	switch fileType {
	case "":
		fallthrough
	case "boardPage":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, fileName)
	case "threadPage":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "res", fileName)
	case "upload":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "src", fileName)
	case "thumb":
		filePath = path.Join(systemCritical.WebRoot, board.Dir, "thumb", fileName)
	}
	return filePath
}

func (board *Board) PagePath(pageNum interface{}) string {
	var page string
	pageNumStr := fmt.Sprintf("%v", pageNum)
	if pageNumStr == "prev" {
		if board.CurrentPage < 2 {
			page = "1"
		} else {
			page = strconv.Itoa(board.CurrentPage - 1)
		}
	} else if pageNumStr == "next" {
		if board.CurrentPage >= board.NumPages {
			page = strconv.Itoa(board.NumPages)
		} else {
			page = strconv.Itoa(board.CurrentPage + 1)
		}
	} else {
		page = pageNumStr
	}
	return board.WebPath(page+".html", "boardPage")
}

func (board *Board) SetDefaults(title string, subtitle string, description string) {
	board.CurrentPage = 1
	board.NumPages = 15
	board.ListOrder = 0
	board.Dir = "test"
	board.Type = 0
	board.UploadType = 0
	if title == "" {
		board.Title = "Testing board"
	} else {
		board.Title = title
	}
	if subtitle == "" {
		board.Subtitle = "Board for testing stuff"
	} else {
		board.Subtitle = subtitle
	}
	if description == "" {
		board.Description = "/test/ board description"
	} else {
		board.Description = description
	}
	board.Section = 1
	board.MaxFilesize = 10000
	board.MaxPages = 21
	board.DefaultStyle = config.GetBoardConfig("").DefaultStyle
	board.Locked = false
	board.CreatedOn = time.Now()
	board.Anonymous = "Anonymous"
	board.ForcedAnon = false
	board.AutosageAfter = 200
	board.NoImagesAfter = 500
	board.MaxMessageLength = 8192
	board.EmbedsAllowed = false
	board.RedirectToThread = false
	board.ShowID = false
	board.RequireFile = false
	board.EnableCatalog = true
	board.EnableSpoileredImages = true
	board.EnableSpoileredThreads = true
	board.Worksafe = true
	board.Cooldowns = BoardCooldowns{
		NewThread:  30,
		Reply:      7,
		ImageReply: 7,
	}
	board.ThreadsPerPage = 20
}

func (board *Board) Create() error {
	return CreateBoard(board)
}

type BoardSection struct {
	ID           int
	ListOrder    int
	Hidden       bool
	Name         string
	Abbreviation string
}

// Post represents each post in the database
// Deprecated. Struct was made for use with old database, deprecated since refactor of april 2020.
// Please refactor all code that uses this struct to use a struct that alligns with the new database structure and functions.
type Post struct {
	ID               int           `json:"no"`
	ParentID         int           `json:"resto"`
	CurrentPage      int           `json:"-"`
	BoardID          int           `json:"-"`
	Name             string        `json:"name"`
	Tripcode         string        `json:"trip"`
	Email            string        `json:"email"`
	Subject          string        `json:"sub"`
	MessageHTML      template.HTML `json:"com"`
	MessageText      string        `json:"-"`
	Password         string        `json:"-"`
	Filename         string        `json:"tim"`
	FilenameOriginal string        `json:"filename"`
	FileChecksum     string        `json:"md5"`
	FileExt          string        `json:"extension"`
	Filesize         int           `json:"fsize"`
	ImageW           int           `json:"w"`
	ImageH           int           `json:"h"`
	ThumbW           int           `json:"tn_w"`
	ThumbH           int           `json:"tn_h"`
	IP               string        `json:"-"`
	Capcode          string        `json:"capcode"`
	Timestamp        time.Time     `json:"time"`
	Autosage         bool          `json:"-"`
	Bumped           time.Time     `json:"last_modified"`
	Stickied         bool          `json:"-"`
	Locked           bool          `json:"-"`
	Reviewed         bool          `json:"-"`
}

func (p *Post) GetURL(includeDomain bool) string {
	postURL := ""
	systemCritical := config.GetSystemCriticalConfig()
	if includeDomain {
		postURL += systemCritical.SiteDomain
	}
	var board Board
	if err := board.PopulateData(p.BoardID); err != nil {
		return postURL
	}

	postURL += systemCritical.WebRoot + board.Dir + "/res/"
	if p.ParentID == 0 {
		postURL += fmt.Sprintf("%d.html#%d", p.ID, p.ID)
	} else {
		postURL += fmt.Sprintf("%d.html#%d", p.ParentID, p.ID)
	}
	return postURL
}

// Sanitize escapes HTML strings in a post. This should be run immediately before
// the post is inserted into the database
func (p *Post) Sanitize() {
	p.Name = html.EscapeString(p.Name)
	p.Email = html.EscapeString(p.Email)
	p.Subject = html.EscapeString(p.Subject)
	p.Password = html.EscapeString(p.Password)
	if p.ParentID < 0 {
		p.ParentID = 0
	}
}

type Report struct {
	ID        uint
	Board     string
	PostID    uint
	Timestamp time.Time
	IP        string
	Reason    string
	Cleared   bool
	IsTemp    bool
}

type LoginSession struct {
	ID      uint
	Name    string
	Data    string
	Expires string
}

// Staff represents a single staff member's info stored in the database
type Staff struct {
	ID               int
	Username         string
	PasswordChecksum string `json:"-"`
	Rank             int
	AddedOn          time.Time
	LastActive       time.Time
}

// CleanSessions clears out all of the sessions with this
// staff ID, regardless of cookie data
func (s *Staff) CleanSessions() (int64, error) {
	var err error
	query := `DELETE FROM DBPREFIXsessions WHERE staff_id = ?`
	result, err := ExecSQL(query, s.ID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Staff) RankString() string {
	switch s.Rank {
	case 3:
		return "Administrator"
	case 2:
		return "Moderator"
	case 1:
		return "Janitor"
	}
	return ""
}

type BoardCooldowns struct {
	NewThread  int `json:"threads"`
	Reply      int `json:"replies"`
	ImageReply int `json:"images"`
}

type MessagePostContainer struct {
	ID         int
	MessageRaw string
	Message    template.HTML
}

// Deprecated. Struct was made for use with old database, deprecated since refactor of april 2020.
// Please refactor all code that uses this struct to use a struct that alligns with the new database structure and functions.
type RecentPost struct {
	BoardName string
	BoardID   int
	PostID    int
	ParentID  int
	Name      string
	Tripcode  string
	Message   template.HTML
	Filename  string
	ThumbW    int
	ThumbH    int
	IP        string
	Timestamp time.Time
}

// GetURL returns the full URL of the recent post, or the full path if includeDomain is false
func (p *RecentPost) GetURL(includeDomain bool) string {
	postURL := ""
	systemCritical := config.GetSystemCriticalConfig()
	if includeDomain {
		postURL += systemCritical.SiteDomain
	}
	idStr := strconv.Itoa(p.PostID)
	postURL += systemCritical.WebRoot + p.BoardName + "/res/"
	if p.ParentID == 0 {
		postURL += idStr + ".html#" + idStr
	} else {
		postURL += strconv.Itoa(p.ParentID) + ".html#" + idStr
	}
	return postURL
}

type Thread struct {
	OP            Post   `json:"-"`
	NumReplies    int    `json:"replies"`
	NumImages     int    `json:"images"`
	OmittedPosts  int    `json:"omitted_posts"`
	OmittedImages int    `json:"omitted_images"`
	BoardReplies  []Post `json:"-"`
	Sticky        int    `json:"sticky"`
	Locked        int    `json:"locked"`
	ThreadPage    int    `json:"-"`
}

//FileBan contains the information associated with a specific file ban
type FileBan struct {
	ID        int       `json:"id"`
	BoardID   *int      `json:"board"`
	StaffID   int       `json:"staff_id"`
	StaffNote string    `json:"staff_note"`
	IssuedAt  time.Time `json:"issued_at"`
	Checksum  string    `json:"checksum"`
}

//FilenameBan contains the information associated with a specific filename ban
type FilenameBan struct {
	ID        int       `json:"id"`
	BoardID   *int      `json:"board"`
	StaffID   int       `json:"staff_id"`
	StaffNote string    `json:"staff_note"`
	IssuedAt  time.Time `json:"issued_at"`
	Filename  string    `json:"filename"`
	IsRegex   bool      `json:"is_regex"`
}

//UsernameBan contains the information associated with a specific username ban
type UsernameBan struct {
	ID        int       `json:"id"`
	BoardID   *int      `json:"board"`
	StaffID   int       `json:"staff_id"`
	StaffNote string    `json:"staff_note"`
	IssuedAt  time.Time `json:"issued_at"`
	Username  string    `json:"username"`
	IsRegex   bool      `json:"is_regex"`
}

//WordFilter contains the information associated with a specific wordfilter
type WordFilter struct {
	ID        int       `json:"id"`
	BoardDirs []string  `json:"boards"`
	StaffID   int       `json:"staff_id"`
	StaffNote string    `json:"staff_note"`
	IssuedAt  time.Time `json:"issued_at"`
	Search    string    `json:"search"`
	IsRegex   bool      `json:"is_regex"`
	ChangeTo  string    `json:"change_to"`
}

//IPBan contains the information association with a specific ip ban
type IPBan struct {
	ID              int           `json:"id"`
	BoardID         *int          `json:"board"`
	StaffID         int           `json:"staff_id"`
	BannedForPostID *int          `json:"banned_for_post_id"`
	CopyPostText    template.HTML `json:"copy_post_text"`
	IsThreadBan     bool          `json:"is_thread_ban"`
	IsActive        bool          `json:"is_active"`
	IP              string        `json:"ip"`
	IssuedAt        time.Time     `json:"issued_at"`
	AppealAt        time.Time     `json:"appeal_at"`
	ExpiresAt       time.Time     `json:"expires_at"`
	Permanent       bool          `json:"permanent"`
	StaffNote       string        `json:"staff_note"`
	Message         string        `json:"message"`
	CanAppeal       bool          `json:"can_appeal"`
}
