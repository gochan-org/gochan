package gcsql

import (
	"errors"
	"fmt"
	"html"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
)

const (
	dirIsAFileStr = `unable to create "%s", path exists and is a file`
	genericErrStr = `unable to create "%s": %s`
	pathExistsStr = `unable to create "%s", path already exists`

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
	AllowRead   bool
	IP          string
	Name        string
	NameIsRegex bool
	SilentBan   uint8
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
	return path.Join(config.Config.DocumentRoot, board.Dir, path.Join(subpath...))
}

// WebPath returns a string that represents the file's path as accessible by a browser
// fileType should be "boardPage", "threadPage", "upload", or "thumb"
func (board *Board) WebPath(fileName string, fileType string) string {
	var filePath string
	switch fileType {
	case "":
		fallthrough
	case "boardPage":
		filePath = path.Join(config.Config.SiteWebfolder, board.Dir, fileName)
	case "threadPage":
		filePath = path.Join(config.Config.SiteWebfolder, board.Dir, "res", fileName)
	case "upload":
		filePath = path.Join(config.Config.SiteWebfolder, board.Dir, "src", fileName)
	case "thumb":
		filePath = path.Join(config.Config.SiteWebfolder, board.Dir, "thumb", fileName)
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

// Build builds the board and its thread files
// if newBoard is true, it adds a row to DBPREFIXboards and fails if it exists
// if force is true, it doesn't fail if the directories exist but does fail if it is a file
func (board *Board) Build(newBoard bool, force bool) error {
	var err error
	if board.Dir == "" {
		return errors.New("board must have a directory before it is built")
	}
	if board.Title == "" {
		return errors.New("board must have a title before it is built")
	}

	dirPath := board.AbsolutePath()
	resPath := board.AbsolutePath("res")
	srcPath := board.AbsolutePath("src")
	thumbPath := board.AbsolutePath("thumb")
	dirInfo, _ := os.Stat(dirPath)
	resInfo, _ := os.Stat(resPath)
	srcInfo, _ := os.Stat(srcPath)
	thumbInfo, _ := os.Stat(thumbPath)
	if dirInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, dirPath)
		}
		if !dirInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, dirPath)
		}
	} else {
		if err = os.Mkdir(dirPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, dirPath, err.Error())
		}
	}

	if resInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, resPath)
		}
		if !resInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, resPath)
		}
	} else {
		if err = os.Mkdir(resPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, resPath, err.Error())
		}
	}

	if srcInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, srcPath)
		}
		if !srcInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, srcPath)
		}
	} else {
		if err = os.Mkdir(srcPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, srcPath, err.Error())
		}
	}

	if thumbInfo != nil {
		if !force {
			return fmt.Errorf(pathExistsStr, thumbPath)
		}
		if !thumbInfo.IsDir() {
			return fmt.Errorf(dirIsAFileStr, thumbPath)
		}
	} else {
		if err = os.Mkdir(thumbPath, 0666); err != nil {
			return fmt.Errorf(genericErrStr, thumbPath, err.Error())
		}
	}

	if newBoard {
		board.CreatedOn = time.Now()
		err := CreateBoard(board)
		if err != nil {
			return err
		}
	} else {
		if err = board.UpdateID(); err != nil {
			return err
		}
	}
	/* buildBoardPages(board)
	buildThreads(true, board.ID, 0)
	resetBoardSectionArrays()
	buildFrontPage()
	if board.EnableCatalog {
		buildCatalog(board.ID)
	}
	buildBoardListJSON() */
	return nil
}

func (board *Board) SetDefaults() {
	board.ListOrder = 0
	board.Section = 1
	board.MaxFilesize = 4096
	board.MaxPages = 11
	board.DefaultStyle = config.Config.DefaultStyle
	board.Locked = false
	board.Anonymous = "Anonymous"
	board.ForcedAnon = false
	board.MaxAge = 0
	board.AutosageAfter = 200
	board.NoImagesAfter = 0
	board.MaxMessageLength = 8192
	board.EmbedsAllowed = true
	board.RedirectToThread = false
	board.ShowID = false
	board.RequireFile = false
	board.EnableCatalog = true
	board.EnableSpoileredImages = true
	board.EnableSpoileredThreads = true
	board.Worksafe = true
	board.ThreadsPerPage = 10
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
	ID               int       `json:"no"`
	ParentID         int       `json:"resto"`
	CurrentPage      int       `json:"-"`
	BoardID          int       `json:"-"`
	Name             string    `json:"name"`
	Tripcode         string    `json:"trip"`
	Email            string    `json:"email"`
	Subject          string    `json:"sub"`
	MessageHTML      string    `json:"com"`
	MessageText      string    `json:"-"`
	Password         string    `json:"-"`
	Filename         string    `json:"tim"`
	FilenameOriginal string    `json:"filename"`
	FileChecksum     string    `json:"md5"`
	FileExt          string    `json:"extension"`
	Filesize         int       `json:"fsize"`
	ImageW           int       `json:"w"`
	ImageH           int       `json:"h"`
	ThumbW           int       `json:"tn_w"`
	ThumbH           int       `json:"tn_h"`
	IP               string    `json:"-"`
	Capcode          string    `json:"capcode"`
	Timestamp        time.Time `json:"time"`
	Autosage         bool      `json:"-"`
	Bumped           time.Time `json:"last_modified"`
	Stickied         bool      `json:"-"`
	Locked           bool      `json:"-"`
	Reviewed         bool      `json:"-"`
}

func (p *Post) GetURL(includeDomain bool) string {
	postURL := ""
	if includeDomain {
		postURL += config.Config.SiteDomain
	}
	var board Board
	if err := board.PopulateData(p.BoardID); err != nil {
		return postURL
	}

	idStr := strconv.Itoa(p.ID)
	postURL += config.Config.SiteWebfolder + board.Dir + "/res/"
	if p.ParentID == 0 {
		postURL += idStr + ".html#" + idStr
	} else {
		postURL += strconv.Itoa(p.ParentID) + ".html#" + idStr
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
	PasswordChecksum string
	Rank             int
	AddedOn          time.Time
	LastActive       time.Time
}

type WordFilter struct {
	ID     int
	From   string
	To     string
	Boards string
	RegEx  bool
}

type BoardCooldowns struct {
	NewThread  int `json:"threads"`
	Reply      int `json:"replies"`
	ImageReply int `json:"images"`
}

type MessagePostContainer struct {
	ID         int
	MessageRaw string
	Message    string
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
	Message   string
	Filename  string
	ThumbW    int
	ThumbH    int
	IP        string
	Timestamp time.Time
}

// GetURL returns the full URL of the recent post, or the full path if includeDomain is false
func (p *RecentPost) GetURL(includeDomain bool) string {
	postURL := ""
	if includeDomain {
		postURL += config.Config.SiteDomain
	}
	idStr := strconv.Itoa(p.PostID)
	postURL += config.Config.SiteWebfolder + p.BoardName + "/res/"
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
