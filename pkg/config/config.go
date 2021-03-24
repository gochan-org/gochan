package config

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"reflect"

	"github.com/gochan-org/gochan/pkg/gclog"
)

const (
	randomStringSize = 16
)

var (
	Config      *GochanConfig
	cfgPath     string
	cfgDefaults = map[string]interface{}{
		"Port":         8080,
		"FirstPage":    []string{"index.html", "board.html", "firstrun.html"},
		"DocumentRoot": "html",
		"TemplateDir":  "templates",
		"LogDir":       "log",

		"SillyTags": []string{},

		"SiteName":      "Gochan",
		"SiteWebFolder": "/",

		"NewThreadDelay": 30,
		"ReplyDelay":     7,

		"MaxLineLength": 150,

		"ThreadsPerPage": 15,

		"RepliesOnBoardPage":       3,
		"StickyRepliesOnBoardPage": 1,

		"ThumbWidth":         200,
		"ThumbHeight":        200,
		"ThumbWidthReply":    125,
		"ThumbHeightReply":   125,
		"ThumbWidthCatalog":  50,
		"ThumbHeightCatalog": 50,

		"BanMsg":           "USER WAS BANNED FOR THIS POST",
		"EmbedWidth":       200,
		"EmbedHeight":      164,
		"ExpandButton":     true,
		"NewTabOnOutlinks": true,

		"MinifyHTML": true,
		"MinifyJS":   true,

		"CaptchaWidth":  240,
		"CaptchaHeight": 80,

		"DateTimeFormat":       "Mon, January 02, 2006 15:04 PM",
		"CaptchaMinutesExpire": 15,
		"EnableGeoIP":          true,
		"GeoIPDBlocation":      "/usr/share/GeoIP/GeoIP.dat",
		"MaxRecentPosts":       3,
		"MaxLogDays":           15,
	}
)

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	Name     string
	Filename string
}

// GochanConfig stores important info and is read from/written to gochan.json.
// If a field has an entry in the defaults map, that value will be used here.
// If a field has a critical struct tag set to "true", a warning will be printed
// if it exists in the defaults map and an error will be printed if it doesn't.
type GochanConfig struct {
	ListenIP   string   `critical:"true"`
	Port       int      `critical:"true"`
	FirstPage  []string `critical:"true"`
	Username   string   `critical:"true"`
	UseFastCGI bool     `critical:"true"`
	DebugMode  bool     `description:"Disables several spam/browser checks that can cause problems when hosting an instance locally."`

	DocumentRoot string `critical:"true"`
	TemplateDir  string `critical:"true"`
	LogDir       string `critical:"true"`

	DBtype     string `critical:"true"`
	DBhost     string `critical:"true"`
	DBname     string `critical:"true"`
	DBusername string `critical:"true"`
	DBpassword string `critical:"true"`
	DBprefix   string `description:"Each table's name in the database will start with this, if it is set"`

	SiteName      string `description:"The name of the site that appears in the header of the front page."`
	SiteSlogan    string `description:"The text that appears below SiteName on the home page"`
	SiteWebfolder string `critical:"true" description:"The HTTP root appearing in the browser (e.g. https://gochan.org/<SiteWebFolder>"`
	SiteDomain    string `critical:"true" description:"The server's domain. Do not edit this unless you know what you are doing or BAD THINGS WILL HAPPEN!"`

	Lockdown        bool     `description:"Disables posting."`
	LockdownMessage string   `description:"Message displayed when someone tries to post while the site is on lockdown."`
	Sillytags       []string `description:"List of randomly selected fake staff tags separated by line, e.g. ## Mod, to be randomly assigned to posts if UseSillytags is checked. Don't include the \"## \""`
	UseSillytags    bool     `description:"Use Sillytags"`
	Modboard        string   `description:"A super secret clubhouse board that only staff can view/post to."`

	Styles       []Style `critical:"true" description:"List of styles (one per line) that should be accessed online at <SiteWebFolder>/css/<Style>"`
	DefaultStyle string  `description:"Filename of the default Style. If this unset, the first entry in the Styles array will be used."`

	RejectDuplicateImages bool     `description:"Enabling this will cause gochan to reject a post if the image has already been uploaded for another post.\nThis may end up being removed or being made board-specific in the future."`
	NewThreadDelay        int      `description:"The amount of time in seconds that is required before an IP can make a new thread.<br />This may end up being removed or being made board-specific in the future."`
	ReplyDelay            int      `description:"Same as the above, but for replies."`
	MaxLineLength         int      `description:"Any line in a post that exceeds this will be split into two (or more) lines.<br />I'm not really sure why this is here, so it may end up being removed."`
	ReservedTrips         []string `description:"Secure tripcodes (!!Something) can be reserved here.<br />Each reservation should go on its own line and should look like this:<br />TripPassword1##Tripcode1<br />TripPassword2##Tripcode2"`

	ThumbWidth         int `description:"OP thumbnails use this as their max width.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger."`
	ThumbHeight        int `description:"OP thumbnails use this as their max height.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger."`
	ThumbWidthReply    int `description:"Same as ThumbWidth and ThumbHeight but for reply images."`
	ThumbHeightReply   int `description:"Same as ThumbWidth and ThumbHeight but for reply images."`
	ThumbWidthCatalog  int `description:"Same as ThumbWidth and ThumbHeight but for catalog images."`
	ThumbHeightCatalog int `description:"Same as ThumbWidth and ThumbHeight but for catalog images."`

	ThreadsPerPage           int
	RepliesOnBoardPage       int    `description:"Number of replies to a thread to show on the board page."`
	StickyRepliesOnBoardPage int    `description:"Same as above for stickied threads."`
	BanMsg                   string `description:"The default public ban message."`
	EmbedWidth               int    `description:"The width for inline/expanded videos."`
	EmbedHeight              int    `description:"The height for inline/expanded videos."`
	ExpandButton             bool   `description:"If checked, adds [Embed] after a Youtube, Vimeo, etc link to toggle an inline video frame."`
	ImagesOpenNewTab         bool   `description:"If checked, thumbnails will open the respective image/video in a new tab instead of expanding them." `
	NewTabOnOutlinks         bool   `description:"If checked, links to external sites will open in a new tab."`
	DisableBBcode            bool   `description:"If checked, gochan will not compile bbcode into HTML"`

	MinifyHTML bool `description:"If checked, gochan will minify html files when building"`
	MinifyJS   bool `description:"If checked, gochan will minify js and json files when building"`

	DateTimeFormat        string `description:"The format used for dates. See <a href=\"https://golang.org/pkg/time/#Time.Format\">here</a> for more info."`
	AkismetAPIKey         string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
	UseCaptcha            bool   `description:"If checked, a captcha will be generated"`
	CaptchaWidth          int    `description:"Width of the generated captcha image"`
	CaptchaHeight         int    `description:"Height of the generated captcha image"`
	CaptchaMinutesExpire  int    `description:"Number of minutes before a user has to enter a new CAPTCHA before posting. If <1 they have to submit one for every post."`
	EnableGeoIP           bool   `description:"If checked, this enables the usage of GeoIP for posts."`
	GeoIPDBlocation       string `description:"Specifies the location of the GeoIP database file. If you're using CloudFlare, you can set it to cf to rely on CloudFlare for GeoIP information."`
	MaxRecentPosts        int    `description:"The maximum number of posts to show on the Recent Posts list on the front page."`
	RecentPostsWithNoFile bool   `description:"If checked, recent posts with no image/upload are shown on the front page (as well as those with images"`
	MaxLogDays            int    `description:"The maximum number of days to keep messages in the moderation/staff log file."`
	RandomSeed            string `critical:"true"`

	jsonLocation string         `json:"-"`
	TimeZone     int            `json:"-"`
	Version      *GochanVersion `json:"-"`
}

// ToMap returns the configuration file as a map
func (cfg *GochanConfig) ToMap() map[string]interface{} {
	cVal := reflect.ValueOf(cfg).Elem()
	cType := reflect.TypeOf(*cfg)
	numFields := cType.NumField()
	out := make(map[string]interface{})
	for f := 0; f < numFields; f++ {
		field := cVal.Field(f)
		if !field.CanSet() {
			continue
		}
		out[cType.Field(f).Name] = field.Elem().Interface()
	}
	return out
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

// ValidateValues checks to make sure that the configuration options are usable
// (e.g., ListenIP is a valid IP address, Port isn't a negative number, etc)
func (cfg *GochanConfig) ValidateValues() error {
	if net.ParseIP(cfg.ListenIP) == nil {
		return &ErrInvalidValue{Field: "ListenIP", Value: cfg.ListenIP}
	}
	changed := false
	if len(cfg.FirstPage) == 0 {
		cfg.FirstPage = cfgDefaults["FirstPage"].([]string)
		changed = true
	}
	if cfg.DBtype != "mysql" && cfg.DBtype != "postgresql" {
		return &ErrInvalidValue{Field: "DBtype", Value: cfg.DBtype, Details: "currently supported values: mysql, postgresql"}
	}
	if len(cfg.Styles) == 0 {
		return &ErrInvalidValue{Field: "Styles", Value: cfg.Styles}
	}
	if cfg.DefaultStyle == "" {
		cfg.DefaultStyle = cfg.Styles[0].Filename
		changed = true
	}
	if cfg.NewThreadDelay == 0 {
		cfg.NewThreadDelay = cfgDefaults["NewThreadDelay"].(int)
		changed = true
	}
	if cfg.ReplyDelay == 0 {
		cfg.ReplyDelay = cfgDefaults["ReplyDelay"].(int)
		changed = true
	}
	if cfg.MaxLineLength == 0 {
		cfg.MaxLineLength = cfgDefaults["MaxLineLength"].(int)
		changed = true
	}
	if cfg.ThumbWidth == 0 {
		cfg.ThumbWidth = cfgDefaults["ThumbWidth"].(int)
		changed = true
	}
	if cfg.ThumbHeight == 0 {
		cfg.ThumbHeight = cfgDefaults["ThumbHeight"].(int)
		changed = true
	}
	if cfg.ThumbWidthReply == 0 {
		cfg.ThumbWidthReply = cfgDefaults["ThumbWidthReply"].(int)
		changed = true
	}
	if cfg.ThumbHeightReply == 0 {
		cfg.ThumbHeightReply = cfgDefaults["ThumbHeightReply"].(int)
		changed = true
	}
	if cfg.ThumbWidthCatalog == 0 {
		cfg.ThumbWidthCatalog = cfgDefaults["ThumbWidthCatalog"].(int)
		changed = true
	}
	if cfg.ThumbHeightCatalog == 0 {
		cfg.ThumbHeightCatalog = cfgDefaults["ThumbHeightCatalog"].(int)
		changed = true
	}
	if cfg.ThreadsPerPage == 0 {
		cfg.ThreadsPerPage = cfgDefaults["ThreadsPerPage"].(int)
		changed = true
	}
	if cfg.RepliesOnBoardPage == 0 {
		cfg.RepliesOnBoardPage = cfgDefaults["RepliesOnBoardPage"].(int)
		changed = true
	}
	if cfg.StickyRepliesOnBoardPage == 0 {
		cfg.StickyRepliesOnBoardPage = cfgDefaults["StickyRepliesOnBoardPage"].(int)
		changed = true
	}
	if cfg.BanMsg == "" {
		cfg.BanMsg = cfgDefaults["BanMsg"].(string)
		changed = true
	}
	if cfg.DateTimeFormat == "" {
		cfg.DateTimeFormat = cfgDefaults["DateTimeFormat"].(string)
		changed = true
	}
	if cfg.CaptchaWidth == 0 {
		cfg.CaptchaWidth = cfgDefaults["CaptchaWidth"].(int)
		changed = true
	}
	if cfg.CaptchaHeight == 0 {
		cfg.CaptchaHeight = cfgDefaults["CaptchaHeight"].(int)
		changed = true
	}
	if cfg.EnableGeoIP {
		if cfg.GeoIPDBlocation == "" {
			return &ErrInvalidValue{Field: "GeoIPDBlocation", Value: "", Details: "GeoIPDBlocation must be set in gochan.json if EnableGeoIP is true"}
		}
	}

	if cfg.MaxLogDays == 0 {
		cfg.MaxLogDays = cfgDefaults["MaxLogDays"].(int)
		changed = true
	}

	if cfg.RandomSeed == "" {
		cfg.RandomSeed = randomString(randomStringSize)
		changed = true
	}
	if !changed {
		return nil
	}
	return cfg.Write()
}

func (cfg *GochanConfig) Write() error {
	str, err := json.MarshalIndent(cfg, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.jsonLocation, str, 0777)
}
