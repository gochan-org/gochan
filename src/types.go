package main 

import (
	"fmt"
	"os"
	"strings"
	"go-logfile/logfile"
	"goconf/conf"
)


var (
	c *conf.ConfigFile
	needs_initial_setup = true
	config GochanConfig
	access_log logfile.Log
	error_log logfile.Log
)

// SQL Table structs

type AnnouncementsTable struct {
	ID uint
	Subject string
	Message string
	Poster string
	Timestamp string
}

type BanlistTable struct {
	ID uint
	Expired bool
	AllowRead bool
	IP string
	SilentBan uint8
	Boards string
	BannedBy string
	Timestamp string
	Expires string
	Reason string
	StaffNote string
	Appeal string
	AppealAt string
}

type BannedHashesTable struct {
	ID uint
	Checksum string
	Description string
}

type BannedTripcodesTable struct {
	ID uint
	Name string
	Tripcode string
}

type BoardsTable struct {
	IName string
	ID uint8
	Order uint8
	Dir string
	Type uint8
	FirstPost uint
	UploadType uint8
	Title string
	Subtitle string
	Description string
	Section int
	MaxImageSize int
	MaxPages uint8
	Locale string
	DefaultStyle string
	Locked bool
	CreatedOn string
	Anonymous string
	ForcedAnon string
	MaxAge uint
	MarkPage uint8
	AutosageAfter uint
	NoImagesAfter uint
	MaxMessageLength uint
	EmbedsAllowed string
	RedirectToThread bool
	ShowId bool
	CompactList bool
	EnableNofile bool
	EnableCatalog bool
}

type BoardSectionsTable struct {
	IName string
	ID int
	Order int
	Hidden bool
	Name string
	Abbreviation string
}

type EmbedsTable struct {
	ID uint8
	Filetype string
	Name string
	URL string
	Width uint16
	Height uint16
	EmbedCode string
}

type FiletypesTable struct {
	ID uint8
	Filetype string
	Mime string
	ThumbImage string
	ImageW uint
	ImageH uint
}

type FrontTable struct {
	IName string
	ID int
	Page int
	Order int
	Subject string
	Message string
	Timestamp string
	Poster string
	Email string
}

type FrontLinksTable struct {
	ID uint8
	Title string
	URL string
}

type LoginAttemptsTable struct {
	ID uint
	IP string
	Timestamp string
}

type ModLogTable struct {
	IP uint
	Entry string
	User string
	Category uint8
	Timestamp string
}

type PollResultsTable struct {
	ID uint
	IP string
	Selection string
	Timestamp string
}

type PostTable struct {
	IName string
	ID uint
	BoarID uint8
	ParentID uint
	Name string
	Tripcode string
	Email string
	Subject string
	Message string
	Password string
	Filename string
	FilenameOriginal string
	FileChecksum string
	Filesize string
	ImageW uint16
	ImageH uint16
	ThumbW uint16
	ThumbH uint16
	IP string
	Tag string
	Timestamp string
	Autosage uint8
	PosterAuthority uint8
	DeletedTimestamp string
	Bumped string
	Stickied bool
	Locked bool
	Reviewed bool
	Sillytag bool
}

type TempPostTable struct {
	ID uint
	Boarid uint8
	Parentid uint
	Name string
	Tripcode string
	Email string
	Subject string
	Message string
	Password string
	Filename string
	FilenameOriginal string
	FileChecksum string
	Filesize string
	ImageW uint16
	ImageH uint16
	ThumbW uint16
	ThumbH uint16
	IP string
	Tag string
	Timestamp string
	Autosage uint8
	PosterAuthority uint8
	DeletedTimestamp string
	Bumped string
	Stickied bool
	Locked bool
	Reviewed bool
	Sillytag bool
}

type ReportsTable struct {
	ID uint
	Board string
	PostID uint
	Timestamp string
	IP string
	Reason string
	Cleared bool
	IsTemp bool
}

type SessionsTable struct {
	ID uint
	Data string
	Expires string
}

type StaffTable struct {
	ID uint16
	Username string
	PasswordChecksum string
	Salt string
	Rank uint8
	Boards string
	AddedOn string
	LastActive string
}

type WordFiltersTable struct {
	ID uint16
	From string
	To string
	Boards string
	RegEx bool
}

type Wrapper struct {
	IName string
	Data []interface{}
}

// Global variables, most initialized by config.cfg

type GochanConfig struct {
	IName string //used by our template parser
	Domain string
	Port int
	FirstPage []string
	Error404Path string
	Error500Path string
	Username string

	DocumentRoot string
	TemplateDir string
	LogDir string
	
	DBtype string
	DBhost string
	DBname string
	DBusername string
	DBpassword string
	DBprefix string
	DBkeepalive bool

	Lockdown bool
	LockdownMessage string
	Sillytags string
	UseSillytags bool
	Modboard string

	SiteName string
	SiteSlogan string
	SiteHeaderURL string
	SiteWebfolder string
	SiteDomain string

	Styles_img []string
	DefaultStyle_img string
	Styles_txt []string
	DefaultStyle_txt string

	AllowDuplicateImages bool
	NewThreadDelay int
	ReplyDelay int
	MaxLineLength int
	ReservedTrips string //eventually this will be map[string]string

	ThumbWidth int
	ThumbHeight int
	ThumbWidth_reply int
	ThumbHeight_reply int
	ThumbWidth_catalog int
	ThumbHeight_catalog int

	ThreadsPerPage_img int
	ThreadsPerPage_txt int
	PostsPerThreadpage int
	RepliesOnBoardpage int
	GenLast50 bool
	GenFirst100 bool
	StickyRepliesOnBoardPage int
	BanColors string //eventually this will be map[string] string
	BanMsg string
	YoutubeWidth int
	YoutubeHeight int
	ExpandButton bool
	ImagesOpenNewTab bool
	MakeURLsHyperlinked bool
	NewTabOnOutlinks bool
	EnableQuickReply bool

	DefaultBanReason string
	EnableGeoIP bool
	GeoIPDBlocation string // set to "cf" or the path to the db
	MaxRecentPosts int
	MakeRSS bool
	MakeSitemap bool
	EnableAppeals bool
	MaxModlogDays int
	RandomSeed string
	Version float32
}

func initConfig() {
	var err error
	c,err = conf.ReadConfigFile("config.cfg")
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	config.IName = "GochanConfig"
	config.Domain,err = c.GetString("server", "domain")
	if err != nil {
		fmt.Println("server.domain not set in config.cfg, halting.")
	}

	config.Port,err = c.GetInt("server", "port")
	if err != nil {
		config.Port = 80
		fmt.Println("server.port not set in config.cfg, defaulting to 80")
	}

	first_page_str,err_ := c.GetString("server", "first_page")
	if err_ != nil {
		first_page_str = "board.html,index.html"
		fmt.Println("server.first_page not set in config.cfg, defaulting to "+first_page_str)
	}

	config.FirstPage = strings.Split(first_page_str, ",")

	config.Error404Path,err = c.GetString("server", "error_404_path")
	if err != nil {
		config.Error404Path = "/error/404.html"
		fmt.Println("server.error_404_path not set in config.cfg, defaulting to "+config.Error404Path)
	}
	config.Error500Path,err = c.GetString("server", "error_500_path")
	if err != nil {
		config.Error500Path = "/error/500.html"
		fmt.Println("server.error_500_path not set in config.cfg, defaulting to "+config.Error500Path)
	}

	config.Username,err = c.GetString("server", "username")
	if err != nil {
		config.Username = "gochan"
		fmt.Println("server.username not set in config.cfg, defaulting to "+config.Username)
	}

	config.DocumentRoot,err = c.GetString("directories", "document_root")
	if err != nil {
		fmt.Println("directories.document_root not set in config.cfg, halting.")
		os.Exit(2)
	}

	config.TemplateDir,err = c.GetString("directories", "template_dir")
	if err != nil {
		config.TemplateDir = "templates"
		fmt.Println("directories.template_dir not set in config.cfg, defaulting to "+config.TemplateDir)
	}

	config.LogDir,err = c.GetString("directories", "log_dir")
	if err != nil {
		config.LogDir = "log"
		fmt.Println("directories.log_dir not set in config.cfg, defaulting to "+config.LogDir)
	}

	access_log,err = logfile.OpenLogFile(config.LogDir+"/access.log",false)
	if err != nil {
		fmt.Println("Couldn't open access log. Returned error: "+err.Error())
	}

	error_log,err = logfile.OpenLogFile(config.LogDir+"/error.log",false)
	if err != nil {
		fmt.Println("Couldn't open error log. Returned error: "+err.Error())
	}

	config.DBtype,err = c.GetString("database", "type")
	if err != nil {
		config.DBtype = "mysql"
		fmt.Println("database.db_type not set in config.cfg, defaulting to "+config.DBtype)
	}

	config.DBhost,err = c.GetString("database", "host")
	if err != nil {
		config.DBhost = "unix(/var/run/mysqld/mysqld.sock)"
		fmt.Println("database.db_host not set in config.cfg, defaulting to "+config.DBhost)
	}

	config.DBname,err = c.GetString("database", "name")
	if err != nil {
		fmt.Println("database.db_name not set in config.cfg, halting.")
		os.Exit(2)
	}

	config.DBusername,err = c.GetString("database", "username")
	if err != nil {
		fmt.Println("database.db_username not set in config.cfg, halting.")
		os.Exit(2)
	}
	config.DBpassword,err = c.GetString("database", "password")
	if err != nil {
		config.DBpassword = ""
	}

	config.DBprefix,err = c.GetString("database", "prefix") 
	if err == nil {
		config.DBprefix += "_"
	} else {
		config.DBprefix = ""
	}

	config.DBkeepalive,err = c.GetBool("database", "keepalive")
	if err != nil {
		config.DBkeepalive = false
	}

	config.Lockdown,err = c.GetBool("gochan", "lockdown")
	if err != nil {
		config.Lockdown = false
	}
	config.LockdownMessage,err = c.GetString("gochan", "lockdown_message")
	if err != nil {
		config.LockdownMessage = ""
	}
	
	config.Sillytags,err = c.GetString("gochan", "sillytags")
	if err != nil {
		config.Sillytags = ""
	}

	config.UseSillytags,err = c.GetBool("gochan", "use_sillytags")
	if err != nil {
		config.UseSillytags = false
	}
	config.Modboard,err = c.GetString("gochan", "mod_board")
	if err != nil {
		config.Modboard = "staff"
	}

	config.SiteName,err = c.GetString("site", "name")
	if err != nil {
		config.SiteName = "An unnamed imageboard"
	}

	config.SiteSlogan,err = c.GetString("site", "slogan")
	if err != nil {
		config.SiteSlogan = ""
	}

	config.SiteWebfolder,err = c.GetString("site", "webfolder")
	if err != nil {
		fmt.Println("site.webfolder not set in config.cfg, halting.")
		os.Exit(2)
	}

	styles_str,err_ := c.GetString("styles", "styles")
	if err == nil {
		config.Styles_img = strings.Split(styles_str, ",")
	}

	config.DefaultStyle_img,err = c.GetString("styles", "default_style")
	if err != nil {
		config.DefaultStyle_img = "pipes"
	}

	styles_txt_str,err_ := c.GetString("styles", "styles_txt")
	if err == nil {
		config.Styles_txt = strings.Split(styles_txt_str, ",")
	}

	config.DefaultStyle_txt,err = c.GetString("styles", "default_txt_style")
	if err != nil {
		config.DefaultStyle_txt = "pipes"
	}


	config.AllowDuplicateImages,err = c.GetBool("posting", "allow_duplicate_images")
	if err != nil {
		config.AllowDuplicateImages = true
	}

	config.NewThreadDelay,err = c.GetInt("posting", "new_thread_delay")
	if err != nil {
		config.NewThreadDelay = 30
	}

	config.ReplyDelay,err = c.GetInt("posting", "reply_delay")
	if err != nil {
		config.ReplyDelay = 7
	}

	config.MaxLineLength,err = c.GetInt("posting", "max_line_length")
	if err != nil {
		config.MaxLineLength = 150
	}
	//ReservedTrips string //eventually this will be map[string]string

	config.ThumbWidth,err = c.GetInt("thumbnails", "thumb_width")
	if err != nil {
		config.ThumbWidth = 200
	}

	config.ThumbWidth,err = c.GetInt("thumbnails", "thumb_height")
	if err != nil {
		config.ThumbHeight = 200
	}

	config.ThumbWidth_reply,err = c.GetInt("thumbnails", "reply_thumb_width")
	if err != nil {
		config.ThumbWidth_reply = 125
	}

	config.ThumbWidth,err = c.GetInt("thumbnails", "reply_thumb_width")
	if err != nil {
		config.ThumbHeight_reply = 125
	}

	config.ThumbWidth,err = c.GetInt("thumbnails", "catalog_thumb_width")
	if err != nil {
		config.ThumbWidth_catalog = 50
	}

	config.ThumbWidth,err = c.GetInt("thumbnails", "catalog_thumb_width")
	if err != nil {
		config.ThumbHeight_catalog = 50
	}


	config.ThreadsPerPage_img,err = c.GetInt("threads", "img_threads_per_page")
	if err != nil {
		config.ThreadsPerPage_img = 10
	}

	config.ThreadsPerPage_txt,err = c.GetInt("threads", "txt_threads_per_page")
	if err != nil {
		config.ThreadsPerPage_txt = 15
	}

	config.PostsPerThreadpage,err = c.GetInt("threads", "posts_per_threadpage")
	if err != nil {
		config.PostsPerThreadpage = 50
	}

	config.RepliesOnBoardpage,err = c.GetInt("threads", "replies_on_boardpage")
	if err != nil {
		config.RepliesOnBoardpage = 3
	}

	config.StickyRepliesOnBoardPage,err = c.GetInt("threads", "sticky_replies_on_boardpage")
	if err != nil {
		config.StickyRepliesOnBoardPage = 1
	}

	config.GenLast50,err = c.GetBool("threads", "gen_last50_page")
	if err != nil {
		config.GenLast50 = true
	}

	config.GenFirst100,err = c.GetBool("threads", "gen_first100_page")
	if err != nil {
		config.GenFirst100 = false
	}

	config.BanColors,err = c.GetString("threads", "ban_colors") //eventually this will be map[string] string
	if err != nil {
		config.BanColors = "admin:#CC0000"
	}

	config.BanMsg,err = c.GetString("threads", "ban_msg")
	if err != nil {
		config.BanMsg = "(USER WAS BANNED FOR THIS POST)"
	}

	config.ExpandButton,err = c.GetBool("threads", "expand_button")
	if err != nil {
		config.ExpandButton = true
	}

	config.ImagesOpenNewTab,err = c.GetBool("threads", "images_open_new_tab")
	if err != nil {
		config.ImagesOpenNewTab = true
	}

	config.MakeURLsHyperlinked,err = c.GetBool("threads", "make_urls_hyperlinked")
	if err != nil {
		config.MakeURLsHyperlinked = true
	}

	config.NewTabOnOutlinks,err = c.GetBool("threads", "new_tab_on_outlinks")
	if err != nil {
		config.NewTabOnOutlinks = true
	}

	config.EnableQuickReply,err = c.GetBool("threads", "quick_reply")
	if err != nil {
		config.EnableQuickReply = true
	}

	config.DefaultBanReason,err = c.GetString("misc","default_ban_reason")
	if err != nil {
		config.DefaultBanReason = ""
	}

	config.EnableGeoIP,err = c.GetBool("misc", "enable_geoip")
	if err != nil {
		config.EnableGeoIP = false
	}

	config.GeoIPDBlocation,err = c.GetString("misc","geoip_location") // cf for cloudflare or a local path
	if err != nil {
		if config.EnableGeoIP {
			fmt.Println("Error: GeoIP enabled but no database provided. Set misc.geoip_location in config.cfg to \"cf\" to use CloudFlare's GeoIP headers, or to a local filepath")
		} else {
			config.GeoIPDBlocation = ""
		}
	}

	config.MaxRecentPosts,err = c.GetInt("misc", "max_recent_posts")
	if err != nil {
		config.MaxRecentPosts = 10
	}

	config.MakeRSS,err = c.GetBool("misc", "make_rss")
	if err != nil {
		config.MakeRSS = false
	}

	config.MakeSitemap,err = c.GetBool("misc", "make_sitemap")
	if err != nil {
		config.MakeSitemap = false
	}

	config.EnableAppeals,err = c.GetBool("misc", "enable_appeals")
	if err != nil {
		config.EnableAppeals = true
	}

	config.MaxModlogDays,err = c.GetInt("misc", "max_modlog_days")
	if err != nil {
		config.MaxModlogDays = 15
	}

	config.RandomSeed,err = c.GetString("misc", "random_seed")
	if err != nil {
	}

	config.Version = version
}