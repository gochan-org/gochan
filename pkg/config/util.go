package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"time"

	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// MissingField represents a field missing from the configuration file
type MissingField struct {
	Name        string
	Critical    bool
	Description string
}

// ErrInvalidValue represents a GochanConfig field with a bad value
type ErrInvalidValue struct {
	Field   string
	Value   interface{}
	Details string
}

func (iv *ErrInvalidValue) Error() string {
	str := fmt.Sprintf("invalid %s value: %#v", iv.Field, iv.Value)
	if iv.Details != "" {
		str += " - " + iv.Details
	}
	return str
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
		if cfgDefaults[fType.Name] != nil {
			// the field isn't in the JSON file but has a default value that we can use
			fVal.Set(reflect.ValueOf(cfgDefaults[fType.Name]))
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
	cfgPath = gcutil.FindResource("gochan.json", "/etc/gochan/gochan.json")
	if cfgPath == "" {
		fmt.Println("gochan.json not found")
		os.Exit(1)
	}

	jfile, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", cfgPath, err.Error())
		os.Exit(1)
	}

	var fields []MissingField
	Config, fields, err = ParseJSON(jfile)
	if err != nil {
		fmt.Printf("Error parsing %s: %s", cfgPath, err.Error())
	}
	Config.jsonLocation = cfgPath

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

	if err = Config.ValidateValues(); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if _, err = os.Stat(Config.DocumentRoot); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(Config.TemplateDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	if _, err = os.Stat(Config.LogDir); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	Config.LogDir = gcutil.FindResource(Config.LogDir, "log", "/var/log/gochan/")
	if err = gclog.InitLogs(
		path.Join(Config.LogDir, "access.log"),
		path.Join(Config.LogDir, "error.log"),
		path.Join(Config.LogDir, "staff.log"),
		Config.DebugMode); err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if Config.Port == 0 {
		Config.Port = 80
	}

	if len(Config.FirstPage) == 0 {
		Config.FirstPage = []string{"index.html", "1.html", "firstrun.html"}
	}

	if Config.SiteWebfolder == "" {
		Config.SiteWebfolder = "/"
	}

	if Config.SiteWebfolder[0] != '/' {
		Config.SiteWebfolder = "/" + Config.SiteWebfolder
	}
	if Config.SiteWebfolder[len(Config.SiteWebfolder)-1] != '/' {
		Config.SiteWebfolder += "/"
	}

	if Config.EnableGeoIP {
		if _, err = os.Stat(Config.GeoIPDBlocation); err != nil {
			gclog.Print(gclog.LErrorLog|gclog.LStdLog, "Unable to find GeoIP file location set in gochan.json, disabling GeoIP")
		}
		Config.EnableGeoIP = false
	}

	_, zoneOffset := time.Now().Zone()
	Config.TimeZone = zoneOffset / 60 / 60

	Config.Version = ParseVersion(versionStr)
	Config.Version.Normalize()
}
