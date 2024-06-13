package building

import (
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"path"
	"testing"

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
	config.SetVersion("3.10.2")
	systemCriticalCfg := config.GetSystemCriticalConfig()
	systemCriticalCfg.DocumentRoot = path.Join(outDir, "html")
	systemCriticalCfg.TemplateDir = path.Join(testRoot, "templates")
	systemCriticalCfg.LogDir = path.Join(outDir, "logs")
	systemCriticalCfg.WebRoot = "/chan"
	systemCriticalCfg.TimeZone = 8
	config.SetSystemCriticalConfig(&systemCriticalCfg)

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
	mock.ExpectPrepare(`SELECT\s*posts.id,\s*posts.message_raw,\s*` +
		`\(SELECT dir FROM boards WHERE id = t.board_id\),\s*` +
		`COALESCE\(f.filename, ''\), op.id\s*` +
		`FROM posts\s*` +
		`LEFT JOIN \(SELECT id, board_id FROM threads\) t ON t.id = posts.thread_id\s+` +
		`LEFT JOIN \(SELECT post_id, filename FROM files\) f on f.post_id = posts.id\s+` +
		`INNER JOIN \(SELECT id, thread_id FROM posts WHERE is_top_post\) op ON op.thread_id = posts.thread_id\s+` +
		`WHERE posts.is_deleted = FALSE\s+` +
		`AND f.filename IS NOT NULL AND f.filename != '' AND f.filename != 'deleted'\s+` +
		`ORDER BY posts.id DESC LIMIT \d+`).ExpectQuery().WillReturnRows(
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
			config.SetVersion("3.10.2")
			systemCriticalCfg := config.GetSystemCriticalConfig()
			systemCriticalCfg.DocumentRoot = path.Join(outDir, "html")
			systemCriticalCfg.TemplateDir = path.Join(testRoot, "templates")
			systemCriticalCfg.LogDir = path.Join(outDir, "logs")
			systemCriticalCfg.WebRoot = "/chan"
			systemCriticalCfg.TimeZone = 8
			config.SetSystemCriticalConfig(&systemCriticalCfg)

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
