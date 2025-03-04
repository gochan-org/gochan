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
	DirFileMode    fs.FileMode = 0775
	NormalFileMode fs.FileMode = 0664
)

var (
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

func loadConfig(versionStr string, searchPaths ...string) (err error) {
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
		cfg.Version = ParseVersion(versionStr)
		cfg.SiteSlogan = "Gochan testing"
		cfg.Verbose = true
		cfg.Captcha.OnlyNeededForThreads = true
		cfg.Cooldowns = BoardCooldowns{0, 0, 0}
		cfg.BanColors = map[string]string{
			"admin":   "#0000A0",
			"somemod": "blue",
		}
		return
	}
	cfgPath = gcutil.FindResource(searchPaths...)
	if cfgPath == "" {
		return errors.New("gochan.json not found")
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
func InitConfig(versionStr string) (err error) {
	var searchPaths []string
	if !testing.Testing() {
		searchPaths = []string{"gochan.json", "/usr/local/etc/gochan/gochan.json", "/etc/gochan/gochan.json"}
	}
	if err = loadConfig(versionStr, searchPaths...); err != nil {
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

	cfg.Version = ParseVersion(versionStr)
	cfg.Version.Normalize()
	return nil
}

// WebPath returns an absolute path, starting at the web root (which is "/" by default)
func WebPath(part ...string) string {
	return path.Join(cfg.WebRoot, path.Join(part...))
}
