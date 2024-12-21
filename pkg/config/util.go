package config

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path"
	"runtime"
	"strconv"
	"strings"
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
	if runtime.GOOS == "windows" || fp == "" || Cfg.Username == "" {
		// Chown returns an error in Windows so skip it, also skip if Username isn't set
		// because otherwise it'll think we want to switch to uid and gid 0 (root)
		return nil
	}
	return os.Chown(fp, uid, gid)
}

func TakeOwnershipOfFile(f *os.File) error {
	if runtime.GOOS == "windows" || f == nil || Cfg.Username == "" {
		// Chown returns an error in Windows so skip it, also skip if Username isn't set
		// because otherwise it'll think we want to switch to uid and gid 0 (root)
		return nil
	}
	return f.Chown(uid, gid)
}

// InitConfig loads and parses gochan.json on startup and verifies its contents
func InitConfig(versionStr string) {
	Cfg = defaultGochanConfig
	if strings.HasSuffix(os.Args[0], ".test") {
		// create a dummy config for testing if we're using go test
		Cfg = defaultGochanConfig
		Cfg.ListenIP = "127.0.0.1"
		Cfg.Port = 8080
		Cfg.UseFastCGI = true
		Cfg.testing = true
		Cfg.TemplateDir = "templates"
		Cfg.DBtype = "sqlite3"
		Cfg.DBhost = "./testdata/gochantest.db"
		Cfg.DBname = "gochan"
		Cfg.DBusername = "gochan"
		Cfg.SiteDomain = "127.0.0.1"
		Cfg.RandomSeed = "test"
		Cfg.Version = ParseVersion(versionStr)
		Cfg.SiteSlogan = "Gochan testing"
		Cfg.Verbose = true
		Cfg.Captcha.OnlyNeededForThreads = true
		Cfg.Cooldowns = BoardCooldowns{0, 0, 0}
		Cfg.BanColors = []string{
			"admin:#0000A0",
			"somemod:blue",
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

	if err = json.Unmarshal(cfgBytes, Cfg); err != nil {
		fmt.Printf("Error parsing %s: %s", cfgPath, err.Error())
		os.Exit(1)
	}
	Cfg.jsonLocation = cfgPath

	if err = Cfg.ValidateValues(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if runtime.GOOS != "windows" {
		var gcUser *user.User
		if Cfg.Username != "" {
			gcUser, err = user.Lookup(Cfg.Username)
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

	if _, err = os.Stat(Cfg.DocumentRoot); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(Cfg.TemplateDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(Cfg.LogDir); os.IsNotExist(err) {
		err = os.MkdirAll(Cfg.LogDir, DirFileMode)
	}
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	Cfg.LogDir = gcutil.FindResource(Cfg.LogDir, "log", "/var/log/gochan/")

	if Cfg.Port == 0 {
		Cfg.Port = 80
	}

	if len(Cfg.FirstPage) == 0 {
		Cfg.FirstPage = []string{"index.html", "1.html", "firstrun.html"}
	}

	if Cfg.WebRoot == "" {
		Cfg.WebRoot = "/"
	}

	if Cfg.WebRoot[0] != '/' {
		Cfg.WebRoot = "/" + Cfg.WebRoot
	}
	if Cfg.WebRoot[len(Cfg.WebRoot)-1] != '/' {
		Cfg.WebRoot += "/"
	}

	_, zoneOffset := time.Now().Zone()
	Cfg.TimeZone = zoneOffset / 60 / 60

	Cfg.Version = ParseVersion(versionStr)
	Cfg.Version.Normalize()
}

// WebPath returns an absolute path, starting at the web root (which is "/" by default)
func WebPath(part ...string) string {
	return path.Join(Cfg.WebRoot, path.Join(part...))
}
