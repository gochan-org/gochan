package manage

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/stretchr/testify/assert"
)

var (
	boardRequestTypeTestCases = []boardsRequestTypeTestCase{
		{
			desc:     "view boards",
			form:     url.Values{},
			method:   http.MethodGet,
			path:     "/manage/boards",
			expected: boardRequestTypeViewBoards,
		},
		{
			desc:     "view single board",
			form:     url.Values{},
			method:   http.MethodGet,
			path:     "/manage/boards/test",
			expected: boardRequestTypeViewSingleBoard,
		},
		{
			desc: "cancel changes",
			form: url.Values{
				"dir":            {"test"},
				"title":          {"Test Board"},
				"subtitle":       {"Subtitle"},
				"description":    {"Description"},
				"section":        {"1"},
				"navbarposition": {"0"},
				"maxfilesize":    {"150000"},
				"maxthreads":     {"-1"},
				"defaultstyle":   {"pipes.css"},
				// "locked": {},
				"anonname":         {"Anonymous"},
				"autosageafter":    {"200"},
				"nouploadsafter":   {"-1"},
				"maxmessagelength": {"1500"},
				"minmessagelength": {"0"},
				"embedsallowed":    {"on"},
				// "redirecttothread": {},
				// "requirefile": {},
				"enablecatalog": {"on"},
				"docancel":      {"Cancel"},
			},
			method:   http.MethodPost,
			path:     "/manage/boards",
			expected: boardRequestTypeCancel,
		},
		{
			desc: "create board",
			form: url.Values{
				"dir":            {"test"},
				"title":          {"Test Board"},
				"subtitle":       {"Subtitle"},
				"description":    {"Description"},
				"section":        {"1"},
				"navbarposition": {"0"},
				"maxfilesize":    {"150000"},
				"maxthreads":     {"-1"},
				"defaultstyle":   {"pipes.css"},
				// "locked": {},
				"anonname":         {"Anonymous"},
				"autosageafter":    {"200"},
				"nouploadsafter":   {"-1"},
				"maxmessagelength": {"1500"},
				"minmessagelength": {"0"},
				"embedsallowed":    {"on"},
				// "redirecttothread": {},
				// "requirefile": {},
				"enablecatalog": {"on"},
				"docreate":      {"Create Board"},
			},
			method:   http.MethodPost,
			path:     "/manage/boards",
			expected: boardRequestTypeCreate,
		},
		{
			desc:   "save changes to board",
			method: http.MethodPost,
			form: url.Values{
				"dir":            {"test"},
				"title":          {"Test Board"},
				"subtitle":       {"Subtitle"},
				"description":    {"Description"},
				"section":        {"1"},
				"navbarposition": {"0"},
				"maxfilesize":    {"150000"},
				"maxthreads":     {"-1"},
				"defaultstyle":   {"pipes.css"},
				// "locked": {},
				"anonname":         {"Anonymous"},
				"autosageafter":    {"200"},
				"nouploadsafter":   {"2000"},
				"maxmessagelength": {"1500"},
				"minmessagelength": {"0"},
				"embedsallowed":    {"on"},
				// "redirecttothread": {},
				// "requirefile": {},
				"enablecatalog": {"on"},
				"domodify":      {"Save Changes"},
			},
			path:     "/manage/boards",
			expected: boardRequestTypeModify,
		},
		{
			desc: "delete board",
			form: url.Values{
				"dir":            {"test"},
				"title":          {"Test Board"},
				"subtitle":       {"Subtitle"},
				"description":    {"Description"},
				"section":        {"1"},
				"navbarposition": {"0"},
				"maxfilesize":    {"150000"},
				"maxthreads":     {"-1"},
				"defaultstyle":   {"pipes.css"},
				// "locked": {},
				"anonname":         {"Anonymous"},
				"autosageafter":    {"200"},
				"nouploadsafter":   {"-1"},
				"maxmessagelength": {"1500"},
				"minmessagelength": {"0"},
				"embedsallowed":    {"on"},
				// "redirecttothread": {},
				// "requirefile": {},
				"enablecatalog": {"on"},
				"dodelete":      {"Delete Board"},
			},
			path:     "/manage/boards",
			method:   http.MethodPost,
			expected: boardRequestTypeDelete,
		},
	}
)

type boardsRequestTypeTestCase struct {
	desc     string
	form     url.Values
	method   string
	path     string
	expected boardRequestType
}

func TestBoardsRequestType(t *testing.T) {
	setupManageTestSuite(t)
	for _, tc := range boardRequestTypeTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			u := "http://localhost/"
			if tc.method == http.MethodGet {
				u += "?" + tc.form.Encode()
			}
			req, err := http.NewRequest(tc.method, u, strings.NewReader(tc.form.Encode()))
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			req.URL.Path = tc.path
			result := boardsRequestType(req)
			assert.Equal(t, tc.expected, result, "expected %s, got %s", tc.expected.String(), result.String())

			switch tc.expected {
			case boardRequestTypeCreate, boardRequestTypeDelete, boardRequestTypeModify:
				var form createOrModifyBoardForm
				err = forms.FillStructFromForm(req, &form)
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				err = form.validate(gcutil.LogWarning())
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, result, form.requestType())
				var board gcsql.Board
				form.fillBoard(&board)
				assert.Equal(t, form.Dir, board.Dir)
				assert.Equal(t, form.Title, board.Title)
				assert.Equal(t, form.Subtitle, board.Subtitle)
				assert.Equal(t, form.Description, board.Description)
				assert.Equal(t, form.Section, board.SectionID)
				assert.Equal(t, form.NavBarPosition, board.NavbarPosition)
				// assert.Equal(t, form.MaxFileSize, board.MaxFilesize)
				// assert.Equal(t, form.MaxThreads, board.MaxThreads)
				// assert.Equal(t, form.DefaultStyle, board.DefaultStyle)
				// assert.Equal(t, form.Locked, board.Locked)
				// assert.Equal(t, form.AnonName, board.AnonymousName)
				// assert.Equal(t, form.AutosageAfter, board.AutosageAfter)
				// assert.Equal(t, form.NoUploadsAfter, board.NoImagesAfter)
				// assert.Equal(t, form.MaxMessageLength, board.MaxMessageLength)
				// assert.Equal(t, form.MinMessageLength, board.MinMessageLength)
				// assert.Equal(t, form.EmbedsAllowed, board.AllowEmbeds)
				// assert.Equal(t, form.RedirectToThread, board.RedirectToThread)
				// assert.Equal(t, form.RequireFile, board.RequireFile)
				// assert.Equal(t, form.EnableCatalog, board.EnableCatalog)
			default:
				// No form to test
			}
		})
	}
}
