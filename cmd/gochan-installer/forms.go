package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

type pathsForm struct {
	ConfigPath   string `form:"configdir,required,notempty" method:"POST"`
	TemplateDir  string `form:"templatedir,required,notempty" method:"POST"`
	DocumentRoot string `form:"documentroot,required,notempty" method:"POST"`
	LogDir       string `form:"logdir,required,notempty" method:"POST"`
	WebRoot      string `form:"webroot,required" method:"POST"`
}

func (pf *pathsForm) validatePath(targetPath *string, desc string, expectDir bool) error {
	p := *targetPath
	if p == "" {
		return fmt.Errorf("%s is required", desc)
	}
	p = path.Clean(p)
	*targetPath = p

	fi, err := os.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s %s does not exist", desc, p)
	}
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("permission denied to %s", p)
	}
	if expectDir && !fi.Mode().IsDir() {
		return fmt.Errorf("%s exists at %s but is not a directory", desc, p)
	}
	return nil
}

func (pf *pathsForm) validate(warnEv, errEv *zerolog.Event) (err error) {
	pf.DocumentRoot = path.Clean(pf.DocumentRoot)
	pf.LogDir = path.Clean(pf.LogDir)

	if pf.ConfigPath == "" {
		warnEv.Msg("Required config output directory not set")
		return errors.New("config output directory is required")
	}
	pf.ConfigPath = path.Clean(pf.ConfigPath)

	validConfigPaths := cfgPaths
	pathsArr := zerolog.Arr()
	for _, p := range validConfigPaths {
		pathsArr.Str(p)
	}

	if !slices.Contains(validConfigPaths, pf.ConfigPath) {
		warnEv.Str("configPath", pf.ConfigPath).
			Array("validConfigPaths", pathsArr).
			Msg("Invalid config output path")
		return fmt.Errorf("config output path %s is not allowed. Valid values are %s", strings.Join(cfgPaths, ", "), pf.ConfigPath)
	}

	configDir := path.Dir(pf.ConfigPath)

	if err = pf.validatePath(&configDir, "config output directory", true); err != nil {
		warnEv.Err(err).
			Msg("Invalid config output directory")
		return err
	}

	if err = pf.validatePath(&pf.TemplateDir, "template directory", true); err != nil {
		warnEv.Err(err).Str("templateDir", pf.TemplateDir).
			Msg("Invalid template directory")
		return err
	}
	if err = pf.validatePath(&pf.DocumentRoot, "document root", true); err != nil {
		warnEv.Err(err).Str("documentRoot", pf.DocumentRoot).
			Msg("Invalid document root")
		return err
	}
	if err = pf.validatePath(&pf.LogDir, "log directory", true); err != nil {
		warnEv.Err(err).Str("logDir", pf.LogDir).
			Msg("Invalid log directory")
		return err
	}

	if pf.WebRoot == "" {
		pf.WebRoot = "/"
	}
	return nil
}

type dbForm struct {
	DBtype   string `form:"dbtype,required,notempty" method:"POST"`
	DBhost   string `form:"dbhost,required,notempty" method:"POST"`
	DBname   string `form:"dbname,required,notempty" method:"POST"`
	DBuser   string `form:"dbuser,required,notempty" method:"POST"`
	DBpass   string `form:"dbpass" method:"POST"`
	DBprefix string `form:"dbprefix" method:"POST"`

	TimeoutSeconds     int `form:"timeoutseconds,required" method:"POST"`
	MaxOpenConns       int `form:"maxopenconns,required" method:"POST"`
	MaxIdleConns       int `form:"maxidleconns,required" method:"POST"`
	ConnMaxLifetimeMin int `form:"connmaxlifetimemin,required" method:"POST"`
}

func (dbf *dbForm) validate() (status dbStatus, err error) {
	if dbf.DBprefix == "" {
		return dbStatusNoPrefix, nil
	}
	if dbf.TimeoutSeconds <= 0 {
		return dbStatusUnknown, errors.New("request timeout must be greater than 0")
	}
	if dbf.MaxOpenConns <= 0 {
		return dbStatusUnknown, errors.New("max open connections must be greater than 0")
	}
	if dbf.MaxIdleConns <= 0 {
		return dbStatusUnknown, errors.New("max idle connections must be greater than 0")
	}
	if dbf.ConnMaxLifetimeMin <= 0 {
		return dbStatusUnknown, errors.New("max lifetime for connections must be greater than 0")
	}

	if err := gcsql.ConnectToDB(&config.SQLConfig{
		// using a dummy config to test connection. It will be set as the main config later
		DBtype:     dbf.DBtype,
		DBhost:     dbf.DBhost,
		DBname:     dbf.DBname,
		DBusername: dbf.DBuser,
		DBpassword: dbf.DBpass,
		DBprefix:   dbf.DBprefix,

		DBTimeoutSeconds:     dbf.TimeoutSeconds,
		DBMaxOpenConnections: dbf.MaxOpenConns,
		DBMaxIdleConnections: dbf.MaxIdleConns,
		DBConnMaxLifetimeMin: dbf.ConnMaxLifetimeMin,
	}); err != nil {
		return dbStatusUnknown, err
	}
	tablesExist, err := gcsql.DoesGochanPrefixTableExist()
	if err != nil {
		return dbStatusUnknown, err
	}
	if tablesExist {
		status = dbStatusTablesExist
	} else {
		status = dbStatusClean
	}
	return
}

type staffForm struct {
	Username        string `form:"username,required,notempty" method:"POST"`
	Password        string `form:"password,required,notempty" method:"POST"`
	ConfirmPassword string `form:"confirmpassword,required,notempty" method:"POST"`
	ToMisc          string `form:"to-misc" method:"POST"`
}

func (sf *staffForm) validate() (err error) {
	if sf.Password != sf.ConfirmPassword {
		return errors.New("passwords do not match")
	}

	return nil
}
