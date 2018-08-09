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

// PollResultsTable may or may not be used in the future for user polls
type PollResultsTable struct {
	ID        uint
	IP        string
	Selection string
	Timestamp time.Time
}

// PostTable represents each post in the database
type PostTable struct {
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

	Lockdown        bool     `description:"Disables posting." default:"unchecked"`
	LockdownMessage string   `description:"Message displayed when someone tries to post while the site is on lockdown."`
	Sillytags       []string `description:"List of randomly selected staff tags separated by line, e.g. <span style=\"color: red;\">## Mod</span>, to be randomly assigned to posts if UseSillytags is checked. Don't include the \"## \""`
	UseSillytags    bool     `description:"Use Sillytags" default:"unchecked"`
	Modboard        string   `description:"A super secret clubhouse board that only staff can view/post to." default:"staff"`

	SiteName      string `description:"The name of the site that appears in the header of the front page." default:"Gochan"`
	SiteSlogan    string `description:"The text that appears below SiteName on the home page"`
	SiteHeaderURL string `description:"To be honest, I'm not even sure what this does. It'll probably be removed later."`
	SiteWebfolder string `description:"The HTTP root appearing in the browser (e.g. https://gochan.org/&lt;SiteWebFolder&gt;" default:"/"`
	SiteDomain    string `description:"The server's domain (duh). Do not edit this unless you know what you are doing or BAD THINGS WILL HAPPEN!" default:"127.0.0.1" critical:"true"`
	DomainRegex   string `description:"Regular expression used for incoming request validation. Do not edit this unless you know what you are doing or BAD THINGS WILL HAPPEN!" default:"(https|http):\\\\/\\\\/(gochan\\\\.lunachan\\.net|gochan\\\\.org)\\/(.*)" critical:"true"`

	Styles       []string `description:"List of styles (one per line) that should be accessed online at /&lt;SiteWebFolder&gt;/css/&lt;Style&gt;/"`
	DefaultStyle string   `description:"Style used by default (duh). This should appear in the list above or bad things might happen."`

	AllowDuplicateImages bool     `description:"Disabling this will cause gochan to reject a post if the image has already been uploaded for another post.<br />This may end up being removed or being made board-specific in the future." default:"checked"`
	AllowVideoUploads    bool     `description:"Allows users to upload .webm videos. <br />This may end up being removed or being made board-specific in the future."`
	NewThreadDelay       int      `description:"The amount of time in seconds that is required before an IP can make a new thread.<br />This may end up being removed or being made board-specific in the future." default:"30"`
	ReplyDelay           int      `description:"Same as the above, but for replies." default:"7"`
	MaxLineLength        int      `description:"Any line in a post that exceeds this will be split into two (or more) lines.<br />I'm not really sure why this is here, so it may end up being removed." default:"150"`
	ReservedTrips        []string `description:"Secure tripcodes (!!Something) can be reserved here.<br />Each reservation should go on its own line and should look like this:<br />TripPassword1##Tripcode1<br />TripPassword2##Tripcode2"`

	ThumbWidth          int `description:"OP thumbnails use this as their max width.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbHeight         int `description:"OP thumbnails use this as their max height.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbWidth_reply    int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbHeight_reply   int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbWidth_catalog  int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"125"`
	ThumbHeight_catalog int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"125"`

	ThreadsPerPage           int      `default:"15"`
	PostsPerThreadPage       int      `description:"Max number of replies to a thread to show on each thread page." default:"50"`
	RepliesOnBoardPage       int      `description:"Number of replies to a thread to show on the board page." default:"3"`
	StickyRepliesOnBoardPage int      `description:"Same as above for stickied threads." default:"1"`
	BanColors                []string `description:"Colors to be used for public ban messages (e.g. USER WAS BANNED FOR THIS POST).<br />Each entry should be on its own line, and should look something like this:<br />username1:#FF0000<br />username2:#FAF00F<br />username3:blue<br />Invalid entries/nonexistent usernames will show a warning and use the default red."`
	BanMsg                   string   `description:"The default public ban message." default:"USER WAS BANNED FOR THIS POST"`
	EmbedWidth               int      `description:"The width for inline/expanded webm videos." default:"200"`
	EmbedHeight              int      `description:"The height for inline/expanded webm videos." default:"164"`
	ExpandButton             bool     `description:"If checked, adds [Embed] after a Youtube, Vimeo, etc link to toggle an inline video frame." default:"checked"`
	ImagesOpenNewTab         bool     `description:"If checked, thumbnails will open the respective image/video in a new tab instead of expanding them." default:"unchecked"`
	MakeURLsHyperlinked      bool     `description:"If checked, URLs in posts will be turned into a hyperlink. If unchecked, ExpandButton and NewTabOnOutlinks are ignored." default:"checked"`
	NewTabOnOutlinks         bool     `description:"If checked, links to external sites will open in a new tab." default:"checked"`
	EnableQuickReply         bool     `description:"If checked, an optional quick reply box is used. This may end up being removed." default:"checked"`

	DateTimeFormat  string `description:"The format used for dates. See <a href=\"https://golang.org/pkg/time/#Time.Format\">here</a> for more info."`
	AkismetAPIKey   string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
	EnableGeoIP     bool   `description:"If checked, this enables the usage of GeoIP for posts." default:"checked"`
	GeoIPDBlocation string `description:"Specifies the location of the GeoIP database file. If you're using CloudFlare, you can set it to cf to rely on CloudFlare for GeoIP information." default:"/usr/share/GeoIP/GeoIP.dat"`
	MaxRecentPosts  int    `description:"The maximum number of posts to show on the Recent Posts list on the front page." default:"3"`
	// Verbosity = 0 for no debugging info. Critical errors and general output only
	// Verbosity = 1 for non-critical warnings and important info
	// Verbosity = 2 for all debugging/benchmarks/warnings
	Verbosity     int    `description:"The level of verbosity to use in error/warning messages. 0 = critical errors/startup messages, 1 = warnings, 2 = benchmarks/notices." default:"0"`
	EnableAppeals bool   `description:"If checked, allow banned users to appeal their bans.<br />This will likely be removed (permanently allowing appeals) or made board-specific in the future." default:"checked"`
	MaxLogDays    int    `description:"The maximum number of days to keep messages in the moderation/staff log file."`
	RandomSeed    string `critical:"true"`
	Version       string `critical:"true"`
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

	accessLogFile, err := os.OpenFile(path.Join(config.LogDir, "access.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		println(0, "Couldn't open access log. Returned error: "+err.Error())
		os.Exit(1)
	} else {
		accessLog = log.New(accessLogFile, "", log.Ltime|log.Ldate)

	}

	errorLogFile, err := os.OpenFile(path.Join(config.LogDir, "error.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
	if err != nil {
		println(0, "Couldn't open error log. Returned error: "+err.Error())
		os.Exit(1)
	} else {
		errorLog = log.New(errorLogFile, "", log.Ltime|log.Ldate)
	}

	modLogFile, err := os.OpenFile(path.Join(config.LogDir, "mod.log"), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0777)
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

	if config.Styles == nil {
		println(0, "Styles not set in gochan.json, halting.")
		os.Exit(2)
	}

	if config.DefaultStyle == "" {
		config.DefaultStyle = config.Styles[0]
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

	if config.ThreadsPerPage == 0 {
		config.ThreadsPerPage = 10
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

	if config.MaxLogDays == 0 {
		config.MaxLogDays = 15
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
