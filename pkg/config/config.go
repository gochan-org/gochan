package config

import (
	"encoding/json"
	"errors"
	"net"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	randomStringSize = 16
	cookieMaxAgeEx   = ` (example: "1 year 2 months 3 days 4 hours", or "1y2mo3d4h"`
)

var (
	cfg     *GochanConfig
	cfgPath string

	boardConfigs    = map[string]BoardConfig{}
	acceptedDrivers = []string{"mysql", "postgres", "sqlite3"}
)

type GochanConfig struct {
	SystemCriticalConfig
	SiteConfig
	BoardConfig
	jsonLocation string `json:"-"`
	testing      bool
}

func (gcfg *GochanConfig) setField(field string, value interface{}) {
	structValue := reflect.ValueOf(gcfg).Elem()
	structFieldValue := structValue.FieldByName(field)
	if !structFieldValue.IsValid() {
		return
	}
	if !structFieldValue.CanSet() {
		return
	}
	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		return
	}

	structFieldValue.Set(val)
}

// ValidateValues checks to make sure that the configuration options are usable
// (e.g., ListenIP is a valid IP address, Port isn't a negative number, etc)
func (gcfg *GochanConfig) ValidateValues() error {
	if net.ParseIP(gcfg.ListenIP) == nil {
		return &InvalidValueError{Field: "ListenIP", Value: gcfg.ListenIP}
	}
	changed := false

	_, err := durationutil.ParseLongerDuration(gcfg.CookieMaxAge)
	if errors.Is(err, durationutil.ErrInvalidDurationString) {
		return &InvalidValueError{Field: "CookieMaxAge", Value: gcfg.CookieMaxAge, Details: err.Error() + cookieMaxAgeEx}
	} else if err != nil {
		return err
	}

	if gcfg.DBtype == "postgresql" {
		gcfg.DBtype = "postgres"
	}
	found := false
	for _, driver := range acceptedDrivers {
		if gcfg.DBtype == driver {
			found = true
			break
		}
	}
	if !found {
		return &InvalidValueError{
			Field:   "DBtype",
			Value:   gcfg.DBtype,
			Details: "currently supported values: " + strings.Join(acceptedDrivers, ",")}
	}

	if gcfg.EnableGeoIP && gcfg.GeoIPDBlocation == "" {
		return &InvalidValueError{
			Field:   "GeoIPDBlocation",
			Value:   "",
			Details: "GeoIPDBlocation must be set in gochan.json if EnableGeoIP is true",
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
					Field: "ExiftoolPath", Value: "", Details: "unable to find exiftool in the system path",
				}
			}
		} else {
			if _, err = exec.LookPath(gcfg.ExiftoolPath); err != nil {
				return &InvalidValueError{
					Field: "ExiftoolPath", Value: gcfg.ExiftoolPath, Details: "unable to find exiftool at the given location",
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
	if gcfg.testing {
		// don't try to write anything if we're doing a test
		return nil
	}
	return os.WriteFile(gcfg.jsonLocation, str, GC_FILE_MODE)
}

/*
SystemCriticalConfig contains configuration options that are extremely important, and fucking with them while
the server is running could have site breaking consequences. It should only be changed by modifying the configuration
file and restarting the server.
*/
type SystemCriticalConfig struct {
	ListenIP       string
	Port           int
	UseFastCGI     bool
	DocumentRoot   string
	TemplateDir    string
	LogDir         string
	Plugins        []string
	PluginSettings map[string]any

	SiteHeaderURL string
	WebRoot       string
	SiteDomain    string

	DBtype     string
	DBhost     string
	DBname     string
	DBusername string
	DBpassword string
	DBprefix   string

	DebugMode  bool
	RandomSeed string
	Version    *GochanVersion `json:"-"`
	TimeZone   int            `json:"-"`
}

// SiteConfig contains information about the site/community, e.g. the name of the site, the slogan (if set),
// the first page to look for if a directory is requested, etc
type SiteConfig struct {
	FirstPage       []string
	Username        string
	CookieMaxAge    string
	Lockdown        bool
	LockdownMessage string

	SiteName   string
	SiteSlogan string
	Modboard   string

	MaxRecentPosts        int
	RecentPostsWithNoFile bool
	Verbosity             int
	EnableAppeals         bool
	MaxLogDays            int

	MinifyHTML      bool
	MinifyJS        bool
	GeoIPDBlocation string
	GeoIPDBType     string
	Captcha         CaptchaConfig
}

func (sc *SiteConfig) validGeoIPType() bool {
	return sc.GeoIPDBType == "" || sc.GeoIPDBType == "legacy" || sc.GeoIPDBType == "geoip2"
}

type CaptchaConfig struct {
	Type                 string
	OnlyNeededForThreads bool
	SiteKey              string
	AccountSecret        string
}

func (cc *CaptchaConfig) UseCaptcha() bool {
	return cc.SiteKey != "" && cc.AccountSecret != ""
}

type BoardCooldowns struct {
	NewThread  int `json:"threads"`
	Reply      int `json:"replies"`
	ImageReply int `json:"images"`
}

type PageBanner struct {
	Filename string
	Width    int
	Height   int
}

// BoardConfig contains information about a specific board to be stored in /path/to/board/board.json
// If a board doesn't have board.json, the site's default board config (with values set in gochan.json) will be used
type BoardConfig struct {
	InheritGlobalStyles bool
	Styles              []Style
	DefaultStyle        string
	Banners             []PageBanner

	PostConfig
	UploadConfig

	DateTimeFormat         string
	ShowPosterID           bool
	EnableSpoileredImages  bool
	EnableSpoileredThreads bool
	Worksafe               bool
	ThreadPage             int
	Cooldowns              BoardCooldowns
	RenderURLsAsLinks      bool
	ThreadsPerPage         int
	EnableGeoIP            bool
	isGlobal               bool
}

// IsGlobal returns
func (bc *BoardConfig) IsGlobal() bool {
	return bc.isGlobal
}

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	Name     string
	Filename string
}

type UploadConfig struct {
	RejectDuplicateImages bool
	ThumbWidth            int
	ThumbHeight           int
	ThumbWidthReply       int
	ThumbHeightReply      int
	ThumbWidthCatalog     int
	ThumbHeightCatalog    int

	AllowOtherExtensions map[string]string

	// Sets what (if any) metadata to remove from uploaded images using exiftool.
	// Valid values are "", "none" (has the same effect as ""), "exif", or "all" (for stripping all metadata)
	StripImageMetadata string
	// The path to the exiftool command. If unset or empty, the system path will be used to find it
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
	MaxLineLength int
	ReservedTrips []string

	ThreadsPerPage           int
	RepliesOnBoardPage       int
	StickyRepliesOnBoardPage int
	NewThreadsRequireUpload  bool

	BanColors        []string
	BanMessage       string
	EmbedWidth       int
	EmbedHeight      int
	EnableEmbeds     bool
	ImagesOpenNewTab bool
	NewTabOnOutlinks bool
	DisableBBcode    bool
}

func WriteConfig() error {
	return cfg.Write()
}

// GetSystemCriticalConfig returns system-critical configuration options like listening IP
// Unlike the other functions returning the sub-configs (GetSiteConfig, GetBoardConfig, etc),
// GetSystemCriticalConfig returns the value instead of a pointer to it, because it is not usually
// safe to edit while Gochan is running.
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

func GetDebugMode() bool {
	return cfg.testing || cfg.SystemCriticalConfig.DebugMode
}

func GetVersion() *GochanVersion {
	return cfg.Version
}

// SetVersion should (in most cases) only be used for tests, where a config file wouldn't be loaded
func SetVersion(version string) {
	if cfg == nil {
		cfg = &GochanConfig{}
		cfg.Version = ParseVersion(version)
	}
}
