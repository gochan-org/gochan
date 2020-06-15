package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/gclog"
)

var Config GochanConfig

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	Name     string
	Filename string
}

// GochanConfig stores crucial info and is read from/written to gochan.json
type GochanConfig struct {
	ListenIP   string
	Port       int
	FirstPage  []string
	Username   string
	UseFastCGI bool
	DebugMode  bool `description:"Disables several spam/browser checks that can cause problems when hosting an instance locally."`

	DocumentRoot string
	TemplateDir  string
	LogDir       string

	DBtype     string
	DBhost     string
	DBname     string
	DBusername string
	DBpassword string
	DBprefix   string

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

	Styles       []Style `description:"List of styles (one per line) that should be accessed online at &lt;SiteWebFolder&gt;/css/&lt;Style&gt;/"`
	DefaultStyle string  `description:"Filename of the default Style. This should appear in the list above or bad things might happen."`

	AllowDuplicateImages bool     `description:"Disabling this will cause gochan to reject a post if the image has already been uploaded for another post.<br />This may end up being removed or being made board-specific in the future." default:"checked"`
	AllowVideoUploads    bool     `description:"Allows users to upload .webm videos. <br />This may end up being removed or being made board-specific in the future."`
	NewThreadDelay       int      `description:"The amount of time in seconds that is required before an IP can make a new thread.<br />This may end up being removed or being made board-specific in the future." default:"30"`
	ReplyDelay           int      `description:"Same as the above, but for replies." default:"7"`
	MaxLineLength        int      `description:"Any line in a post that exceeds this will be split into two (or more) lines.<br />I'm not really sure why this is here, so it may end up being removed." default:"150"`
	ReservedTrips        []string `description:"Secure tripcodes (!!Something) can be reserved here.<br />Each reservation should go on its own line and should look like this:<br />TripPassword1##Tripcode1<br />TripPassword2##Tripcode2"`

	ThumbWidth         int `description:"OP thumbnails use this as their max width.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbHeight        int `description:"OP thumbnails use this as their max height.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger." default:"200"`
	ThumbWidthReply    int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbHeightReply   int `description:"Same as ThumbWidth and ThumbHeight but for reply images." default:"125"`
	ThumbWidthCatalog  int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"50"`
	ThumbHeightCatalog int `description:"Same as ThumbWidth and ThumbHeight but for catalog images." default:"50"`

	ThreadsPerPage           int      `default:"15"`
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
	DisableBBcode            bool     `description:"If checked, gochan will not compile bbcode into HTML" default:"unchecked"`

	MinifyHTML          bool `description:"If checked, gochan will minify html files when building" default:"checked"`
	MinifyJS            bool `description:"If checked, gochan will minify js and json files when building" default:"checked"`
	UseMinifiedGochanJS bool `json:"-"`

	DateTimeFormat        string `description:"The format used for dates. See <a href=\"https://golang.org/pkg/time/#Time.Format\">here</a> for more info." default:"Mon, January 02, 2006 15:04 PM"`
	AkismetAPIKey         string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
	UseCaptcha            bool   `description:"If checked, a captcha will be generated"`
	CaptchaWidth          int    `description:"Width of the generated captcha image" default:"240"`
	CaptchaHeight         int    `description:"Height of the generated captcha image" default:"80"`
	CaptchaMinutesExpire  int    `description:"Number of minutes before a user has to enter a new CAPTCHA before posting. If <1 they have to submit one for every post." default:"15"`
	EnableGeoIP           bool   `description:"If checked, this enables the usage of GeoIP for posts." default:"checked"`
	GeoIPDBlocation       string `description:"Specifies the location of the GeoIP database file. If you're using CloudFlare, you can set it to cf to rely on CloudFlare for GeoIP information." default:"/usr/share/GeoIP/GeoIP.dat"`
	MaxRecentPosts        int    `description:"The maximum number of posts to show on the Recent Posts list on the front page." default:"3"`
	RecentPostsWithNoFile bool   `description:"If checked, recent posts with no image/upload are shown on the front page (as well as those with images" default:"unchecked"`
	EnableAppeals         bool   `description:"If checked, allow banned users to appeal their bans.<br />This will likely be removed (permanently allowing appeals) or made board-specific in the future." default:"checked"`
	MaxLogDays            int    `description:"The maximum number of days to keep messages in the moderation/staff log file."`
	RandomSeed            string `critical:"true"`

	TimeZone int            `json:"-"`
	Version  *GochanVersion `json:"-"`
}

func (cfg *GochanConfig) checkString(val, defaultVal string, critical bool, msg string) string {
	if val == "" {
		val = defaultVal
		flags := gclog.LStdLog | gclog.LErrorLog
		if critical {
			flags |= gclog.LFatal
		}
		if msg != "" {
			gclog.Print(flags, msg)
		}
	}
	return val
}

func (cfg *GochanConfig) checkInt(val, defaultVal int, critical bool, msg string) int {
	if val == 0 {
		val = defaultVal
		flags := gclog.LStdLog | gclog.LErrorLog
		if critical {
			flags |= gclog.LFatal
		}
		if msg != "" {
			gclog.Print(flags, msg)
		}
	}
	return val
}

// InitConfig loads and parses gochan.json and verifies its contents
func InitConfig(versionStr string) {
	cfgPath := findResource("gochan.json", "/etc/gochan/gochan.json")
	if cfgPath == "" {
		fmt.Println("gochan.json not found")
		os.Exit(1)
	}

	jfile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	if err = json.Unmarshal(jfile, &Config); err != nil {
		fmt.Printf("Error parsing %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	Config.LogDir = findResource(Config.LogDir, "log", "/var/log/gochan/")
	if err = gclog.InitLogs(
		path.Join(Config.LogDir, "access.log"),
		path.Join(Config.LogDir, "error.log"),
		path.Join(Config.LogDir, "staff.log"),
		Config.DebugMode); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	Config.checkString(Config.ListenIP, "", true,
		"ListenIP not set in gochan.json, halting.")

	if Config.Port == 0 {
		Config.Port = 80
	}

	if len(Config.FirstPage) == 0 {
		Config.FirstPage = []string{"index.html", "1.html", "firstrun.html"}
	}

	Config.Username = Config.checkString(Config.Username, "gochan", false,
		"Username not set in gochan.json, using 'gochan' as default")
	Config.DocumentRoot = Config.checkString(Config.DocumentRoot, "gochan", true,
		"DocumentRoot not set in gochan.json, halting.")

	wd, wderr := os.Getwd()
	if wderr == nil {
		_, staterr := os.Stat(path.Join(wd, Config.DocumentRoot, "css"))
		if staterr == nil {
			Config.DocumentRoot = path.Join(wd, Config.DocumentRoot)
		}
	}

	Config.TemplateDir = Config.checkString(
		findResource(Config.TemplateDir, "templates", "/usr/local/share/gochan/templates/", "/usr/share/gochan/templates/"), "", true,
		"TemplateDir not set in gochan.json or unable to locate template directory, halting.")

	Config.checkString(Config.DBtype, "", true,
		"DBtype not set in gochan.json, halting (currently supported values: mysql,postgresql)")
	Config.checkString(Config.DBhost, "", true,
		"DBhost not set in gochan.json, halting.")
	Config.DBname = Config.checkString(Config.DBname, "gochan", false,
		"DBname not set in gochan.json, setting to 'gochan'")

	Config.checkString(Config.DBusername, "", true,
		"DBusername not set in gochan, halting.")
	Config.checkString(Config.DBpassword, "", true,
		"DBpassword not set in gochan, halting.")
	Config.LockdownMessage = Config.checkString(Config.LockdownMessage,
		"The administrator has temporarily disabled posting. We apologize for the inconvenience", false, "")

	Config.checkString(Config.SiteName, "", true,
		"SiteName not set in gochan.json, halting.")
	Config.checkString(Config.SiteDomain, "", true,
		"SiteName not set in gochan.json, halting.")

	if Config.SiteWebfolder == "" {
		gclog.Print(gclog.LErrorLog|gclog.LStdLog, "SiteWebFolder not set in gochan.json, using / as default.")
	} else if string(Config.SiteWebfolder[0]) != "/" {
		Config.SiteWebfolder = "/" + Config.SiteWebfolder
	}
	if Config.SiteWebfolder[len(Config.SiteWebfolder)-1:] != "/" {
		Config.SiteWebfolder += "/"
	}

	if Config.Styles == nil {
		gclog.Print(gclog.LErrorLog|gclog.LStdLog|gclog.LFatal, "Styles not set in gochan.json, halting.")
	}

	Config.DefaultStyle = Config.checkString(Config.DefaultStyle, Config.Styles[0].Filename, false, "")

	Config.NewThreadDelay = Config.checkInt(Config.NewThreadDelay, 30, false, "")
	Config.ReplyDelay = Config.checkInt(Config.ReplyDelay, 7, false, "")
	Config.MaxLineLength = Config.checkInt(Config.MaxLineLength, 150, false, "")
	//ReservedTrips string //eventually this will be map[string]string

	Config.ThumbWidth = Config.checkInt(Config.ThumbWidth, 200, false, "")
	Config.ThumbHeight = Config.checkInt(Config.ThumbHeight, 200, false, "")
	Config.ThumbWidthReply = Config.checkInt(Config.ThumbWidthReply, 125, false, "")
	Config.ThumbHeightReply = Config.checkInt(Config.ThumbHeightReply, 125, false, "")
	Config.ThumbWidthCatalog = Config.checkInt(Config.ThumbWidthCatalog, 50, false, "")
	Config.ThumbHeightCatalog = Config.checkInt(Config.ThumbHeightCatalog, 50, false, "")

	Config.ThreadsPerPage = Config.checkInt(Config.ThreadsPerPage, 10, false, "")
	Config.RepliesOnBoardPage = Config.checkInt(Config.RepliesOnBoardPage, 3, false, "")
	Config.StickyRepliesOnBoardPage = Config.checkInt(Config.StickyRepliesOnBoardPage, 1, false, "")

	/*config.BanColors, err = c.GetString("threads", "ban_colors") //eventually this will be map[string] string
	if err != nil {
		config.BanColors = "admin:#CC0000"
	}*/

	Config.BanMsg = Config.checkString(Config.BanMsg, "(USER WAS BANNED FOR THIS POST)", false, "")
	Config.DateTimeFormat = Config.checkString(Config.DateTimeFormat, "Mon, January 02, 2006 15:04 PM", false, "")

	Config.CaptchaWidth = Config.checkInt(Config.CaptchaWidth, 240, false, "")
	Config.CaptchaHeight = Config.checkInt(Config.CaptchaHeight, 80, false, "")

	if Config.EnableGeoIP {
		if Config.GeoIPDBlocation == "" {
			gclog.Print(gclog.LErrorLog|gclog.LStdLog, "GeoIPDBlocation not set in gochan.json, disabling EnableGeoIP")
			Config.EnableGeoIP = false
		}
	}

	if Config.MaxLogDays == 0 {
		Config.MaxLogDays = 15
	}

	if Config.RandomSeed == "" {
		gclog.Print(gclog.LErrorLog|gclog.LStdLog, "RandomSeed not set in gochan.json, Generating a random one.")
		for i := 0; i < 8; i++ {
			num := rand.Intn(127-32) + 32
			Config.RandomSeed += fmt.Sprintf("%c", num)
		}
		configJSON, _ := json.MarshalIndent(Config, "", "\t")
		if err = ioutil.WriteFile(cfgPath, configJSON, 0777); err != nil {
			gclog.Printf(gclog.LErrorLog|gclog.LStdLog|gclog.LFatal, "Unable to write %s with randomly generated seed: %s", cfgPath, err.Error())
		}
	}

	_, zoneOffset := time.Now().Zone()
	Config.TimeZone = zoneOffset / 60 / 60

	// msgfmtr.InitBBcode()

	Config.Version = ParseVersion(versionStr)
	Config.Version.Normalize()
}
