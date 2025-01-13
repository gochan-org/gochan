package building

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

func TestBuildJS(t *testing.T) {
	testRoot, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}

	outDir := t.TempDir()
	config.SetVersion("3.11.0")
	systemCriticalCfg := config.GetSystemCriticalConfig()
	systemCriticalCfg.DocumentRoot = path.Join(outDir, "html")
	systemCriticalCfg.TemplateDir = path.Join(testRoot, "templates")
	systemCriticalCfg.LogDir = path.Join(outDir, "logs")
	systemCriticalCfg.WebRoot = "/chan"
	systemCriticalCfg.TimeZone = 8
	config.SetSystemCriticalConfig(systemCriticalCfg)

	boardCfg := config.GetBoardConfig("")
	boardCfg.Styles = []config.Style{
		{Name: "test1", Filename: "test1.css"},
		{Name: "test2", Filename: "test2.css"},
	}
	boardCfg.DefaultStyle = "test1.css"

	serverutil.InitMinifier()

	os.MkdirAll(path.Join(systemCriticalCfg.DocumentRoot, "js"), config.DirFileMode)
	if err = BuildJS(); !assert.NoError(t, err) {
		return
	}

	jsFile, err := os.Open(path.Join(systemCriticalCfg.DocumentRoot, "js/consts.js"))
	if !assert.NoError(t, err) {
		return
	}
	defer jsFile.Close()
	ba, err := io.ReadAll(jsFile)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, expectedMinifiedJS, string(ba))

	siteCfg := config.GetSiteConfig()
	siteCfg.MinifyJS = false

	if err = BuildJS(); !assert.NoError(t, err) {
		return
	}
	jsFile.Seek(0, io.SeekStart)
	if ba, err = io.ReadAll(jsFile); !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, jsFile.Close())
	assert.Equal(t, expectedUnminifiedJS, string(ba))
}

func doFrontBuildingTest(t *testing.T, mock sqlmock.Sqlmock, expectOut string) {
	serverutil.InitMinifier()

	mock.ExpectPrepare(`SELECT\s*` +
		`boards.id, section_id, uri, dir, navbar_position, title, subtitle, description,\s*` +
		`max_file_size, max_threads, default_style, locked, created_at, anonymous_name, force_anonymous,\s*` +
		`autosage_after, no_images_after, max_message_length, min_message_length, allow_embeds, redirect_to_thread,\s*` +
		`require_file, enable_catalog\s*` +
		`FROM boards\s*` +
		`INNER JOIN \(\s*` +
		`SELECT id, hidden FROM sections\s*` +
		`\) s ON boards.section_id = s.id\s*` +
		`WHERE s\.hidden = FALSE\s*` +
		`ORDER BY navbar_position ASC, boards.id ASC`).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{
			"boards.id", "section_id", "uri", "dir", "navbar_position", "title", "subtitle", "description",
			"max_file_size", "max_threads", "default_style", "locked", "created_at", "anonymous_name", "force_anonymous",
			"autosage_after", "no_images_after", "max_message_length", "min_message_length", "allow_embeds", "redirect_to_thread",
			"require_file", "enable_catalog",
		}).AddRows([]driver.Value{
			1, 1, "test", "test", 1, "Testing board", "Board for testing", "Board for testing description",
			15000, 100, "pipes.css", false, time.Now(), "Anonymous", false,
			1500, 2000, 1500, 0, true, false, false, true,
		}).AddRows([]driver.Value{
			1, 1, "test2", "test2", 1, "Testing board 2", "Board for testing 2", "Board for testing description 2",
			15000, 100, "pipes.css", false, time.Now(), "Anonymous", false,
			1500, 2000, 1500, 0, true, false, false, true,
		}),
	)

	mock.ExpectPrepare(
		`SELECT id, name, abbreviation, position, hidden FROM sections WHERE hidden = FALSE ORDER BY position ASC, name ASC`,
	).ExpectQuery().
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "abbreviation", "position", "hidden"}).
			AddRows([]driver.Value{1, "Main", "main", 1, false}))

	mock.ExpectPrepare(`SELECT id, message_raw, dir, filename, op_id FROM v_front_page_posts_with_file ORDER BY id DESC LIMIT 15`).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{"posts.id", "posts.message_raw", "dir", "filename", "op.id"}).
			AddRows(
				[]driver.Value{1, "message_raw", "test", "filename", 1},
				[]driver.Value{2, "message_raw", "test", "filename", 1},
			))

	err := BuildFrontPage()
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	frontFile, err := os.Open(path.Join(config.GetSystemCriticalConfig().DocumentRoot, "index.html"))
	if !assert.NoError(t, err) {
		return
	}
	defer frontFile.Close()
	ba, err := io.ReadAll(frontFile)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, expectOut, string(ba))
	assert.NoError(t, frontFile.Close())
}

func TestBuildFrontPage(t *testing.T) {
	testRoot, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}

	for _, driver := range sql.Drivers() {
		if driver == "sqlmock" || t.Failed() {
			continue
		}
		t.Run(driver, func(t *testing.T) {
			outDir := t.TempDir()
			config.SetVersion("3.11.0")
			systemCriticalCfg := config.GetSystemCriticalConfig()
			systemCriticalCfg.DocumentRoot = path.Join(outDir, "html")
			systemCriticalCfg.TemplateDir = path.Join(testRoot, "templates")
			systemCriticalCfg.LogDir = path.Join(outDir, "logs")
			systemCriticalCfg.WebRoot = "/chan"
			systemCriticalCfg.TimeZone = 8
			config.SetSystemCriticalConfig(systemCriticalCfg)

			boardCfg := config.GetBoardConfig("")
			boardCfg.Styles = []config.Style{{Name: "test1", Filename: "test1.css"}}
			boardCfg.DefaultStyle = "test1.css"

			os.MkdirAll(systemCriticalCfg.DocumentRoot, config.DirFileMode)

			mock, err := gcsql.SetupMockDB(driver)
			if !assert.NoError(t, err) {
				return
			}
			siteCfg := config.GetSiteConfig()
			siteCfg.MinifyHTML = true
			config.SetSiteConfig(siteCfg)
			doFrontBuildingTest(t, mock, expectedMinifiedFront)
			siteCfg.MinifyHTML = false
			config.SetSiteConfig(siteCfg)
			doFrontBuildingTest(t, mock, expectedUnminifiedFront)
		})
	}
}
