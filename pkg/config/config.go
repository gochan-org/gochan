package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"html/template"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
)

const (
	randomStringSize = 16
	cookieMaxAgeEx   = ` (example: "1 year 2 months 3 days 4 hours", or "1y2mo3d4h"`

	DefaultSQLTimeout            = 15
	DefaultSQLMaxConns           = 10
	DefaultSQLConnMaxLifetimeMin = 3
)

var (
	cfg     *GochanConfig
	cfgPath string

	boardConfigs = map[string]BoardConfig{}
)

type GochanConfig struct {
	SystemCriticalConfig
	SiteConfig
	BoardConfig
	jsonLocation string
}

// JSONLocation returns the path to the configuration file, if loaded
func JSONLocation() string {
	if cfg == nil {
		return ""
	}
	return cfg.jsonLocation
}

func (gcfg *GochanConfig) updateDeprecatedFields() (changed bool) {
	if gcfg.ListenIP != "" && gcfg.ListenAddress == "" {
		gcfg.ListenAddress = gcfg.ListenIP
		gcfg.ListenIP = ""
		changed = true
	}
	if gcfg.SiteDomain != "" && gcfg.SiteHost == "" {
		gcfg.SiteHost = gcfg.SiteDomain
		gcfg.SiteDomain = ""
		changed = true
	}
	if gcfg.NewTabOnOutlinks && !gcfg.NewTabOnExternalLinks {
		gcfg.NewTabOnExternalLinks = true
		changed = true
	}
	return changed
}

// ValidateValues checks to make sure that the configuration options are usable
// (e.g., ListenAddress is a valid IP address, Port isn't a negative number, etc)
func (gcfg *GochanConfig) ValidateValues() error {
	changed := gcfg.updateDeprecatedFields()

	if gcfg.SiteHost == "" {
		return &InvalidValueError{Field: "SiteHost", Value: gcfg.SiteHost, Details: "must be set"}
	}
	if strings.Contains(gcfg.SiteHost, " ") || strings.Contains(gcfg.SiteHost, "://") {
		return &InvalidValueError{Field: "SiteHost", Value: gcfg.SiteHost, Details: "must be a valid host (port optional)"}
	}

	_, err := durationutil.ParseLongerDuration(gcfg.CookieMaxAge)
	if errors.Is(err, durationutil.ErrInvalidDurationString) {
		return &InvalidValueError{Field: "CookieMaxAge", Value: gcfg.CookieMaxAge, Details: err.Error() + cookieMaxAgeEx}
	} else if err != nil {
		return err
	}

	_, err = durationutil.ParseLongerDuration(gcfg.StaffSessionDuration)
	if errors.Is(err, durationutil.ErrInvalidDurationString) {
		return &InvalidValueError{Field: "StaffSessionDuration", Value: gcfg.StaffSessionDuration, Details: err.Error() + cookieMaxAgeEx}
	} else if err != nil {
		return err
	}

	if gcfg.DBtype == "postgresql" {
		gcfg.DBtype = "postgres"
		changed = true
	}
	found := false
	drivers := sql.Drivers()
	for _, driver := range drivers {
		if gcfg.DBtype == driver {
			found = true
			break
		}
	}
	if !found {
		return &InvalidValueError{
			Field:   "DBtype",
			Value:   gcfg.DBtype,
			Details: "currently supported values: " + strings.Join(drivers, ",")}
	}

	if gcfg.RandomSeed == "" {
		gcfg.RandomSeed = gcutil.RandomString(randomStringSize)
		changed = true
	}

	if gcfg.StripImageMetadata == "exif" || gcfg.StripImageMetadata == "all" {
		if gcfg.ExiftoolPath == "" {
			if gcfg.ExiftoolPath, err = exec.LookPath("exiftool"); err != nil {
				return &InvalidValueError{
					Field:   "ExiftoolPath",
					Value:   "",
					Details: "unable to find exiftool in the system path",
				}
			}
		} else {
			if _, err = exec.LookPath(gcfg.ExiftoolPath); err != nil {
				return &InvalidValueError{
					Field:   "ExiftoolPath",
					Value:   gcfg.ExiftoolPath,
					Details: "unable to find exiftool at the given location",
				}
			}
		}
	} else if gcfg.StripImageMetadata != "" && gcfg.StripImageMetadata != "none" {
		return &InvalidValueError{
			Field:   "StripImageMetadata",
			Value:   gcfg.StripImageMetadata,
			Details: `valid values are "","none","exif", or "all"`,
		}
	}

	for m, matcher := range gcfg.EmbedMatchers {
		if _, err = regexp.Compile(matcher.URLRegex); err != nil {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].URLRegex",
				Value:   matcher.URLRegex,
				Details: "invalid regular expression",
			}
		}
		if _, err = template.New(m + "framevalidate").Parse(matcher.EmbedTemplate); err != nil {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].EmbedTemplate",
				Value:   matcher.EmbedTemplate,
				Details: "invalid template",
			}
		}
		if matcher.ThumbnailURLTemplate != "" {
			if _, err = url.Parse(matcher.ThumbnailURLTemplate); err != nil {
				return &InvalidValueError{
					Field:   "EmbedMatchers[" + m + "].ThumbnailURLTemplate",
					Value:   matcher.ThumbnailURLTemplate,
					Details: "invalid URL",
				}
			}
			if _, err = template.New(m + "thumbvalidate").Parse(matcher.ThumbnailURLTemplate); err != nil {
				return &InvalidValueError{
					Field:   "EmbedMatchers[" + m + "].ThumbnailURLTemplate",
					Value:   matcher.ThumbnailURLTemplate,
					Details: "invalid template",
				}
			}
		}
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
	if testing.Testing() {
		// don't try to write anything if we're doing a test
		return nil
	}
	return os.WriteFile(gcfg.jsonLocation, str, NormalFileMode)
}

type SQLConfig struct {
	// DBtype is the type of SQL database to use. Currently supported values are "mysql", "postgres", and "sqlite3"
	DBtype string

	// DBhost is the hostname or IP address of the SQL server, or the path to the SQLite database file.
	// To connect to a MySQL database, set `DBhost` to "x.x.x.x:3306" (replacing x.x.x.x with your database server's
	// IP or domain) or a different port, if necessary. You can also use a UNIX socket if you have it set up, like
	// "unix(/var/run/mysqld/mysqld.sock)".
	// To connect to a PostgreSQL database, set `DBhost` to the IP address or hostname. Using a UNIX socket may work
	// as well, but it is currently untested.
	DBhost string

	// DBname is the name of the SQL database to connect to
	DBname string

	// DBusername is the username to use when authenticating with the SQL server
	DBusername string

	// DBpassword is the password to use when authenticating with the SQL server
	DBpassword string

	// DBprefix is the prefix to add to table names in the database. It is not requried but may be useful if you need to share a database.
	// Once you set it and do the initial setup, do not change it, as gochan will think the tables are missing and try to recreate them.
	DBprefix string

	// DBTimeoutSeconds sets the timeout for SQL queries in seconds, 0 means no timeout.
	// Default: 15
	DBTimeoutSeconds int

	// DBMaxOpenConnections is the maximum number of open connections to the database connection pool.
	// Default: 10
	DBMaxOpenConnections int

	// DBMaxIdleConnections is the maximum number of idle connections to the database connection pool.
	// Default: 10
	DBMaxIdleConnections int

	// DBConnMaxLifetimeMin is the maximum lifetime of a connection in minutes.
	// Default: 3
	DBConnMaxLifetimeMin int
}

// SystemCriticalConfig contains configuration options that are extremely important, and fucking with them while
// the server is running could have site breaking consequences. It should only be changed by modifying the configuration
// file and restarting the server.
type SystemCriticalConfig struct {
	// ListenAddress is the IP address or domain name that the server will listen on
	ListenAddress string

	// ListenIP is an alias for the ListenAddress field.
	//
	// Deprecated: Use ListenAddress instead
	ListenIP string `json:",omitempty"`

	// Port is the port that the server will listen on
	// Default: 80
	Port int

	// UseFastCGI tells the server to listen on FastCGI instead of HTTP if true
	// Default: false
	UseFastCGI bool

	// DocumentRoot is the path to the directory that contains the served static files
	DocumentRoot string

	// TemplateDir is the path to the directory that contains the template files
	TemplateDir string

	// LogDir is the path to the directory that contains the log files. It must be writable by the server and will be created if it doesn't exist
	LogDir string

	// Plugins is a list of paths to plugins to be loaded on startup. In Windows, only .lua plugins are supported. In Unix, .so plugins are also supported,
	// but they must be compiled with the same Go version as the server and must be compiled in plugin mode
	Plugins []string

	// WebRoot is the base URL path that the server will serve files and generated pages from.
	// Default: /
	WebRoot string

	// SiteHost is the publicly accessible domain name or IP address of the site, e.g. "example.com" used for anti-spam checking
	SiteHost string

	// SiteDomain is an alias for the the SiteHost field.
	//
	// Deprecated: Use SiteHost instead
	SiteDomain string `json:",omitempty"`

	SQLConfig

	// CheckRequestReferer tells the server to validate the Referer header from requests to prevent CSRF attacks.
	// Default: true
	CheckRequestReferer bool

	// Verbose currently is not used and may be removed, to be replaced with more granular logging options
	Verbose bool `json:"DebugMode"`

	// RandomSeed is a random string used for generating secure tokens. It will be generated if not set and must not be changed
	RandomSeed string

	Version  *GochanVersion `json:"-"`
	TimeZone int            `json:"-"`
}

// SiteConfig contains information about the site/community, e.g. the name of the site, the slogan (if set),
// the first page to look for if a directory is requested, etc
type SiteConfig struct {
	// FirstPage is a list of possible filenames to look for if a directory is requested
	// Default: ["index.html", "firstrun.html", "1.html"]
	FirstPage []string

	// Username is the name of the user that the server will run as, if set, or the current user if empty or unset.
	// It must be a valid user on the system if it is set
	Username string

	// CookieMaxAge is the parsed max age duration of cookies, e.g. "1 year 2 months 3 days 4 hours" or "1y2mo3d4h".
	// Default: 1y
	CookieMaxAge string

	// StaffSessionDuration is the parsed max age duration of staff session cookies, e.g. "1 year 2 months 3 days 4 hours" or "1y2mo3d4h".
	// Default: 3mo
	StaffSessionDuration string

	// SiteName is the name of the site, displayed in the title and front page header
	// Default: Gochan
	SiteName string

	// SiteSlogan is the community slogan displayed on the front page below the site name
	SiteSlogan string

	// Modboard was intended to be the board that moderators would use to discuss moderation, but it is not currently used.
	// Deprecated: This field is not currently used and may be removed in the future
	Modboard string

	// MaxRecentPosts is the number of recent posts to display on the front page
	// Default: 15
	MaxRecentPosts int

	// RecentPostsWithNoFile determines whether to include posts with no file in the recent posts list
	// Default: false
	RecentPostsWithNoFile bool

	// EnableAppeals determines whether to allow users to appeal bans
	// Default: true
	EnableAppeals bool

	// MinifyHTML tells the server to minify HTML output before sending it to the client
	// Default: true
	MinifyHTML bool

	// MinifyJS tells the server to minify JavaScript and JSON output before sending it to the client
	// Default: true
	MinifyJS bool

	// GeoIPType is the type of GeoIP database to use. Currently only "mmdb" is supported, though other types may be provided by plugins
	GeoIPType string

	// GeoIPOptions is a map of options to pass to the GeoIP plugin
	GeoIPOptions map[string]any

	// Captcha options for spam prevention. Currently only hcaptcha is supported
	Captcha CaptchaConfig

	// FingerprintVideoThumbnails determines whether to use video thumbnails for image fingerprinting. If false, the video file will not be checked by fingerprinting filters
	// Default: false
	FingerprintVideoThumbnails bool

	// FingerprintHashLength is the length of the hash used for image fingerprinting
	// Default: 16
	FingerprintHashLength int

	cookieMaxAgeDuration time.Duration
}

func (sc *SiteConfig) CookieMaxAgeDuration() (time.Duration, error) {
	var err error
	if sc.cookieMaxAgeDuration == 0 {
		sc.cookieMaxAgeDuration, err = durationutil.ParseLongerDuration(sc.CookieMaxAge)
	}
	return sc.cookieMaxAgeDuration, err
}

type CaptchaConfig struct {
	// Type is the type of captcha to use. Currently only "hcaptcha" is supported
	Type string

	// OnlyNeededForThreads determines whether to require a captcha only when creating a new thread, or for all posts
	OnlyNeededForThreads bool

	// SiteKey is the public key for the captcha service. Usage depends on the captcha service
	SiteKey string

	// AccountSecret is the secret key for the captcha service. Usage depends on the captcha service
	AccountSecret string
}

func (cc *CaptchaConfig) UseCaptcha() bool {
	return cc.SiteKey != "" && cc.AccountSecret != ""
}

type BoardCooldowns struct {
	// NewThread is the number of seconds the user must wait before creating new threads.
	// Default: 30
	NewThread int `json:"threads"`

	// NewReply is the number of seconds the user must wait after replying to a thread before they can create another reply.
	// Default: 7
	Reply int `json:"replies"`

	// NewImageReply is the number of seconds the user must wait after replying to a thread with an upload before they can create another reply.
	// Default: 7
	ImageReply int `json:"images"`
}

// PageBanner represents the filename and dimensions of a banner image to display on board and thread pages
type PageBanner struct {
	// Filename is the name of the image file to display as seen by the browser
	Filename string
	// Width is the width of the image in pixels
	Width int
	// Height is the height of the image in pixels
	Height int
}

// BoardConfig contains information about a specific board to be stored in /path/to/board/board.json
// or all boards if it is stored in the main gochan.json file. If a board doesn't have board.json,
// the site's default board config (with values set in gochan.json) will be used
type BoardConfig struct {
	// InheritGlobalStyles determines whether to use the global styles in addition to the board's styles, as opposed to only the board's styles
	// Default: true
	InheritGlobalStyles bool

	// Styles is a list of Gochan themes with Name and Filename fields, choosable by the user
	Styles []Style

	// DefaultStyle is the filename of the default style to use for the board or the site. If it is not set, the first style in the Styles list will be used
	// Default: pipes.css
	DefaultStyle string

	// Banners is a list of banners to display on the board's front page, with Filename, Width, and Height fields
	Banners []PageBanner

	// Lockdown prevents users from posting if true
	// Default: false
	Lockdown bool

	// LockdownMessage is the message displayed to users if they try to cretae a post when the site is in lockdown
	// Default: This imageboard has temporarily disabled posting. We apologize for the inconvenience
	LockdownMessage string

	PostConfig
	UploadConfig

	// DateTimeFormat is the human readable format to use for showing post timestamps. See [the official documentation](https://pkg.go.dev/time#Time.Format) for more information.
	// Default: Mon, January 02, 2006 3:04:05 PM
	DateTimeFormat string

	// ShowPosterID determines whether to show the generated thread-unique poster ID in the post header (not yet implemented)
	// Default: false
	ShowPosterID bool

	// EnableSpoileredImages determines whether to allow users to spoiler images (not yet implemented)
	// Default: true
	EnableSpoileredImages bool

	// EnableSpoileredThreads determines whether to allow users to spoiler threads (not yet implemented)
	// Default: true
	EnableSpoileredThreads bool

	// Worksafe determines whether the board is worksafe or not. If it is set to true, threads cannot be marked NSFW
	// (given a hashtag with the text NSFW, case insensitive).
	// Default: true
	Worksafe bool

	// Cooldowns is used to prevent spamming by setting the number of seconds the user must wait before creating new threads or replies
	Cooldowns BoardCooldowns

	// RenderURLsAsLinks determines whether to render URLs as clickable links in posts
	// Default: true
	RenderURLsAsLinks bool

	// ThreadsPerPage is the number of threads to display per page
	// Default: 20
	ThreadsPerPage int

	// EnableGeoIP shows a dropdown box allowing the user to set their post flag as their country
	// Default: false
	EnableGeoIP bool

	// EnableNoFlag allows the user to post without a flag. It is only used if EnableGeoIP or CustomFlags is true
	// Default: false
	EnableNoFlag bool

	// CustomFlags is a list of non-geoip flags with Name (viewable to the user) and Flag (flag image filename) fields
	CustomFlags []geoip.Country
	isGlobal    bool
}

// CheckCustomFlag returns true if the given flag and name are configured for
// the board (or are globally set)
func (bc *BoardConfig) CheckCustomFlag(flag string) (string, bool) {
	for _, country := range bc.CustomFlags {
		if flag == country.Flag {
			return country.Name, true
		}
	}
	return "", false
}

// IsGlobal returns true if this is the global configuration applied to all
// boards by default, or false if it is an explicitly configured board
func (bc *BoardConfig) IsGlobal() bool {
	return bc.isGlobal
}

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	Name     string
	Filename string
}

type UploadConfig struct {
	// RejectDuplicateUploads determines whether to reject images and videos that have already been uploaded
	// Default: false
	RejectDuplicateUploads bool

	// ThumbWidth is the maximum width that thumbnails in the top thread post will be scaled down to
	// Default: 200
	ThumbWidth int

	// ThumbHeight is the maximum height that thumbnails in the top thread post will be scaled down to
	// Default: 200
	ThumbHeight int

	// ThumbWidthReply is the maximum width that thumbnails in thread replies will be scaled down to
	// Default: 125
	ThumbWidthReply int

	// ThumbHeightReply is the maximum height that thumbnails in thread replies will be scaled down to
	// Default: 125
	ThumbHeightReply int

	// ThumbWidthCatalog is the maximum width that thumbnails on the board catalog page will be scaled down to
	// Default: 50
	ThumbWidthCatalog int

	// ThumbHeightCatalog is the maximum height that thumbnails on the board catalog page will be scaled down to
	// Default: 50
	ThumbHeightCatalog int

	// AllowOtherExtensions is a map of file extensions to use for uploads that are not images or videos
	// The key is the extension (e.g. ".pdf") and the value is the filename of the thumbnail to use in /static
	AllowOtherExtensions map[string]string

	// StripImageMetadata sets what (if any) metadata to remove from uploaded images using exiftool.
	// Valid values are "", "none" (has the same effect as ""), "exif", or "all" (for stripping all metadata)
	StripImageMetadata string

	// ExiftoolPath is the path to the exiftool command. If unset or empty, the system path will be used to find it
	ExiftoolPath string
}

func (uc *UploadConfig) AcceptedExtension(filename string) bool {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	// images
	case ".gif":
		fallthrough
	case ".jfif":
		fallthrough
	case ".jpeg":
		fallthrough
	case ".jpg":
		fallthrough
	case ".png":
		fallthrough
	case ".webp":
		fallthrough
	// videos
	case ".mp4":
		fallthrough
	case ".webm":
		return true
	}
	// other formats as configured
	_, ok := uc.AllowOtherExtensions[ext]
	return ok
}

type PostConfig struct {
	// MaxMessageLength is the maximum number of characters allowed in a post
	// Default: 2000
	MaxMessageLength int

	// ReservedTrips is used for reserving secure tripcodes. It should be a map of input strings to output tripcode strings. For example, if you have `{"abcd":"WXYZ"}` and someone posts with the name Name##abcd, their name will instead show up as Name!!WXYZ on the site.
	ReservedTrips map[string]string

	// ThreadsPerPage is the number of threads to display per page
	// Default: 20
	ThreadsPerPage int

	// RepliesOnBoardPage is the number of replies to display on the board page
	// Default: 3
	RepliesOnBoardPage int

	// StickyRepliesOnBoardPage is the number of replies to display on the board page for sticky threads
	// Default: 1
	StickyRepliesOnBoardPage int

	// NewThreadsRequireUpload determines whether to require an upload to create a new thread
	// Default: false
	NewThreadsRequireUpload bool

	// EnableCyclicThreads allows users to create threads that have a maximum number of replies before the oldest reply is deleted
	// Default: true
	EnableCyclicThreads bool

	// CyclicThreadNumPost determines the number of posts a cyclic thread can have before the oldest post is deleted
	// Default: 500
	CyclicThreadNumPosts int

	// BanColors is a list of colors to use for the ban message with the staff name as the key. If the staff name is not found in the list, the default style color will be used.
	BanColors map[string]string

	// BanMessage is the default message shown on a post that a user was banned for
	// Default: USER WAS BANNED FOR THIS POST
	BanMessage string

	// EmbedWidth is the width of embedded external videos
	// Default: 200
	EmbedWidth int

	// EmbedHeight is the height of embedded external videos
	// Default: 164
	EmbedHeight int

	// EmbedMatchers is a map of site ID keys to objects used to match (via regular expression) URLs and embed them in posts via templates,
	// with an optional image thumbnail if supported. If a URL template is not provided, the video/frame will be embedded directly.
	// If EmbedMatchers is nil, embedding is disabled for the board, or globally if it is in the global configuration.
	EmbedMatchers map[string]EmbedMatcher

	// ImagesOpenNewTab determines whether to open images in a new tab when an image link is clicked
	// Default: true
	ImagesOpenNewTab bool

	// NewTabOnExternalLinks determines whether to open external links in a new tab
	// Default: true
	NewTabOnExternalLinks bool

	// NewTabOnOutlinks is an alias for the NewTabOnExternalLinks field.
	//
	// Deprecated: Use NewTabOnExternalLinks instead
	NewTabOnOutlinks bool `json:",omitempty"`

	// DisableBBcode will disable BBCode to HTML conversion if true
	// Default: false
	DisableBBcode bool

	// AllowDiceRerolls determines whether to allow users to edit posts to reroll dice
	// Default: false
	AllowDiceRerolls bool
}

type EmbedMatcher struct {
	// URLRegex checks the incoming embed and determines if it should be embedded with the EmbedTemplate
	URLRegex string
	// EmbedTemplate is the template for embedding the video in place of an upload
	EmbedTemplate string
	// ImageURLSubmatchIndex is the index of the submatch in the URLRegex that contains the image URL
	// Default: 1
	VideoIDSubmatchIndex *int
	// ThumbnailURLTemplate is the template for embedding the video thumbnail in place of the EmbedTemplate
	// HTML. If it is not set, the video will be embedded directly
	ThumbnailURLTemplate string
}

func WriteConfig() error {
	return cfg.Write()
}

// GetSQLConfig returns SQL configuration info. It returns a value instead of a a pointer to it
// because it is not safe to edit while Gochan is running
func GetSQLConfig() SQLConfig {
	return cfg.SQLConfig
}

// GetSystemCriticalConfig returns system-critical configuration options like listening IP
// It returns a value instead of a pointer, because it is not usually safe to edit while Gochan is running.
func GetSystemCriticalConfig() *SystemCriticalConfig {
	return &cfg.SystemCriticalConfig
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

// UpdateBoardConfig updates or establishes the configuration for the given board
func UpdateBoardConfig(dir string) error {
	ba, err := os.ReadFile(path.Join(cfg.DocumentRoot, dir, "board.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// board doesn't have a custom config, use global config
			return nil
		}
		return err
	}
	boardcfg := cfg.BoardConfig
	if err = json.Unmarshal(ba, &boardcfg); err != nil {
		return err
	}
	boardcfg.isGlobal = false
	boardConfigs[dir] = boardcfg
	return nil
}

// DeleteBoardConfig removes the custom board configuration data, normally should be used
// when a board is deleted
func DeleteBoardConfig(dir string) {
	delete(boardConfigs, dir)
}

func GetVersion() *GochanVersion {
	return cfg.Version
}
