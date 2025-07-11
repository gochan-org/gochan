package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

const (
	InitialSetupStatusUnknown InitialSetupStatus = iota
	InitialSetupNotStarted
	InitialSetupComplete

	// DirFileMode is the default file mode for directories created by gochan
	DirFileMode fs.FileMode = 0775
	// NormalFileMode is the default file mode for files created by gochan
	NormalFileMode fs.FileMode = 0664

	// ConfigPathEnvVar is the environment variable used to set the path to gochan.js if it is set
	ConfigPathEnvVar = "GOCHAN_CONFIG"
)

var (
	uid                     int
	gid                     int
	ErrGochanConfigNotFound                    = errors.New("gochan.json not found")
	initialSetupStatus      InitialSetupStatus = InitialSetupStatusUnknown
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

// GetGochanJSONPath returns the location of gochan.json. If the GOCHAN_CONFIG environment variable is set,
// it returns the value, whether or not a file exists at that location. Otherwise, it searches for gochan.json
// in the following locations, returning the first one found:
//
//	./gochan.json (working directory)
//	/usr/local/etc/gochan/gochan.json
//	/opt/homebrew/etc/gochan/gochan.json
//	/etc/gochan/gochan.json
//
// If gochan.json is not found, it returns an empty string.
func GetGochanJSONPath() string {
	if cfgPath != "" {
		return cfgPath
	}
	jsonPath := os.Getenv(ConfigPathEnvVar)
	if jsonPath != "" {
		return jsonPath
	}
	return gcutil.FindResource(StandardConfigSearchPaths...)
}

// GetUser returns the IDs of the user and group gochan should be acting as
// when creating files. If they are 0, it is using the current user
func GetUser() (int, int) {
	return uid, gid
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

// SetSystemCriticalConfig sets system critical configuration values
func SetSystemCriticalConfig(systemCritical *SystemCriticalConfig) {
	setDefaultCfgIfNotSet()
	cfg.SystemCriticalConfig = *systemCritical
}

// SetSiteConfig sets the site configuration values
func SetSiteConfig(siteConfig *SiteConfig) {
	setDefaultCfgIfNotSet()
	cfg.SiteConfig = *siteConfig
}

func loadConfig() (err error) {
	cfg = defaultGochanConfig
	if testing.Testing() {
		// create a dummy config for testing if we're using go test
		cfg = defaultGochanConfig
		cfg.ListenAddress = "127.0.0.1"
		cfg.Port = 8080
		cfg.UseFastCGI = true
		cfg.TemplateDir = "templates"
		cfg.LogDir = "log"
		cfg.DBtype = "sqlite3"
		cfg.DocumentRoot = "html"
		cfg.DBhost = "./testdata/gochantest.db"
		cfg.DBname = "gochan"
		cfg.DBusername = "gochan"
		cfg.SiteHost = "127.0.0.1"
		cfg.RandomSeed = "test"
		cfg.SiteSlogan = "Gochan testing"
		cfg.Cooldowns = BoardCooldowns{0, 0, 0}
		cfg.BanColors = map[string]string{
			"admin":   "#0000A0",
			"somemod": "blue",
		}
		return
	}
	cfgPath = gcutil.FindResource(StandardConfigSearchPaths...)
	if cfgPath == "" {
		return ErrGochanConfigNotFound
	}

	cfgBytes, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("error reading %s: %w", cfgPath, err)
	}

	if err = json.Unmarshal(cfgBytes, cfg); err != nil {
		var unmarshalTypeError *json.UnmarshalTypeError
		if errors.As(err, &unmarshalTypeError) {
			return fmt.Errorf("invalid field type %s in %s: expected %s, found %s",
				unmarshalTypeError.Field, cfgPath, unmarshalTypeError.Type, unmarshalTypeError.Value)
		}
		return fmt.Errorf("error parsing %s: %w", cfgPath, err)
	}
	cfg.jsonLocation = cfgPath
	return nil
}

// InitConfig loads and parses gochan.json on startup and verifies its contents
func InitConfig() (err error) {
	initialSetupStatus = InitialSetupNotStarted
	if err = loadConfig(); err != nil {
		return err
	}

	if err = cfg.ValidateValues(); err != nil {
		return err
	}

	if runtime.GOOS != "windows" {
		var gcUser *user.User
		if cfg.Username != "" {
			gcUser, err = user.Lookup(cfg.Username)
		} else {
			gcUser, err = user.Current()
		}
		if err != nil {
			return err
		}
		if uid, err = strconv.Atoi(gcUser.Uid); err != nil {
			return err
		}

		if gid, err = strconv.Atoi(gcUser.Gid); err != nil {
			return err
		}
	}

	if _, err = os.Stat(cfg.DocumentRoot); err != nil {
		return err
	}
	if _, err = os.Stat(cfg.TemplateDir); err != nil {
		return err
	}
	if _, err = os.Stat(cfg.LogDir); os.IsNotExist(err) {
		err = os.MkdirAll(cfg.LogDir, DirFileMode)
	}
	if err != nil {
		return err
	}

	cfg.LogDir = gcutil.FindResource(cfg.LogDir, "log", "/var/log/gochan/")

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

	_, zoneOffset := time.Now().Zone()
	cfg.TimeZone = zoneOffset / 60 / 60
	initialSetupStatus = InitialSetupComplete
	return nil
}

// WebPath returns an absolute path, starting at the web root (which is "/" by default)
func WebPath(part ...string) string {
	return path.Join(cfg.WebRoot, path.Join(part...))
}
