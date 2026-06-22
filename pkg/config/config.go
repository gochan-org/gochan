package config

import (
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"slices"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

const (
	randomStringSize = 16
	cookieMaxAgeEx   = ` (example: "1 year 2 months 3 days 4 hours", or "1y2mo3d4h"`

	DefaultSQLTimeout            = 15
	DefaultSQLMaxConns           = 10
	DefaultSQLConnMaxLifetimeMin = 3

	GochanVersion = "4.3.0"
)

var (
	cfg     *GochanConfig
	cfgPath string

	boardConfigs              = map[string]BoardConfig{}
	ErrNoMatchingEmbedHandler = errors.New("no matching handler for the embed URL")
)

type InitialSetupStatus int

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
		gcfg.NewTabOnOutlinks = false
		changed = true
	}
	return changed
}

// ValidateValues checks to make sure that the configuration options are usable
// (e.g., ListenAddress is a valid IP address, Port isn't a negative number, etc)
//
// If noWrite is provided and true, the config file will not be rewritten with
// any updated/deprecated fields.
func (gcfg *GochanConfig) ValidateValues(noWrite ...bool) error {
	changed := gcfg.updateDeprecatedFields()

	if gcfg.SiteHost == "" {
		return &InvalidValueError{Field: "SiteHost", Value: gcfg.SiteHost, Details: "must be set"}
	}
	if strings.Contains(gcfg.SiteHost, " ") || strings.Contains(gcfg.SiteHost, "://") {
		return &InvalidValueError{Field: "SiteHost", Value: gcfg.SiteHost, Details: "must be a valid host (port optional)"}
	}

	if gcfg.CookieMaxAge == "" {
		gcfg.CookieMaxAge = defaultGochanConfig.CookieMaxAge
		changed = true
	}
	_, err := durationutil.ParseLongerDuration(gcfg.CookieMaxAge)
	if errors.Is(err, durationutil.ErrInvalidDurationString) {
		return &InvalidValueError{Field: "CookieMaxAge", Value: gcfg.CookieMaxAge, Details: err.Error() + cookieMaxAgeEx}
	} else if err != nil {
		return err
	}

	if gcfg.StaffSessionDuration == "" {
		gcfg.StaffSessionDuration = defaultGochanConfig.StaffSessionDuration
		changed = true
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
	if slices.Contains(drivers, gcfg.DBtype) {
		found = true
	}
	if !found {
		return &InvalidValueError{
			Field:   "DBtype",
			Value:   gcfg.DBtype,
			Details: "currently supported values: " + strings.Join(drivers, ",")}
	}

	if gcfg.LogLevelStr == "" {
		gcfg.LogLevelStr = "info"
		gcfg.logLevelParsed = true
		changed = true
	}
	if gcfg.logLevel, err = zerolog.ParseLevel(gcfg.LogLevelStr); err != nil {
		return &InvalidValueError{
			Field:   "LogLevel",
			Value:   gcfg.LogLevelStr,
			Details: "valid values are trace, debug, info, warn, error, fatal, and panic, or empty string for info level",
		}
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

	if err = gcfg.validateBoardConfig(); err != nil {
		return err
	}

	if !changed {
		return nil
	}
	if len(noWrite) > 0 && noWrite[0] {
		return nil
	}
	return gcfg.Write()
}

func (gcfg *GochanConfig) Write() error {
	fd, err := os.OpenFile(gcfg.jsonLocation, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, NormalFileMode)
	if err != nil {
		return err
	}
	defer fd.Close()

	enc := json.NewEncoder(fd)
	enc.SetIndent("", "\t")
	enc.SetEscapeHTML(false) // prevent things like <, >, & from being escaped
	return enc.Encode(gcfg)
}

// 	str, err := json.MarshalIndent(gcfg, "", "\t")
// 	if err != nil {
// 		return err
// 	}
// 	if testing.Testing() {
// 		// don't try to write anything if we're doing a test
// 		return nil
// 	}
// 	return os.WriteFile(gcfg.jsonLocation, str, NormalFileMode)
// }

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

	// LogLevel determines the minimum level of log event to output. Any events lower than this level will be ignored.
	// Valid values are "trace", "debug", "info", "warn", "error", "fatal", and "panic".
	// Default: info
	LogLevelStr string `json:"LogLevel"`

	// RandomSeed is a random string used for generating secure tokens. It will be generated if not set and must not be changed
	RandomSeed string

	TimeZone int `json:"-"`

	// ExiftoolPath is the path to the exiftool command. If unset or empty, the system path will be used to find it
	ExiftoolPath string

	logLevel       zerolog.Level
	logLevelParsed bool
}

// LogLevel returns the minimum log event level to write to the log file
func (scc *SystemCriticalConfig) LogLevel() zerolog.Level {
	if !scc.logLevelParsed {
		scc.logLevel = zerolog.InfoLevel
		if scc.LogLevelStr != "" {
			scc.logLevel, _ = zerolog.ParseLevel(scc.LogLevelStr)
		}
		scc.logLevelParsed = true
	}
	return scc.logLevel
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
	Captcha *CaptchaConfig

	// FingerprintVideoThumbnails determines whether to use video thumbnails for image fingerprinting. If false, the video file will not be checked by fingerprinting filters
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

type EmbedTemplateData struct {
	MediaID     string
	HandlerID   string
	ThumbWidth  int
	ThumbHeight int
	MediaURL    string
}

type EmbedMatcher struct {
	// URLRegex checks the incoming embed and determines if it should be embedded with the EmbedTemplate
	URLRegex string

	// EmbedTemplate is the template for embedding the media in place of an upload. It uses the MediaID, HandlerID,
	// ThumbWidth, ThumbHeight fields of EmbedMediaData
	EmbedTemplate string

	// MediaIDSubmatchIndex is the index of the submatch in the URLRegex that contains the media ID
	// Default: 1
	MediaIDSubmatchIndex *int

	// ThumbnailURLTemplate is the template for embedding the media thumbnail in place of the EmbedTemplate
	// HTML. If it is not set, the media will be embedded directly. It uses the MediaID field of EmbedMediaData
	ThumbnailURLTemplate string

	// MediaURLTemplate is used to construct the media URL from the media ID. It uses the MediaID field of EmbedMediaData
	MediaURLTemplate string
}

func (em *EmbedMatcher) HasThumbnail() bool {
	return em.ThumbnailURLTemplate != ""
}

func GetInitialSetupStatus() InitialSetupStatus {
	return initialSetupStatus
}

func GetDefaultConfig() *GochanConfig {
	return defaultGochanConfig
}

func WriteConfig(path ...string) error {
	if cfg == nil {
		return errors.New("configuration not loaded")
	}
	if len(path) > 0 {
		cfg.jsonLocation = path[0]
	}
	if cfg.jsonLocation == "" {
		return errors.New("configuration file path not set")
	}
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
