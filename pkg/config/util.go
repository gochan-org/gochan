package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	GC_DIR_MODE  fs.FileMode = 0775
	GC_FILE_MODE fs.FileMode = 0664
)

var (
	criticalFields = []string{
		"ListenIP", "Port", "Username", "UseFastCGI", "DocumentRoot", "TemplateDir", "LogDir",
		"DBtype", "DBhost", "DBname", "DBusername", "DBpassword", "SiteDomain", "Styles",
	}
	uid int
	gid int
)

// MissingField represents a field missing from the configuration file
type MissingField struct {
	Name        string
	Critical    bool
	Description string
}

// InvalidValueError represents a GochanConfig field with a bad value
type InvalidValueError struct {
	Field   string
	Value   any
	Details string
}

func (iv *InvalidValueError) Error() string {
	str := fmt.Sprintf("invalid %s value: %#v", iv.Field, iv.Value)
	if iv.Details != "" {
		str += " - " + iv.Details
	}
	return str
}

func getDefaultSetting[E any](key string) (E, error) {
	defaultValInterface, ok := defaults[key]
	var defaultVal E
	if !ok {
		return defaultVal, nil // not set in defaults map, use the default value for the type
	}
	defaultVal, ok = defaultValInterface.(E)
	if ok {
		return defaultVal, nil
	}
	return defaultVal, &InvalidValueError{
		Field:   key,
		Value:   defaultValInterface,
		Details: "invalid value type",
	}
}

func TakeOwnership(fp string) (err error) {
	if runtime.GOOS == "windows" || fp == "" || cfg.Username == "" {
		// Chown returns an error in Windows so skip it, also skip if Username isn't set
		// because otherwise it'll think we want to switch to uid and gid 0 (root)
		return nil
	}
	return os.Chown(fp, uid, gid)
}

func TakeOwnershipOfFile(f *os.File) error {
	if runtime.GOOS == "windows" || f == nil || cfg.Username == "" {
		// Chown returns an error in Windows so skip it, also skip if Username isn't set
		// because otherwise it'll think we want to switch to uid and gid 0 (root)
		return nil
	}
	return f.Chown(uid, gid)
}

// ParseJSON loads and parses JSON data, returning a GochanConfig pointer, any critical missing
// fields that don't have defaults, and any error from parsing the file. This doesn't mean that the
// values are valid, just that they exist
func ParseJSON(ba []byte) (*GochanConfig, []MissingField, error) {
	var missing []MissingField
	cfg := &GochanConfig{}
	err := json.Unmarshal(ba, cfg)
	if err != nil {
		// checking for malformed JSON, invalid field types
		return cfg, nil, err
	}

	var checker map[string]interface{} // using this for checking for missing fields
	json.Unmarshal(ba, &checker)

	cVal := reflect.ValueOf(cfg).Elem()
	cType := reflect.TypeOf(*cfg)
	numFields := cType.NumField()
	for f := 0; f < numFields; f++ {
		fType := cType.Field(f)
		fVal := cVal.Field(f)
		critical := fType.Tag.Get("critical") == "true"
		if !fVal.CanSet() || fType.Tag.Get("json") == "-" {
			// field is unexported and isn't read from the JSON file
			continue
		}

		if checker[fType.Name] != nil {
			// field is in the JSON file
			continue
		}
		if defaults[fType.Name] != nil {
			// the field isn't in the JSON file but has a default value that we can use
			fVal.Set(reflect.ValueOf(defaults[fType.Name]))
			continue
		}
		if critical {
			// the field isn't in the JSON file and has no default value
			missing = append(missing, MissingField{
				Name:        fType.Name,
				Description: fType.Tag.Get("description"),
				Critical:    critical,
			})
		}
	}
	return cfg, missing, err
}

// InitConfig loads and parses gochan.json on startup and verifies its contents
func InitConfig(versionStr string) {
	if flag.Lookup("test.v") != nil {
		// create a dummy config for testing if we're using go test
		cfg = &GochanConfig{
			testing: true,
			SystemCriticalConfig: SystemCriticalConfig{
				ListenIP:     "127.0.0.1",
				Port:         8080,
				UseFastCGI:   true,
				DebugMode:    true,
				DocumentRoot: "html",
				TemplateDir:  "templates",
				LogDir:       "",
				DBtype:       "sqlite3",
				DBhost:       "./testdata/gochantest.db",
				DBname:       "gochan",
				DBusername:   "gochan",
				DBpassword:   "",
				DBprefix:     "gc_",
				SiteDomain:   "127.0.0.1",
				WebRoot:      "/",
				RandomSeed:   "abcd",
				Version:      ParseVersion(versionStr),
			},
			SiteConfig: SiteConfig{
				Username:        "",
				FirstPage:       []string{"index.html", "firstrun.html", "1.html"},
				Lockdown:        false,
				LockdownMessage: "This imageboard has temporarily disabled posting. We apologize for the inconvenience",
				SiteName:        "Gochan",
				SiteSlogan:      "Gochan testing",
				MinifyHTML:      true,
				MinifyJS:        true,
				EnableAppeals:   true,
				MaxLogDays:      14,
				Verbosity:       1,

				MaxRecentPosts:        12,
				RecentPostsWithNoFile: false,
				Captcha: CaptchaConfig{
					OnlyNeededForThreads: true,
				},
			},
			BoardConfig: BoardConfig{
				Sillytags:    []string{"Admin", "Mod", "Janitor", "Dweeb", "Kick me", "Troll", "worst pony"},
				UseSillytags: false,
				Styles: []Style{
					{Name: "Pipes", Filename: "pipes.css"},
					{Name: "BunkerChan", Filename: "bunkerchan.css"},
					{Name: "Burichan", Filename: "burichan.css"},
					{Name: "Clear", Filename: "clear.css"},
					{Name: "Dark", Filename: "dark.css"},
					{Name: "Photon", Filename: "photon.css"},
					{Name: "Yotsuba", Filename: "yotsuba.css"},
					{Name: "Yotsuba B", Filename: "yotsubab.css"},
					{Name: "Windows 9x", Filename: "win9x.css"},
				},
				DefaultStyle: "pipes.css",

				Cooldowns: BoardCooldowns{
					NewThread: 30,
					Reply:     7,
				},
				PostConfig: PostConfig{
					ThreadsPerPage:           15,
					RepliesOnBoardPage:       3,
					StickyRepliesOnBoardPage: 1,
					BanColors: []string{
						"admin:#0000A0",
						"somemod:blue",
					},
					BanMessage:       "USER WAS BANNED FOR THIS POST",
					EnableEmbeds:     true,
					EmbedWidth:       200,
					EmbedHeight:      164,
					ImagesOpenNewTab: true,
					NewTabOnOutlinks: true,
				},
				UploadConfig: UploadConfig{
					ThumbWidth:         200,
					ThumbHeight:        200,
					ThumbWidthReply:    125,
					ThumbHeightReply:   125,
					ThumbWidthCatalog:  50,
					ThumbHeightCatalog: 50,
				},
				DateTimeFormat: "Mon, January 02, 2006 3:04 PM",
			},
		}
		return
	}
	cfgPath = gcutil.FindResource(
		"gochan.json",
		"/usr/local/etc/gochan/gochan.json",
		"/etc/gochan/gochan.json")
	if cfgPath == "" {
		fmt.Println("gochan.json not found")
		os.Exit(1)
	}

	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	var fields []MissingField
	cfg, fields, err = ParseJSON(cfgBytes)
	if err != nil {
		fmt.Printf("Error parsing %s: %s", cfgPath, err.Error())
	}
	cfg.jsonLocation = cfgPath

	numMissing := 0
	for _, missing := range fields {
		fmt.Println("Missing field:", missing.Name)
		if missing.Description != "" {
			fmt.Println("Description:", missing.Description)
		}
		numMissing++
	}
	if numMissing > 0 {
		fmt.Println("gochan failed to load the configuration file because there are fields missing.\nSee gochan.example.json in sample-configs for an example configuration file")
		os.Exit(1)
	}

	if err = cfg.ValidateValues(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if runtime.GOOS != "windows" {
		var gcUser *user.User
		if cfg.Username != "" {
			gcUser, err = user.Lookup(cfg.Username)
		} else {
			gcUser, err = user.Current()
		}
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		if uid, err = strconv.Atoi(gcUser.Uid); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}

		if gid, err = strconv.Atoi(gcUser.Gid); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	if _, err = os.Stat(cfg.DocumentRoot); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(cfg.TemplateDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(cfg.LogDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	cfg.LogDir = gcutil.FindResource(cfg.LogDir, "log", "/var/log/gochan/")
	if err = gcutil.InitLogs(cfg.LogDir, cfg.DebugMode, uid, gid); err != nil {
		fmt.Println("Error opening logs:", err.Error())
		os.Exit(1)
	}

	if cfg.Port == 0 {
		cfg.Port = 80
	}

	if len(cfg.FirstPage) == 0 {
		cfg.FirstPage = []string{"index.html", "1.html", "firstrun.html"}
	}

	if cfg.WebRoot == "" {
		cfg.WebRoot = "/"
	}

	if cfg.WebRoot[0] != '/' {
		cfg.WebRoot = "/" + cfg.WebRoot
	}
	if cfg.WebRoot[len(cfg.WebRoot)-1] != '/' {
		cfg.WebRoot += "/"
	}

	if cfg.EnableGeoIP {
		if _, err = os.Stat(cfg.GeoIPDBlocation); err != nil {
			gcutil.LogError(err).
				Str("location", cfg.GeoIPDBlocation).
				Msg("Unable to load GeoIP file location set in gochan.json, disabling GeoIP")
		}
		cfg.EnableGeoIP = false
	}

	_, zoneOffset := time.Now().Zone()
	cfg.TimeZone = zoneOffset / 60 / 60

	cfg.Version = ParseVersion(versionStr)
	cfg.Version.Normalize()
}

// TODO: use reflect to check if the field exists in SystemCriticalConfig
func fieldIsCritical(field string) bool {
	for _, cF := range criticalFields {
		if field == cF {
			return true
		}
	}
	return false
}

// WebPath returns an absolute path, starting at the web root (which is "/" by default)
func WebPath(part ...string) string {
	return path.Join(cfg.WebRoot, path.Join(part...))
}

// UpdateFromMap updates the configuration with the given key->values for use in things like the
// config editor page and possibly others
func UpdateFromMap(m map[string]interface{}, validate bool) error {
	for key, val := range m {
		if fieldIsCritical(key) {
			// don't mess with critical/read-only fields (ListenIP, DocumentRoot, etc)
			// after the server has started
			continue
		}
		cfg.setField(key, val)
	}
	if validate {
		return cfg.ValidateValues()
	}
	return nil
}
