package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

type pathsForm struct {
	ConfigDir    string `form:"configdir,required,notempty" method:"POST"`
	TemplateDir  string `form:"templatedir,required,notempty" method:"POST"`
	DocumentRoot string `form:"documentroot,required,notempty" method:"POST"`
	LogDir       string `form:"logdir,required,notempty" method:"POST"`
	WebRoot      string `form:"webroot,required,notempty" method:"POST"`
}

func (pf *pathsForm) validateDir(pDir *string, desc string) error {
	dir := *pDir
	if dir == "" {
		return fmt.Errorf("%s is required", desc)
	}
	dir = path.Clean(dir)
	*pDir = dir

	fi, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s %s does not exist", desc, dir)
	}
	if errors.Is(err, fs.ErrPermission) {
		return fmt.Errorf("permission denied to %s", dir)
	}
	if !fi.Mode().IsDir() {
		return fmt.Errorf("%s exists at %s but is not a directory", desc, dir)
	}
	return nil
}

func (pf *pathsForm) validate(warnEv, errEv *zerolog.Event) (err error) {
	pf.DocumentRoot = path.Clean(pf.DocumentRoot)
	pf.LogDir = path.Clean(pf.LogDir)

	if pf.ConfigDir == "" {
		warnEv.Msg("Required config output directory not set")
		return errors.New("config output directory is required")
	}
	pf.ConfigDir = path.Clean(pf.ConfigDir)

	validConfigDirs := []string{".", "/usr/local/etc/gochan", "/etc/gochan"}

	if !slices.Contains(validConfigDirs, pf.ConfigDir) {
		warnEv.Str("configDir", pf.ConfigDir).
			Msg("Invalid config output directory")
		return fmt.Errorf("config output directory %s is not allowed. Valid values are ., /usr/local/etc/gochan, or /etc/gochan", pf.ConfigDir)
	}

	if err = pf.validateDir(&pf.ConfigDir, "config output directory"); err != nil {
		warnEv.Err(err).
			Msg("Invalid config output directory")
		return err
	}

	if _, err = os.Stat(path.Join(pf.ConfigDir, "gochan.json")); err == nil {
		warnEv.Str("configDir", pf.ConfigDir).
			Msg("Config output directory already exists")
		return fmt.Errorf("gochan.json already exists in %s", pf.ConfigDir)
	}

	if err = pf.validateDir(&pf.TemplateDir, "template directory"); err != nil {
		warnEv.Err(err).Str("templateDir", pf.TemplateDir).
			Msg("Invalid template directory")
		return err
	}
	if err = pf.validateDir(&pf.DocumentRoot, "document root"); err != nil {
		warnEv.Err(err).Str("documentRoot", pf.DocumentRoot).
			Msg("Invalid document root")
		return err
	}
	if err = pf.validateDir(&pf.LogDir, "log directory"); err != nil {
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
}

func (dbf *dbForm) validate() (tablesExist bool, err error) {
	var connStr string
	var query string
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch dbf.DBtype {
	case "mysql":
		connStr = fmt.Sprintf(gcsql.MySQLConnStr, dbf.DBuser, dbf.DBpass, dbf.DBhost, dbf.DBname)
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?`
	case "postgres":
		connStr = fmt.Sprintf(gcsql.PostgresConnStr, dbf.DBuser, dbf.DBpass, dbf.DBhost, dbf.DBname)
		query = `SELECT COUNT(*) FROM information_schema.TABLES WHERE table_catalog = CURRENT_DATABASE() AND table_name = ?`
	case "sqlite3":
		connStr = fmt.Sprintf(gcsql.SQLite3ConnStr, dbf.DBhost, dbf.DBuser, dbf.DBpass)
		query = `SELECT COUNT(*) FROM sqlite_master WHERE name = ? AND type = 'table'`
	default:
		return false, gcsql.ErrUnsupportedDB
	}
	db, err := sql.Open(dbf.DBtype, connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	var count int
	stmt, err := db.PrepareContext(ctx, query)
	if err != nil {
		return false, err
	}
	defer stmt.Close()

	if err = stmt.QueryRowContext(ctx, dbf.DBprefix+"database_version").Scan(&count); err != nil {
		return false, err
	}
	tablesExist = count > 0
	if err = stmt.Close(); err != nil {
		return
	}
	if err = db.Close(); err != nil {
		return
	}
	return
}
