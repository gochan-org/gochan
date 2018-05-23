package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"github.com/frustra/bbcode"
)

var (
	config        GochanConfig
	accessLog     *log.Logger
	errorLog      *log.Logger
	modLog        *log.Logger
	readBannedIPs []string
	bbcompiler    bbcode.Compiler
)

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

type Thread struct {
	IName         string
	OP            PostTable
	NumReplies    int
	NumImages     int
	OmittedImages int
	BoardReplies  []PostTable
	Stickied      bool
	ThreadPage    int
}

// SQL Table structs

type AnnouncementsTable struct {
	ID        uint
	Subject   string
	Message   string
	Poster    string
	Timestamp time.Time
}

type BanlistTable struct {
	ID            uint
	AllowRead     bool
	IP            string
	Name          string
	Tripcode      string
	Message       string
	SilentBan     uint8
	Boards        string
	BannedBy      string
	Timestamp     time.Time
	Expires       time.Time
	Reason        string
	StaffNote     string
	AppealMessage string
	AppealAt      time.Time
}

type BannedHashesTable struct {
	ID          uint
	Checksum    string
	Description string
}

type BoardsTable struct {
	IName                  string
	ID                     int
	CurrentPage            int
	NumPages               int
	Order                  int
	Dir                    string
	Type                   int
	UploadType             int
	Title                  string
	Subtitle               string
	Description            string
	Section                int
	MaxImageSize           int
	MaxPages               int
	Locale                 string
	DefaultStyle           string
	Locked                 bool
	CreatedOn              time.Time
	Anonymous              string
	ForcedAnon             bool
	MaxAge                 int
	AutosageAfter          int
	NoImagesAfter          int
	MaxMessageLength       int
	EmbedsAllowed          bool
	RedirectToThread       bool
	ShowID                 bool
	RequireFile            bool
	EnableCatalog          bool
	EnableSpoileredImages  bool
	EnableSpoileredThreads bool
	EnableNSFW             bool
	ThreadPage             int
}

type BoardSectionsTable struct {
	IName        string
	ID           int
	Order        int
	Hidden       bool
	Name         string
	Abbreviation string
}

// EmbedsTable represents the embedable media on different sites.
// It's held over from Kusaba X and may be removed in the future
type EmbedsTable struct {
	ID        uint8
	Filetype  string
	Name      string
	URL       string
	Width     uint16
	Height    uint16
	EmbedCode string
}

// FiletypesTable represents the allowed filetypes
// It's held over from Kusaba X and may be removed in the future
type FiletypesTable struct {
	ID         uint8
	Filetype   string
	Mime       string
	ThumbImage string
	ImageW     uint
	ImageH     uint
}

// FrontTable represents the information (News, rules, etc) on the front page
type FrontTable struct {
	IName     string
	ID        int
	Page      int
	Order     int
	Subject   string
	Message   string
	Timestamp time.Time
	Poster    string
	Email     string
}

// FrontLinksTable is used for linking to sites that the admin linkes
type FrontLinksTable struct {
	ID    uint8
	Title string
	URL   string
}

type LoginAttemptsTable struct {
	ID        uint
	IP        string
	Timestamp time.Time
}

type ModLogTable struct {
	IP        uint
	Entry     string
	User      string
	Category  uint8
	Timestamp time.Time
}

// PollResultsTable may or may not be used in the future for polls (duh)
type PollResultsTable struct {
	ID        uint
	IP        string
	Selection string
	Timestamp time.Time
}

// PostTable represents each post in the database
type PostTable struct {
	IName            string
	ID               int
	CurrentPage      int
	NumPages         int
	BoardID          int
	ParentID         int
	Name             string
	Tripcode         string
	Email            string
	Subject          string
	MessageHTML      string
	MessageText      string
	Password         string
	Filename         string
	FilenameOriginal string
	FileChecksum     string
	Filesize         int
	ImageW           int
	ImageH           int
	ThumbW           int
	ThumbH           int
	IP               string
	Tag              string
	Timestamp        time.Time
	Autosage         int
	PosterAuthority  int
	DeletedTimestamp time.Time
	Bumped           time.Time
	Stickied         bool
	Locked           bool
	Reviewed         bool
	Sillytag         bool
}

type ReportsTable struct {
	ID        uint
	Board     string
	PostID    uint
	Timestamp time.Time
	IP        string
	Reason    string
	Cleared   bool
	IsTemp    bool
}

type SessionsTable struct {
	ID      uint
	Data    string
	Expires string
}

// StaffTable represents a single staff member's info stored in the database
type StaffTable struct {
	ID               int
	Username         string
	PasswordChecksum string
	Salt             string
	Rank             int
	Boards           string
	AddedOn          time.Time
	LastActive       time.Time
}

type WordFiltersTable struct {
	ID     int
	From   string
	To     string
	Boards string
	RegEx  bool
}

// Types for the JSON files we generate as a sort of "API"
type BoardJSONWrapper struct {
	Boards []BoardJSON `json:"boards"`
}

type BoardJSON struct {
	BoardName        string         `json:"board"`
	Title            string         `json:"title"`
	WorkSafeBoard    int            `json:"ws_board"`
	ThreadsPerPage   int            `json:"per_page"`
	Pages            int            `json:"pages"`
	MaxFilesize      int            `json:"max_filesize"`
	MaxMessageLength int            `json:"max_comment_chars"`
	BumpLimit        int            `json:"bump_limit"`
	ImageLimit       int            `json:"image_limit"`
	Cooldowns        BoardCooldowns `json:"cooldowns"`
	Description      string         `json:"meta_description"`
	IsArchived       int            `json:"is_archived"`
}

type BoardCooldowns struct {
	NewThread  int `json:"threads"`
	Reply      int `json:"replies"`
	ImageReply int `json:"images"`
}

type ThreadJSONWrapper struct {
	Posts []PostJSON `json:"posts"`
}

type PostJSON struct {
	ID           int    `json:"no"`
	ParentID     int    `json:"resto"`
	Subject      string `json:"sub"`
	Message      string `json:"com"`
	Name         string `json:"name"`
	Tripcode     string `json:"trip"`
	Timestamp    int64  `json:"time"`
	Bumped       int64  `json:"last_modified"`
	ThumbWidth   int    `json:"tn_w"`
	ThumbHeight  int    `json:"tn_h"`
	ImageWidth   int    `json:"w"`
	ImageHeight  int    `json:"h"`
	FileSize     int    `json:"fsize"`
	OrigFilename string `json:"filename"`
	Extension    string `json:"ext"`
	Filename     string `json:"tim"`
	FileChecksum string `json:"md5"`
}

type BoardPageJSON struct {
	Threads []ThreadJSON `json:"threads"`
	Page    int          `json:"page"`
}

type ThreadJSON struct {
	*PostJSON
	OmittedPosts    int `json:"omitted_posts"`
	OmittedImages   int `json:"omitted_images"`
	Replies         int `json:"replies"`
	ImagesOnArchive int `json:"images"`
	Sticky          int `json:"sticky"`
	Locked          int `json:"locked"`
}

// GochanConfig stores crucial info and is read from/written to gochan.json
type GochanConfig struct {
	IName        string //used by our template parser
	ListenIP     string
	Port         int
	FirstPage    []string
	Error404Path string
	Error500Path string
	Username     string
	UseFastCGI   bool

	DocumentRoot string
	TemplateDir  string
	LogDir       string

	DBtype      string
	DBhost      string
	DBname      string
	DBusername  string
	DBpassword  string
	DBprefix    string
	DBkeepalive bool

	Lockdown        bool
	LockdownMessage string
	Sillytags       []string
	UseSillytags    bool
	Modboard        string

	SiteName      string
	SiteSlogan    string
	SiteHeaderURL string
	SiteWebfolder string
	SiteDomain    string
	DomainRegex   string

	Styles_img       []string
	DefaultStyle_img string
	Styles_txt       []string
	DefaultStyle_txt string

	AllowDuplicateImages bool
	AllowVideoUploads    bool
	NewThreadDelay       int
	ReplyDelay           int
	MaxLineLength        int
	ReservedTrips        []interface{}

	ThumbWidth          int
	ThumbHeight         int
	ThumbWidth_reply    int
	ThumbHeight_reply   int
	ThumbWidth_catalog  int
	ThumbHeight_catalog int

	ThreadsPerPage_img       int
	ThreadsPerPage_txt       int
	PostsPerThreadPage       int
	RepliesOnBoardPage       int
	StickyRepliesOnBoardPage int
	BanColors                []interface{}
	BanMsg                   string
	EmbedWidth               int
	EmbedHeight              int
	ExpandButton             bool
	ImagesOpenNewTab         bool
	MakeURLsHyperlinked      bool
	NewTabOnOutlinks         bool
	EnableQuickReply         bool

	DateTimeFormat   string
	DefaultBanReason string
	AkismetAPIKey    string
	EnableGeoIP      bool
	GeoIPDBlocation  string // set to "cf" or the path to the db
	MaxRecentPosts   int
	MakeRSS          bool
	MakeSitemap      bool
	EnableAppeals    bool
	MaxModlogDays    int
	RandomSeed       string
	Version          string
	Verbosity        int
}

func initConfig() {
	jfile, err := ioutil.ReadFile("gochan.json")
	if err != nil {
		printf(0, "Error reading \"gochan.json\": %s\n", err.Error())
		os.Exit(2)
	}

	if err = json.Unmarshal(jfile, &config); err != nil {
		printf(0, "Error parsing \"gochan.json\": %s\n", err.Error())
		os.Exit(2)
	}

	config.IName = "GochanConfig"
	if config.ListenIP == "" {
		println(0, "ListenIP not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.Port == 0 {
		config.Port = 80
	}

	if len(config.FirstPage) == 0 {
		config.FirstPage = []string{"index.html", "board.html"}
	}

	if config.Error404Path == "" {
		println(0, "Error404Path not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.Error500Path == "" {
		println(0, "Error500Path not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.Username == "" {
		config.Username = "gochan"
	}

	if config.DocumentRoot == "" {
		println(0, "DocumentRoot not set in gochan.json, halting.")
		os.Exit(2)
	}

	wd, wderr := os.Getwd()
	if wderr == nil {
		_, staterr := os.Stat(path.Join(wd, config.DocumentRoot, "css"))
		if staterr == nil {
			config.DocumentRoot = path.Join(wd, config.DocumentRoot)
		}
	}

	if config.TemplateDir == "" {
		println(0, "TemplateDir not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.LogDir == "" {
		println(0, "LogDir not set in gochan.json, halting.")
		os.Exit(2)
	}

	accessLogFile, err := os.OpenFile(path.Join(config.LogDir, "access.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		println(0, "Couldn't open access log. Returned error: "+err.Error())
		os.Exit(1)
	} else {
		accessLog = log.New(accessLogFile, "", log.Ltime|log.Ldate)

	}

	errorLogFile, err := os.OpenFile(path.Join(config.LogDir, "error.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		println(0, "Couldn't open error log. Returned error: "+err.Error())
		os.Exit(1)
	} else {
		errorLog = log.New(errorLogFile, "", log.Ltime|log.Ldate)
	}

	modLogFile, err := os.OpenFile(path.Join(config.LogDir, "mod.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		println(0, "Couldn't open mod log. Returned error: "+err.Error())
	} else {
		modLog = log.New(modLogFile, "", log.Ltime|log.Ldate)
	}

	if config.DBtype == "" {
		println(0, "DBtype not set in gochan.json, halting (currently supported values: mysql).")
		os.Exit(2)
	}

	if config.DBhost == "" {
		println(0, "DBhost not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DBname == "" {
		config.DBname = "gochan"
	}

	if config.DBusername == "" {
		config.DBusername = "gochan"
	}

	if config.DBpassword == "" {
		println(0, "DBpassword not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DBprefix == "" {
		config.DBprefix = "gc_"
	} else {
		config.DBprefix += "_"
	}

	if config.LockdownMessage == "" {
		config.LockdownMessage = "This imageboard has temporarily disabled posting. We apologize for the inconvenience"
	}

	if config.Modboard == "" {
		config.Modboard = "staff"
	}

	if config.SiteName == "" {
		config.SiteName = "An unnamed imageboard"
	}

	if config.SiteDomain == "" {
		println(0, "SiteDomain not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.SiteWebfolder == "" {
		println(0, "SiteWebfolder not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DomainRegex == "" {
		println(0, "DomainRegex not set in gochan.json, consider using (https|http):\\/\\/("+config.SiteDomain+")\\/(.*)")
		println(0, "This should work in most cases. Halting")
		os.Exit(2)
		//config.DomainRegex = "(https|http):\\/\\/(" + config.SiteDomain + ")\\/(.*)"
	}

	if config.Styles_img == nil {
		println(0, "Styles_img not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DefaultStyle_img == "" {
		config.DefaultStyle_img = config.Styles_img[0]
	}

	if config.Styles_txt == nil {
		println(0, "Styles_txt not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DefaultStyle_txt == "" {
		config.DefaultStyle_txt = config.Styles_txt[0]
	}

	if config.NewThreadDelay == 0 {
		config.NewThreadDelay = 30
	}

	if config.ReplyDelay == 0 {
		config.ReplyDelay = 7
	}

	if config.MaxLineLength == 0 {
		config.MaxLineLength = 150
	}

	//ReservedTrips string //eventually this will be map[string]string

	if config.ThumbWidth == 0 {
		config.ThumbWidth = 200
	}

	if config.ThumbHeight == 0 {
		config.ThumbHeight = 200
	}

	if config.ThumbWidth_reply == 0 {
		config.ThumbWidth_reply = 125
	}

	if config.ThumbHeight_reply == 0 {
		config.ThumbHeight_reply = 125
	}

	if config.ThumbWidth_catalog == 0 {
		config.ThumbWidth_catalog = 50
	}

	if config.ThumbHeight_catalog == 0 {
		config.ThumbHeight_catalog = 50
	}

	if config.ThreadsPerPage_img == 0 {
		config.ThreadsPerPage_img = 10
	}

	if config.ThreadsPerPage_txt == 0 {
		config.ThreadsPerPage_txt = 15
	}

	if config.PostsPerThreadPage == 0 {
		config.PostsPerThreadPage = 4
	}

	if config.RepliesOnBoardPage == 0 {
		config.PostsPerThreadPage = 3
	}

	if config.StickyRepliesOnBoardPage == 0 {
		config.StickyRepliesOnBoardPage = 1
	}

	/*config.BanColors, err = c.GetString("threads", "ban_colors") //eventually this will be map[string] string
	if err != nil {
		config.BanColors = "admin:#CC0000"
	}*/

	if config.BanMsg == "" {
		config.BanMsg = "(USER WAS BANNED FOR THIS POST)"
	}

	if config.DateTimeFormat == "" {
		config.DateTimeFormat = "Mon, January 02, 2006 15:04 PM"
	}

	if config.EnableGeoIP {
		if config.GeoIPDBlocation == "" {
			println(0, "GeoIPDBlocation not set in gochan.json, disabling EnableGeoIP.")
			config.EnableGeoIP = false
		}
	}

	if config.MaxRecentPosts == 0 {
		config.MaxRecentPosts = 10
	}

	if config.MaxModlogDays == 0 {
		config.MaxModlogDays = 15
	}

	if config.RandomSeed == "" {
		println(0, "RandomSeed not set in gochan.json, halting.")
		os.Exit(2)
	}
	bbcompiler = bbcode.NewCompiler(true, true)
	bbcompiler.SetTag("center", nil)
	bbcompiler.SetTag("code", nil)
	bbcompiler.SetTag("color", nil)
	bbcompiler.SetTag("img", nil)
	bbcompiler.SetTag("quote", nil)
	bbcompiler.SetTag("size", nil)

	config.Version = version
}
