package building

import (
	"database/sql/driver"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PuerkitoBio/goquery"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

func doCatalogTest(t *testing.T, minified bool) {
	baseDir := t.TempDir()
	config.InitTestConfig()
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	systemCriticalConfig := config.GetSystemCriticalConfig()
	systemCriticalConfig.TemplateDir = "templates"
	systemCriticalConfig.LogDir = path.Join(baseDir, "logs")
	systemCriticalConfig.DocumentRoot = path.Join(baseDir, "html")
	systemCriticalConfig.DBtype = "sqlite3"
	systemCriticalConfig.DBhost = path.Join(baseDir, "gochan.db")
	systemCriticalConfig.DBname = "gochan"
	systemCriticalConfig.DBusername = "gochan"
	systemCriticalConfig.DBpassword = "gochan"
	systemCriticalConfig.WebRoot = "/chan/"
	config.SetSystemCriticalConfig(systemCriticalConfig)

	boardCfg := config.GetBoardConfig("test")
	boardCfg.DefaultStyle = "teststyle.css"
	assert.NoError(t, config.SetBoardConfig("test", boardCfg))

	if !assert.NoError(t, os.Mkdir(systemCriticalConfig.LogDir, 0755)) {
		t.FailNow()
	}
	if !assert.NoError(t, os.MkdirAll(path.Join(systemCriticalConfig.DocumentRoot, "test"), 0755)) {
		t.FailNow()
	}

	if !assert.NoError(t, gcutil.InitLogs(systemCriticalConfig.LogDir, nil)) {
		t.FailNow()
	}
	defer gcutil.CloseLogs()

	siteConfig := config.GetSiteConfig()
	siteConfig.MinifyHTML = minified
	siteConfig.MinifyJS = minified
	config.SetSiteConfig(siteConfig)

	dbDriver := config.GetSQLConfig().DBtype

	mock := gcsql.SetupMockDB(t, dbDriver)
	defer gcsql.Close()

	mockSelectNonHiddenBoards(mock)
	mockSelectNonHiddenSections(mock)
	getBoardQuery := `SELECT\s+` +
		`boards.id, section_id, uri, dir, navbar_position, title, subtitle, description, created_at\s+` +
		`FROM boards INNER JOIN\s*\(\s*SELECT id, hidden FROM sections\s*\) s ON boards.section_id = s\.id WHERE boards\.id = `
	if dbDriver == "mysql" {
		getBoardQuery += `\?`
	} else {
		getBoardQuery += `\$1`
	}
	mock.ExpectPrepare(getBoardQuery).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "section_id", "uri", "dir", "navbar_position", "title", "subtitle", "description", "created_at",
		}).AddRows([]driver.Value{
			1, 1, "test", "test", 1, "Testing board", "Board for testing", "Board for testing description", time.Now(),
		}),
	)

	mock.ExpectPrepare(`SELECT ` +
		`id, thread_id, ip, name, tripcode, is_secure_tripcode, email, subject, created_on,\s+last_modified, parent_id, last_bump, ` +
		`message, message_raw, banned_message, board_id, dir, original_filename, filename,\s+checksum, filesize, tw, th, width, height, ` +
		`spoiler_file, locked, stickied, cyclic, spoiler_thread, flag, country, is_deleted\s+FROM v_building_posts`).ExpectQuery().WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "thread_id", "ip", "name", "tripcode", "is_secure_tripcode", "email", "subject", "created_on",
			"last_modified", "parent_id", "last_bump", "message", "message_raw", "banned_message", "board_id",
			"dir", "original_filename", "filename", "checksum", "filesize", "tw", "th", "width", "height",
			"spoiler_file", "locked", "stickied", "cyclic", "spoiler_thread", "flag", "country", "is_deleted",
		}).AddRows([]driver.Value{
			1, 1, "192.168.1.1", "Anonymous", "", false, "", "Normal thread", time.Now(),
			time.Now(), 1, time.Now(), "Lorem ipsum<br/>blah blah blah", "Lorem ipsum\nblah blah blah", "", 1,
			"test", "test.jpg", "test.jpg", "checksum", 12345, 150, 150, 1920, 1080,
			false, false, false, false, false, "US", "United States", false,
		}, []driver.Value{
			2, 2, "192.168.1.2", "Name", "!Trip", false, "email@example.com", "", time.Now(),
			time.Now(), 1, time.Now(), "Thread with name and trip<b>bold</b>", "Thread with name and trip[b]bold[/b]", "", 1,
			"test", "", "", "", 0, 0, 0, 0, 0,
			false, false, false, false, false, "CA", "Canada", false,
		}, []driver.Value{
			3, 3, "192.168.1.3", "", "!Trip", false, "email@example.com", "Status Icons Test (Cyclic, Locked, Stickied)", time.Now(),
			time.Now(), 1, time.Now(), "This thread is cyclic, locked, and stickied.", "This thread is cyclic, locked, and stickied.", "", 1,
			"test", "", "", "", 0, 0, 0, 0, 0,
			true, true, true, true, false, "GB", "United Kingdom", false,
		}),
	)
	mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM posts WHERE thread_id = \(\s*SELECT thread_id FROM posts WHERE id = \$1\) AND is_deleted = FALSE AND is_top_post = FALSE`).ExpectQuery().
		WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(4))
	mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM posts WHERE thread_id = \(\s*SELECT thread_id FROM posts WHERE id = \$1\) AND is_deleted = FALSE AND is_top_post = FALSE`).ExpectQuery().
		WithArgs(2).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
	mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM posts WHERE thread_id = \(\s*SELECT thread_id FROM posts WHERE id = \$1\) AND is_deleted = FALSE AND is_top_post = FALSE`).ExpectQuery().
		WithArgs(3).WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(3))
	serverutil.InitMinifier()

	if !assert.NoError(t, BuildCatalog(1)) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}

	// Verify that the catalog file has the expected thread data
	docRoot := config.GetSystemCriticalConfig().DocumentRoot
	fd, err := os.Open(path.Join(docRoot, "test/catalog.html"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer fd.Close()

	doc, err := goquery.NewDocumentFromReader(fd)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	assert.Equal(t, "/test/ - Testing board", doc.Find("header h1").Text())
	assert.Equal(t, "Catalog", doc.Find("header div#board-subtitle").Text())
	themeVal, exists := doc.Find("link#theme").Attr("href")
	if assert.True(t, exists) {
		assert.Equal(t, "/chan/css/teststyle.css", themeVal)
	}

	threads := doc.Find(".catalog-thread").Nodes
	if assert.Len(t, threads, 3) {
		firstThread := goquery.NewDocumentFromNode(threads[0])
		assert.Equal(t, "Normal thread:", firstThread.Find(".subject").Text())
		html, err := firstThread.Find(".post-message").Html()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.Equal(t, html, "Lorem ipsum<br/>blah blah blah")
		assert.Equal(t, "R: 4", strings.TrimSpace(firstThread.Find(".replies").Text()))

		secondThread := goquery.NewDocumentFromNode(threads[1])
		assert.Equal(t, "R: 1", strings.TrimSpace(secondThread.Find(".replies").Text()))
		assert.Equal(t, "Thread with name and tripbold", secondThread.Find(".post-message").Text())

		thirdThread := goquery.NewDocumentFromNode(threads[2])
		statusIcons := thirdThread.Find(".status-icons img")
		assert.Equal(t, "R: 3", strings.TrimSpace(thirdThread.Find(".replies").Text()))
		if assert.Len(t, statusIcons.Nodes, 3) {
			assert.Equal(t, "/chan/static/lock.png", statusIcons.Eq(0).AttrOr("src", ""))
			assert.Equal(t, "/chan/static/sticky.png", statusIcons.Eq(1).AttrOr("src", ""))
			assert.Equal(t, "/chan/static/cyclic.png", statusIcons.Eq(2).AttrOr("src", ""))
		}
	}
}

func TestBuildCatalog(t *testing.T) {
	t.Run("minified", func(t *testing.T) {
		doCatalogTest(t, true)
	})
	t.Run("not minified", func(t *testing.T) {
		doCatalogTest(t, false)
	})
}
