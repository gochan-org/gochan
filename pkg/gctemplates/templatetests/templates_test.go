package templatetests_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	_ "github.com/gochan-org/gochan/pkg/gcsql/initsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	_ "github.com/gochan-org/gochan/pkg/posting/uploads/inituploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/stretchr/testify/assert"
)

const (
	selectBoardsQueryExpectation = `SELECT\s+boards\.id, section_id,\s*uri,\s*dir,\s*navbar_position,\s*title,\s*` +
		`subtitle,\s*description,\s*max_file_size,\s*max_threads,\s*default_style,\s*locked,\s*created_at,\s*` +
		`anonymous_name,\s*force_anonymous,\s*autosage_after,\s*no_images_after,\s*max_message_length,\s*` +
		`min_message_length,\s*allow_embeds,\s*redirect_to_thread,\s*require_file,\s*enable_catalog\s+` +
		`FROM boards\s+INNER JOIN\s*\(\s*SELECT id,\s*hidden\s+FROM sections\s*\)\s+s\s+ON ` +
		`boards\.section_id = s\.id\s+WHERE s\.hidden = FALSE ORDER BY navbar_position ASC, boards\.id ASC`

	selectSectionsQueryExpectation = `SELECT\s+id,\s*name,\s*abbreviation,\s*position,\s*hidden\s+FROM\s+sections\s+WHERE\s+hidden\s*=\s*FALSE\s+` +
		`ORDER BY\s+position ASC,\s*name ASC`
)

func initTemplatesMock(t *testing.T, mock sqlmock.Sqlmock, which ...string) bool {
	t.Helper()
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		return false
	}

	rows := sqlmock.NewRows([]string{"boards.id", "section_id", "uri", "dir", "navbar_position", "title",
		"subtitle", "description", "max_file_size", "max_threads", "default_style", "locked", "created_at",
		"anonymous_name", "force_anonymous", "autosage_after", "no_images_after", "max_message_length",
		"min_message_length", "allow_embeds", "redirect_to_thread", "require_file", "enable_catalog"}).
		AddRow(1, 1, "test", "test", 1, "Testing board", "Board for testing", "Board for testing", 500, 500,
			"pipes.css", false, time.Now(), "Anonymous", false, 200, 500, 1500, 0, false, false, false, true).
		AddRow(2, 1, "test2", "test2", 2, "Testing board #2", "Board for testing", "Board for testing", 500, 500,
			"pipes.css", false, time.Now(), "Anonymous", false, 200, 500, 1500, 0, false, false, false, true)

	mock.ExpectPrepare(selectBoardsQueryExpectation).
		ExpectQuery().WithoutArgs().WillReturnRows(rows)

	rows = sqlmock.NewRows([]string{"id", "name", "abbreviation", "position", "hidden"}).
		AddRow(1, "Main", "main", 1, false)

	mock.ExpectPrepare(selectSectionsQueryExpectation).
		ExpectQuery().WithoutArgs().WillReturnRows(rows)

	config.SetTestTemplateDir("templates")

	if !assert.NoError(t, gctemplates.InitTemplates(which...)) {
		return false
	}
	return assert.NoError(t, mock.ExpectationsWereMet())
}

func runTemplateTestCases(t *testing.T, templateName string, testCases []templateTestCase) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	config.SetVersion("4.0.0")
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "gochan", "")
	if !assert.NoError(t, gcsql.SetTestingDB("mysql", "gochan", "", db)) {
		return
	}

	if !initTemplatesMock(t, mock) {
		return
	}

	serverutil.InitMinifier()
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			tC.Run(t, templateName)
		})
	}
}

func TestBanPageTemplate(t *testing.T) {
	runTemplateTestCases(t, gctemplates.BanPage, banPageCases)
}

func TestBoardPageTemplate(t *testing.T) {
	runTemplateTestCases(t, gctemplates.BoardPage, boardPageTestCases)
}

func TestJsConstsTemplate(t *testing.T) {
	runTemplateTestCases(t, gctemplates.JsConsts, jsConstsCases)
}

func TestTemplateBase(t *testing.T) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "gochan", "")
	if !assert.NoError(t, gcsql.SetTestingDB("mysql", "gochan", "", db)) {
		return
	}

	initTemplatesMock(t, mock)
}

func TestBaseFooter(t *testing.T) {
	runTemplateTestCases(t, gctemplates.PageFooter, baseFooterCases)
}

func TestBaseHeader(t *testing.T) {
	runTemplateTestCases(t, gctemplates.PageHeader, baseHeaderCases)
}
