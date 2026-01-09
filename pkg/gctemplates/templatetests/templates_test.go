package templatetests_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/gochan-org/gochan/pkg/building"
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
		`subtitle,\s*description,\s*created_at\s+FROM boards\s+INNER JOIN\s*\(\s*SELECT id,\s*hidden\s+FROM sections\s*\)\s+s\s+ON ` +
		`boards\.section_id = s\.id\s+WHERE s\.hidden = FALSE ORDER BY navbar_position ASC, boards\.id ASC`

	selectSectionsQueryExpectation = `SELECT\s+id,\s*name,\s*abbreviation,\s*position,\s*hidden\s+FROM\s+sections\s+WHERE\s+hidden\s*=\s*FALSE\s+` +
		`ORDER BY\s+position ASC,\s*name ASC`
)

func initTemplatesMock(t *testing.T, mock sqlmock.Sqlmock, which ...string) {
	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	rows := sqlmock.NewRows([]string{"boards.id", "section_id", "uri", "dir", "navbar_position", "title",
		"subtitle", "description", "created_at"}).
		AddRow(1, 1, "test", "test", 1, "Testing board", "Board for testing", "Board for testing", time.Now()).
		AddRow(2, 1, "test2", "test2", 2, "Testing board #2", "Board for testing", "Board for testing", time.Now())

	mock.ExpectPrepare(selectBoardsQueryExpectation).
		ExpectQuery().WithoutArgs().WillReturnRows(rows)

	rows = sqlmock.NewRows([]string{"id", "name", "abbreviation", "position", "hidden"}).
		AddRow(1, "Main", "main", 1, false)

	mock.ExpectPrepare(selectSectionsQueryExpectation).
		ExpectQuery().WithoutArgs().WillReturnRows(rows)

	config.SetTestTemplateDir("templates")

	if len(which) != 0 && which[0] == "" {
		which = nil
	}

	if !assert.NoError(t, gctemplates.InitTemplates(which...)) {
		t.FailNow()
	}

	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}

}

func runTemplateTestCases(t *testing.T, templateName string, testCases []templateTestCase) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	config.InitTestConfig()
	config.SetTestDBConfig("mysql", "localhost", "gochan", "gochan", "gochan", "")
	if !assert.NoError(t, gcsql.SetTestingDB("mysql", "gochan", "", db)) {
		return
	}

	initTemplatesMock(t, mock, templateName)

	serverutil.InitMinifier()
	for _, tC := range testCases {
		if tC.getDefaultStyle {
			mock.ExpectPrepare(`SELECT default_style FROM boards WHERE dir = \?`).ExpectQuery().
				WithArgs("test").WillReturnRows(sqlmock.NewRows([]string{"default_style"}).AddRow("pipes.css"))
		}
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
	runTemplateTestCases(t, "", nil)
}

func TestBaseFooter(t *testing.T) {
	runTemplateTestCases(t, gctemplates.PageFooter, baseFooterCases)
}

func TestBaseHeader(t *testing.T) {
	runTemplateTestCases(t, gctemplates.PageHeader, baseHeaderCases)
}
