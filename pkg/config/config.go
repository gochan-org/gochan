package config

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"reflect"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	randomStringSize = 16
	cookieMaxAgeEx   = ` (example: "1 year 2 months 3 days 4 hours", or "1y2mo3d4h"`
	/* currentConfig = iota
	oldConfig
	invalidConfig */
)

var (
	cfg          *GochanConfig
	cfgPath      string
	cfgDefaults  = map[string]interface{}{}
	boardConfigs = map[string]BoardConfig{}
)

type GochanConfig struct {
	SystemCriticalConfig
	SiteConfig
	BoardConfig
	jsonLocation string `json:"-"`
}

// ToMap returns the configuration file as a map. This will probably be removed
func (gcfg *GochanConfig) ToMap() map[string]interface{} {
	cVal := reflect.ValueOf(gcfg).Elem()
	cType := reflect.TypeOf(*gcfg)
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

// ValidateValues checks to make sure that the configuration options are usable
// (e.g., ListenIP is a valid IP address, Port isn't a negative number, etc)
func (gcfg *GochanConfig) ValidateValues() error {
	if net.ParseIP(gcfg.ListenIP) == nil {
		return &ErrInvalidValue{Field: "ListenIP", Value: gcfg.ListenIP}
	}
	changed := false
	if len(gcfg.FirstPage) == 0 {
		gcfg.FirstPage = cfgDefaults["FirstPage"].([]string)
		changed = true
	}
	_, err := gcutil.ParseDurationString(gcfg.CookieMaxAge)
	if err == gcutil.ErrInvalidDurationString {
		return &ErrInvalidValue{Field: "CookieMaxAge", Value: gcfg.CookieMaxAge, Details: err.Error() + cookieMaxAgeEx}
	} else if err == gcutil.ErrEmptyDurationString {
		return &ErrInvalidValue{Field: "CookieMaxAge", Details: err.Error() + cookieMaxAgeEx}
	} else if err != nil {
		return err
	}
	if gcfg.DBtype != "mysql" && gcfg.DBtype != "postgresql" {
		return &ErrInvalidValue{Field: "DBtype", Value: gcfg.DBtype, Details: "currently supported values: mysql, postgresql"}
	}
	if len(gcfg.Styles) == 0 {
		return &ErrInvalidValue{Field: "Styles", Value: gcfg.Styles}
	}
	if gcfg.DefaultStyle == "" {
		gcfg.DefaultStyle = gcfg.Styles[0].Filename
		changed = true
	}
	if gcfg.NewThreadDelay == 0 {
		gcfg.NewThreadDelay = cfgDefaults["NewThreadDelay"].(int)
		changed = true
	}
	if gcfg.ReplyDelay == 0 {
		gcfg.ReplyDelay = cfgDefaults["ReplyDelay"].(int)
		changed = true
	}
	if gcfg.MaxLineLength == 0 {
		gcfg.MaxLineLength = cfgDefaults["MaxLineLength"].(int)
		changed = true
	}
	if gcfg.ThumbWidth == 0 {
		gcfg.ThumbWidth = cfgDefaults["ThumbWidth"].(int)
		changed = true
	}
	if gcfg.ThumbHeight == 0 {
		gcfg.ThumbHeight = cfgDefaults["ThumbHeight"].(int)
		changed = true
	}
	if gcfg.ThumbWidthReply == 0 {
		gcfg.ThumbWidthReply = cfgDefaults["ThumbWidthReply"].(int)
		changed = true
	}
	if gcfg.ThumbHeightReply == 0 {
		gcfg.ThumbHeightReply = cfgDefaults["ThumbHeightReply"].(int)
		changed = true
	}

	if gcfg.ThumbWidthCatalog == 0 {
		gcfg.ThumbWidthCatalog = cfgDefaults["ThumbWidthCatalog"].(int)
		changed = true
	}
	if gcfg.ThumbHeightCatalog == 0 {
		gcfg.ThumbHeightCatalog = cfgDefaults["ThumbHeightCatalog"].(int)
		changed = true
	}
	if gcfg.ThreadsPerPage == 0 {
		gcfg.ThreadsPerPage = cfgDefaults["ThreadsPerPage"].(int)
		changed = true
	}
	if gcfg.RepliesOnBoardPage == 0 {
		gcfg.RepliesOnBoardPage = cfgDefaults["RepliesOnBoardPage"].(int)
		changed = true
	}
	if gcfg.StickyRepliesOnBoardPage == 0 {
		gcfg.StickyRepliesOnBoardPage = cfgDefaults["StickyRepliesOnBoardPage"].(int)
		changed = true
	}
	if gcfg.BanMsg == "" {
		gcfg.BanMsg = cfgDefaults["BanMsg"].(string)
		changed = true
	}
	if gcfg.DateTimeFormat == "" {
		gcfg.DateTimeFormat = cfgDefaults["DateTimeFormat"].(string)
		changed = true
	}
	if gcfg.CaptchaWidth == 0 {
		gcfg.CaptchaWidth = cfgDefaults["CaptchaWidth"].(int)
		changed = true
	}
	if gcfg.CaptchaHeight == 0 {
		gcfg.CaptchaHeight = cfgDefaults["CaptchaHeight"].(int)
		changed = true
	}
	if gcfg.EnableGeoIP {
		if gcfg.GeoIPDBlocation == "" {
			return &ErrInvalidValue{Field: "GeoIPDBlocation", Value: "", Details: "GeoIPDBlocation must be set in gochan.json if EnableGeoIP is true"}
		}
	}

	if gcfg.MaxLogDays == 0 {
		gcfg.MaxLogDays = cfgDefaults["MaxLogDays"].(int)
		changed = true
	}

	if gcfg.RandomSeed == "" {
		gcfg.RandomSeed = gcutil.RandomString(randomStringSize)
		changed = true
	}
	if !changed {
		return nil
	}
	return gcfg.Write()
}

func (gcfg *GochanConfig) Write() error {
	str, err := json.MarshalIndent(gcfg, "", "\t")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(gcfg.jsonLocation, str, 0777)
}

/*
 SystemCriticalConfig contains configuration options that are extremely important, and fucking with them while
 the server is running could have site breaking consequences. It should only be changed by modifying the configuration
 file and restarting the server.
*/
type SystemCriticalConfig struct {
	ListenIP     string
	Port         int
	UseFastCGI   bool
	DocumentRoot string
	TemplateDir  string
	LogDir       string

	SiteHeaderURL string
	WebRoot       string `description:"The HTTP root appearing in the browser (e.g. '/', 'https://yoursite.net/', etc) that all internal links start with"`
	SiteDomain    string `description:"The server's domain (e.g. gochan.org, 127.0.0.1, etc)"`

	DBtype     string
	DBhost     string
	DBname     string
	DBusername string
	DBpassword string
	DBprefix   string `description:"Each table's name in the database will start with this, if it is set"`

	DebugMode  bool `description:"Disables several spam/browser checks that can cause problems when hosting an instance locally."`
	RandomSeed string
	Version    *GochanVersion `json:"-"`
	TimeZone   int            `json:"-"`
}

type SiteConfig struct {
	FirstPage       []string
	Username        string
	CookieMaxAge    string `description:"The amount of time that session cookies will exist before they expire (ex: 1y2mo3d4h or 1 year 2 months 3 days 4 hours). Default is 1 year"`
	Lockdown        bool   `description:"Disables posting."`
	LockdownMessage string `description:"Message displayed when someone tries to post while the site is on lockdown."`

	SiteName   string `description:"The name of the site that appears in the header of the front page."`
	SiteSlogan string `description:"The text that appears below SiteName on the home page"`
	Modboard   string `description:"A super secret clubhouse board that only staff can view/post to."`

	MaxRecentPosts        int  `description:"The maximum number of posts to show on the Recent Posts list on the front page."`
	RecentPostsWithNoFile bool `description:"If checked, recent posts with no image/upload are shown on the front page (as well as those with images"`
	Verbosity             int
	EnableAppeals         bool
	MaxLogDays            int `description:"The maximum number of days to keep messages in the moderation/staff log file."`

	MinifyHTML      bool   `description:"If checked, gochan will minify html files when building"`
	MinifyJS        bool   `description:"If checked, gochan will minify js and json files when building"`
	GeoIPDBlocation string `description:"Specifies the location of the GeoIP database file. If you're using CloudFlare, you can set it to cf to rely on CloudFlare for GeoIP information."`
	AkismetAPIKey   string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
}

type BoardConfig struct {
	InheritGlobalStyles bool     `description:"If checked, a board uses the global Styles array + the board config's styles (with duplicates removed)"`
	Styles              []Style  `description:"List of styles (one per line) that should be accessed online at <SiteWebFolder>/css/<Style>"`
	DefaultStyle        string   `description:"Filename of the default Style. If this unset, the first entry in the Styles array will be used."`
	Sillytags           []string `description:"List of randomly selected fake staff tags separated by line, e.g. ## Mod, to be randomly assigned to posts if UseSillytags is checked. Don't include the \"## \""`
	UseSillytags        bool     `description:"Use Sillytags"`

	PostConfig
	UploadConfig

	DateTimeFormat        string `description:"The format used for dates. See <a href=\"https://golang.org/pkg/time/#Time.Format\">here</a> for more info."`
	AkismetAPIKey         string `description:"The API key to be sent to Akismet for post spam checking. If the key is invalid, Akismet won't be used."`
	UseCaptcha            bool
	CaptchaWidth          int
	CaptchaHeight         int
	CaptchaMinutesTimeout int

	EnableGeoIP bool
}

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	Name     string
	Filename string
}

type UploadConfig struct {
	RejectDuplicateImages bool `description:"Enabling this will cause gochan to reject a post if the image has already been uploaded for another post.\nThis may end up being removed or being made board-specific in the future."`
	AllowVideoUploads     bool
	ThumbWidth            int `description:"OP thumbnails use this as their max width.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger."`
	ThumbHeight           int `description:"OP thumbnails use this as their max height.<br />To keep the aspect ratio, the image will be scaled down to the ThumbWidth or ThumbHeight, whichever is larger."`
	ThumbWidthReply       int `description:"Same as ThumbWidth and ThumbHeight but for reply images."`
	ThumbHeightReply      int `description:"Same as ThumbWidth and ThumbHeight but for reply images."`
	ThumbWidthCatalog     int `description:"Same as ThumbWidth and ThumbHeight but for catalog images."`
	ThumbHeightCatalog    int `description:"Same as ThumbWidth and ThumbHeight but for catalog images."`
}

type PostConfig struct {
	NewThreadDelay int `description:"The amount of time in seconds that is required before an IP can make a new thread.<br />This may end up being removed or being made board-specific in the future."`

	ReplyDelay    int      `description:"Same as the above, but for replies."`
	MaxLineLength int      `description:"Any line in a post that exceeds this will be split into two (or more) lines.<br />I'm not really sure why this is here, so it may end up being removed."`
	ReservedTrips []string `description:"Secure tripcodes (!!Something) can be reserved here.<br />Each reservation should go on its own line and should look like this:<br />TripPassword1##Tripcode1<br />TripPassword2##Tripcode2"`

	ThreadsPerPage           int
	PostsPerThreadPage       int
	RepliesOnBoardPage       int `description:"Number of replies to a thread to show on the board page."`
	StickyRepliesOnBoardPage int `description:"Same as above for stickied threads."`

	BanColors        []string
	BanMsg           string `description:"The default public ban message."`
	EmbedWidth       int    `description:"The width for inline/expanded videos."`
	EmbedHeight      int    `description:"The height for inline/expanded videos."`
	ExpandButton     bool   `description:"If checked, adds [Embed] after a Youtube, Vimeo, etc link to toggle an inline video frame."`
	ImagesOpenNewTab bool   `description:"If checked, thumbnails will open the respective image/video in a new tab instead of expanding them." `
	NewTabOnOutlinks bool   `description:"If checked, links to external sites will open in a new tab."`
	DisableBBcode    bool   `description:"If checked, gochan will not compile bbcode into HTML"`
}

func WriteConfig() error {
	return cfg.Write()
}

// GetSystemCriticalConfig returns system-critical configuration options like listening IP
func GetSystemCriticalConfig() SystemCriticalConfig {
	return cfg.SystemCriticalConfig
}

// GetSiteConfig returns the global site configuration (site name, slogan, etc)
func GetSiteConfig() *SiteConfig {
	return &cfg.SiteConfig
}

// GetBoardConfig returns the custom configuration for the specified board (if it exists)
// or the global board configuration if board is an empty string or it doesn't exist
func GetBoardConfig(board string) *BoardConfig {
	bc, exists := boardConfigs[board]
	if board == "" || !exists {
		return &cfg.BoardConfig
	}
	return &bc
}

func GetVersion() *GochanVersion {
	return cfg.Version
}
