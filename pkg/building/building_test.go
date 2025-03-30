package building

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PuerkitoBio/goquery"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

var (
	pageHeaderTestCases = []pageHeaderTestCase{
		{
			desc:            "Front page with includes",
			pageTitle:       "Gochan",
			includeJS:       []config.IncludeScript{{Location: "test.js", Defer: true}, {Location: "test2.js", Defer: false}},
			includeCSS:      []string{"test.css"},
			expectTitleText: "Gochan",
			misc: map[string]any{
				"documentTitle": "Gochan",
			},
		},
		{
			desc:            "Front page without includes",
			pageTitle:       "Gochan",
			board:           "",
			expectTitleText: "Gochan",
			misc: map[string]any{
				"documentTitle": "Gochan",
			},
		},
		{
			desc: "Regular ban page",
			misc: map[string]any{
				"ban": gcsql.IPBan{},
			},
			expectTitleText: "YOU ARE BANNED :(",
		},
		{
			desc: "Unappealable permaban",
			misc: map[string]any{
				"ban": gcsql.IPBan{
					IPBanBase: gcsql.IPBanBase{
						CanAppeal: false,
						Permanent: true,
						IsActive:  true,
					},
				},
			},
			expectTitleText: "YOU'RE PERMABANNED,\u00a0IDIOT!",
		},
		{
			desc:      "Board page",
			pageTitle: "Gochan",
			board:     "test",
			misc: map[string]any{
				"documentTitle": "/test/ - Testing board",
			},
			expectTitleText: "/test/ - Testing board",
		},
	}
)

func TestBuildJS(t *testing.T) {
	testRoot, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}

	outDir := t.TempDir()
	config.SetVersion("4.0.2")
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

func mockSetupBoards(mock sqlmock.Sqlmock) {
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
}

func doFrontBuildingTest(t *testing.T, mock sqlmock.Sqlmock) {
	serverutil.InitMinifier()

	mockSetupBoards(mock)

	mock.ExpectPrepare(`SELECT id, message_raw, dir, filename, op_id FROM v_front_page_posts_with_file ORDER BY id DESC LIMIT 15`).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{"posts.id", "posts.message_raw", "dir", "filename", "op.id"}).
			AddRows(
				[]driver.Value{1, "message_raw 1", "test", "filename.png", 1},
				[]driver.Value{2, "message_raw 2", "test", "", 1},
				[]driver.Value{3, "message_raw 3", "test", "deleted", 1},
			))

	err := BuildFrontPage()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	frontFile, err := os.Open(path.Join(config.GetSystemCriticalConfig().DocumentRoot, "index.html"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer frontFile.Close()

	doc, err := goquery.NewDocumentFromReader(frontFile)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	boardsDiv := doc.Find("div#frontpage div.section-block:nth-of-type(2)")
	if !assert.Equal(t, 1, boardsDiv.Length()) {
		t.FailNow()
	}
	assert.Equal(t, "Boards", boardsDiv.Find("div.section-title-block").Text())
	sectionUl := boardsDiv.Find("div.section-body ul")
	if !assert.Equal(t, 1, sectionUl.Length()) {
		t.FailNow()
	}
	li := sectionUl.Find("li")
	assert.Equal(t, 3, li.Length())
	assert.Equal(t, config.GetSiteConfig().SiteName, doc.Find("title").Text())
	assert.Equal(t, config.GetSiteConfig().SiteName, doc.Find("div#top-pane h1").Text())
	assert.Equal(t, config.GetSiteConfig().SiteSlogan, doc.Find("div#top-pane span#site-slogan").Text())
	assert.Equal(t, "Main", li.Eq(0).Text())
	assert.Equal(t, "/test/ — Testing board", li.Eq(1).Text())
	assert.Equal(t, "/test2/ — Testing board 2", li.Eq(2).Text())
	assert.Equal(t, "/chan/test/", li.Eq(1).Find("a").AttrOr("href", ""))
	assert.Equal(t, "/chan/test2/", li.Eq(2).Find("a").AttrOr("href", ""))

	recentPostsContainer := doc.Find("div#frontpage div.section-block:nth-of-type(3)")
	if !assert.Equal(t, 1, recentPostsContainer.Length()) {
		t.FailNow()
	}
	assert.Equal(t, "Recent Posts", recentPostsContainer.Find("div.section-title-block").Text())
	recentPosts := recentPostsContainer.Find("div.section-body div.recent-post")
	if !assert.Equal(t, 3, recentPosts.Length()) {
		t.FailNow()
	}

	assert.Regexp(t, `/test/\s*message_raw 1`, recentPosts.Eq(0).Text())
	assert.Equal(t, 1, recentPosts.Eq(0).Find(`img[src="/chan/test/thumb/filenamet.png"]`).Length())
	assert.Equal(t, 1, recentPosts.Eq(1).Find("div.file-deleted-box").Length())
	assert.Equal(t, 1, recentPosts.Eq(2).Find("div.file-deleted-box").Length())

	assert.NoError(t, frontFile.Close())
}

func TestBuildFrontPage(t *testing.T) {
	testRoot, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	for _, driver := range sql.Drivers() {
		if driver == "sqlmock" {
			continue
		}
		t.Run(driver, func(t *testing.T) {
			outDir := t.TempDir()
			config.SetVersion("4.0.2")
			systemCriticalCfg := config.GetSystemCriticalConfig()
			systemCriticalCfg.DocumentRoot = path.Join(outDir, "html")
			systemCriticalCfg.TemplateDir = path.Join(testRoot, "templates")
			systemCriticalCfg.LogDir = path.Join(outDir, "logs")
			systemCriticalCfg.WebRoot = "/chan"
			systemCriticalCfg.TimeZone = 8
			config.SetSystemCriticalConfig(systemCriticalCfg)

			siteConfig := config.GetSiteConfig()
			siteConfig.SiteName = "Gochan"
			siteConfig.SiteSlogan = "Gochan description"
			config.SetSiteConfig(siteConfig)

			boardCfg := config.GetBoardConfig("")
			boardCfg.Styles = []config.Style{{Name: "test1", Filename: "test1.css"}}
			boardCfg.DefaultStyle = "test1.css"

			os.MkdirAll(systemCriticalCfg.DocumentRoot, config.DirFileMode)

			mock, err := gcsql.SetupMockDB(driver)
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			siteCfg := config.GetSiteConfig()
			siteCfg.MinifyHTML = true
			config.SetSiteConfig(siteCfg)
			doFrontBuildingTest(t, mock)
			siteCfg.MinifyHTML = false
			config.SetSiteConfig(siteCfg)
			doFrontBuildingTest(t, mock)
		})
	}
}

type pageHeaderTestCase struct {
	desc            string
	pageTitle       string
	board           string
	misc            map[string]any
	includeJS       []config.IncludeScript
	includeCSS      []string
	expectError     bool
	expectTitleText string
}

func (p *pageHeaderTestCase) runTest(t *testing.T, driver string) {
	mock, err := gcsql.SetupMockDB(driver)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	boardCfg := config.GetBoardConfig(p.board)
	boardCfg.IncludeGlobalStyles = p.includeCSS
	boardCfg.IncludeScripts = p.includeJS
	config.SetBoardConfig(p.board, boardCfg)

	mockSetupBoards(mock)

	err = gctemplates.InitTemplates()
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	var buf bytes.Buffer
	err = BuildPageHeader(&buf, p.pageTitle, p.board, p.misc)
	if p.expectError {
		assert.Error(t, err)
	} else {
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}

	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}

	doc, err := goquery.NewDocumentFromReader(&buf)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, len(p.includeJS)+2, doc.Find("script").Length())

	for i, js := range p.includeJS {
		script := doc.Find("script").Eq(i + 2)
		assert.Equal(t, js.Location, script.AttrOr("src", ""))
		_, hasDefer := script.Attr("defer")
		assert.Equal(t, js.Defer, hasDefer)
	}

	assert.Equal(t, len(p.includeCSS)+2, doc.Find(`link[rel="stylesheet"]`).Length())
	assert.Equal(t, p.expectTitleText, doc.Find("title").Text())
	if _, ok := p.misc["ban"]; !ok {
		topbarItems := doc.Find("a.topbar-item")
		assert.Equal(t, 3, topbarItems.Length())
		assert.Equal(t, "home", topbarItems.Eq(0).Text())
		assert.Equal(t, "/test/", topbarItems.Eq(1).Text())
		assert.Equal(t, "/test2/", topbarItems.Eq(2).Text())
	}
}

func TestBuildPageHeader(t *testing.T) {
	config.SetVersion("4.0.2")
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}

	systemCriticalConfig := config.GetSystemCriticalConfig()
	systemCriticalConfig.TemplateDir = "templates"
	config.SetSystemCriticalConfig(systemCriticalConfig)

	for _, tc := range pageHeaderTestCases {
		for _, driver := range sql.Drivers() {
			if driver == "sqlmock" {
				continue
			}
			t.Run(tc.desc+" - "+driver, func(t *testing.T) {
				tc.runTest(t, driver)
			})
		}
	}
}

func TestBuildPageFooter(t *testing.T) {
	config.SetVersion("4.0.2")
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	systemCriticalConfig := config.GetSystemCriticalConfig()
	systemCriticalConfig.TemplateDir = "templates"
	config.SetSystemCriticalConfig(systemCriticalConfig)

	mock, err := gcsql.SetupMockDB("sqlite3")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	var buf bytes.Buffer
	if !assert.NoError(t, BuildPageFooter(&buf)) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	doc, err := goquery.NewDocumentFromReader(&buf)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Regexp(t, `Powered by Gochan \d+\.\d+\.\d+`, doc.Find("footer").Text())
}
