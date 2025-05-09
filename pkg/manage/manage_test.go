package manage

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"maps"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/PuerkitoBio/goquery"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	loginQueryRE       = `SELECT\s*id,\s*username,\s*password_checksum,\s*global_rank,\s*added_on,\s*last_login,\s*is_active\s*FROM staff WHERE username = \? AND is_active = TRUE`
	insertSessionRE    = `INSERT INTO sessions \(staff_id,data,expires\) VALUES\(\?,\?,\?\)`
	updateStaffLoginRE = `UPDATE staff SET last_login = CURRENT_TIMESTAMP WHERE id = \?`
)

var (
	genericStaffList = []gcsql.Staff{
		{Username: "admin", Rank: 3},
		{Username: "mod", Rank: 2},
		{Username: "janitor", Rank: 1},
	}

	loginTestCases = []manageCallbackTestCase{
		{
			desc:         "GET login",
			path:         "/manage/login",
			method:       "GET",
			expectStatus: http.StatusOK,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				if !assert.NotNil(t, output) {
					t.FailNow()
				}
				doc, err := goquery.NewDocumentFromReader(strings.NewReader(output.(string)))
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				assert.Equal(t, 1, doc.Find("input[name=username]").Length())
				assert.Equal(t, 1, doc.Find("input[name=password]").Length())
				assert.Equal(t, 1, doc.Find("input[value=Login]").Length())
			},
		}, {
			desc:   "POST login",
			method: "POST",
			path:   "/manage/login",
			header: http.Header{
				"Referer": []string{"http://localhost/manage/login"},
			},
			form: url.Values{
				"username": {"admin"},
				"password": {"password"},
			},
			expectStatus: http.StatusFound,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				expectedSum := gcutil.BcryptSum("password")
				mock.ExpectPrepare(loginQueryRE).ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", expectedSum, 1, time.Now(), time.Now(), true),
				)
				mock.ExpectBegin()
				mock.ExpectPrepare(insertSessionRE).ExpectExec().WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectPrepare(updateStaffLoginRE).ExpectExec().WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				assert.Nil(t, output) // redirect, output is nil
			},
		}, {
			desc:   "POST login with invalid credentials",
			method: "POST",
			path:   "/manage/login",
			header: http.Header{
				"Referer": []string{"http://localhost/manage/login"},
			},
			form: url.Values{
				"username": {"admin"},
				"password": {"wrongpassword"},
			},
			expectStatus: http.StatusUnauthorized,
			expectError:  true,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				notExpectedSum := gcutil.BcryptSum("password")
				mock.ExpectPrepare(loginQueryRE).ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", notExpectedSum, 1, time.Now(), time.Now(), true),
				)
			},
		},
	}
	staffTestCases = []manageCallbackTestCase{
		{
			desc:         "View staff list as admin",
			method:       "GET",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Rank: 3, Username: "admin"},
			expectStatus: http.StatusOK,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, newUserForm)
			},
		},
		{
			desc:         "View staff list as mod",
			method:       "GET",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusOK,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "mod", Rank: 2}, output, noForm)
			},
		},
		{
			desc:         "View change rank form as admin",
			method:       "GET",
			path:         "/manage/staff?changerank=admin",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusOK,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id, username, password_checksum, global_rank, added_on, last_login, is_active FROM staff WHERE username = \? AND is_active = TRUE`).
					ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", gcutil.BcryptSum("password"), 3, time.Now(), time.Now(), true),
				)
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, changeRankForm)
			},
		},
		{
			desc:         "View change password form as admin",
			method:       "GET",
			path:         "/manage/staff?changepass=admin",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusOK,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id, username, password_checksum, global_rank, added_on, last_login, is_active FROM staff WHERE username = \? AND is_active = TRUE`).
					ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", gcutil.BcryptSum("password"), 3, time.Now(), time.Now(), true),
				)
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, changePasswordForm)
			},
		},
		{
			desc:         "View change password form as mod for self",
			method:       "GET",
			path:         "/manage/staff?changepass=mod",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusOK,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id, username, password_checksum, global_rank, added_on, last_login, is_active FROM staff WHERE username = \? AND is_active = TRUE`).
					ExpectQuery().WithArgs("mod").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(2, "mod", gcutil.BcryptSum("password"), 2, time.Now(), time.Now(), true),
				)
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "mod", Rank: 2}, output, changePasswordForm)
			},
		},
		{
			desc:         "View change password form as mod for another account",
			method:       "GET",
			path:         "/manage/staff?changepass=janitor",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusForbidden,
			expectError:  true,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error) {
				assert.ErrorIs(t, err, ErrInsufficientPermission)
				assert.Empty(t, output)
			},
		},
		{
			desc:         "View change change rank form as mod",
			method:       "GET",
			path:         "/manage/staff?changerank=mod",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusForbidden,
			expectError:  true,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error) {
				assert.ErrorIs(t, err, ErrInsufficientPermission)
				assert.Empty(t, output)
			},
		},
		{
			desc:         "Create new user as admin",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusOK,
			form: url.Values{
				"do":              {"add"},
				"username":        {"newuser"},
				"password":        {"newpassword"},
				"passwordconfirm": {"newpassword"},
				"rank":            {"1"},
			},
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM staff WHERE username = \?`).ExpectQuery().WithArgs("newuser").
					WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(0))
				mock.ExpectPrepare(`INSERT INTO staff \(username, password_checksum, global_rank\) VALUES\(\?,\?,\?\)`).ExpectExec().
					WithArgs("newuser", sqlmock.AnyArg(), 1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				getStaffMockHelper(t, mock,
					gcsql.Staff{Username: "admin", Rank: 3},
					gcsql.Staff{Username: "mod", Rank: 2},
					gcsql.Staff{Username: "janitor", Rank: 1},
					gcsql.Staff{Username: "newuser", Rank: 1})
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				expectedStaff := append(genericStaffList, gcsql.Staff{Username: "newuser", Rank: 1})
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, newUserForm, expectedStaff...)
			},
		},
		{
			desc:         "Try to create existing user as admin",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusBadRequest,
			form: url.Values{
				"do":              {"add"},
				"username":        {"mod"},
				"password":        {"newpassword"},
				"passwordconfirm": {"newpassword"},
				"rank":            {"1"},
			},
			expectError: true,
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT COUNT\(\*\) FROM staff WHERE username = \?`).ExpectQuery().WithArgs("mod").
					WillReturnRows(sqlmock.NewRows([]string{"COUNT(*)"}).AddRow(1))
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error) {
				assert.ErrorIs(t, err, gcsql.ErrStaffAlreadyExists)
				assert.Empty(t, output)
			},
		},
		{
			desc:         "Change password as admin",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusOK,
			form: url.Values{
				"do":              {"changepass"},
				"username":        {"mod"},
				"password":        {"newpassword"},
				"passwordconfirm": {"newpassword"},
			},
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id FROM staff WHERE username = \?`).ExpectQuery().WithArgs("mod").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
				mock.ExpectPrepare(`UPDATE staff SET password_checksum = \? WHERE id = \?`).ExpectExec().
					WithArgs(sqlmock.AnyArg(), 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, newUserForm)
			},
		},
		{
			desc:         "Change own password as mod",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusOK,
			form: url.Values{
				"do":              {"changepass"},
				"username":        {"mod"},
				"password":        {"newpassword"},
				"passwordconfirm": {"newpassword"},
			},
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id FROM staff WHERE username = \?`).ExpectQuery().WithArgs("mod").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(2))
				mock.ExpectPrepare(`UPDATE staff SET password_checksum = \? WHERE id = \?`).ExpectExec().
					WithArgs(sqlmock.AnyArg(), 2).
					WillReturnResult(sqlmock.NewResult(1, 1))
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "mod", Rank: 2}, output, noForm)
			},
		},
		{
			desc:         "Try to change password as mod for another account",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusForbidden,
			form: url.Values{
				"do":              {"changepass"},
				"username":        {"janitor"},
				"password":        {"newpassword"},
				"passwordconfirm": {"newpassword"},
			},
			expectError: true,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error) {
				assert.ErrorIs(t, err, ErrInsufficientPermission)
				assert.Empty(t, output)
			},
		},
		{
			desc:         "Change rank as admin",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "admin", Rank: 3},
			expectStatus: http.StatusOK,
			form: url.Values{
				"do":       {"changerank"},
				"username": {"janitor"},
				"rank":     {"2"},
			},
			prepareMock: func(t *testing.T, mock sqlmock.Sqlmock) {
				mock.ExpectPrepare(`SELECT id FROM staff WHERE username = \?`).ExpectQuery().WithArgs("janitor").
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(3))
				mock.ExpectPrepare(`UPDATE staff SET global_rank = \? WHERE id = \?`).ExpectExec().
					WithArgs(2, 3).
					WillReturnResult(sqlmock.NewResult(1, 1))
				getStaffMockHelper(t, mock)
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, _ error) {
				validateStaffOutput(t, &gcsql.Staff{Username: "admin", Rank: 3}, output, newUserForm, genericStaffList...)
			},
		},
		{
			desc:         "Try to change rank as mod",
			method:       "POST",
			path:         "/manage/staff",
			staff:        &gcsql.Staff{Username: "mod", Rank: 2},
			expectStatus: http.StatusForbidden,
			form: url.Values{
				"do":       {"changerank"},
				"username": {"janitor"},
				"rank":     {"2"},
			},
			expectError: true,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error) {
				assert.ErrorIs(t, err, ErrInsufficientPermission)
				assert.Empty(t, output)
			},
		},
	}
)

func getStaffMockHelper(t *testing.T, mock sqlmock.Sqlmock, expectedStaff ...gcsql.Staff) {
	t.Helper()

	if len(expectedStaff) == 0 {
		expectedStaff = genericStaffList
	}

	rows := sqlmock.NewRows([]string{"id", "username", "global_rank", "added_on", "last_login", "is_active"})
	for _, staff := range expectedStaff {
		rows.AddRow(1, staff.Username, staff.Rank, time.Now(), time.Now(), true)
	}
	mock.ExpectPrepare(`SELECT\s*id,\s*username,\s*global_rank,\s*added_on,\s*last_login,\s*is_active\s*FROM staff WHERE is_active`).
		ExpectQuery().WillReturnRows(rows)
}

func validateStaffOutput(t *testing.T, staff *gcsql.Staff, output any, expectedFormMode formMode, expectedStaffList ...gcsql.Staff) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(output.(string)))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if len(expectedStaffList) == 0 {
		expectedStaffList = append(expectedStaffList, genericStaffList...)
	}

	staffRows := doc.Find("table.stafflist tr")
	assert.Equal(t, len(expectedStaffList)+1, staffRows.Length())
	staffRows.Each(func(i int, s *goquery.Selection) {
		if i == 0 {
			return
		}
		if i > len(expectedStaffList) {
			assert.Fail(t, "More staff rows than expected")
			return
		}
		expectedStaff := expectedStaffList[i-1]
		assert.Equal(t, expectedStaff.Username, s.Find("td").Eq(0).Text())
		assert.Equal(t, expectedStaff.RankTitle(), s.Find("td").Eq(1).Text())
		if staff.Rank == 3 && expectedStaff.Username == staff.Username {
			assert.Equal(t, "Change Password | Change Rank", s.Find("td").Eq(3).Text())
		} else if staff.Rank == 3 {
			assert.Equal(t, "Change Password | Change Rank | Delete", s.Find("td").Eq(3).Text())
		} else if staff.Rank < 3 && expectedStaff.Username == staff.Username {
			assert.Equal(t, "Change Password", s.Find("td").Eq(3).Text())
		} else {
			assert.Equal(t, "", s.Find("td").Eq(3).Text())
		}
	})

	hidden := doc.Find("input[type=hidden]")
	switch expectedFormMode {
	case newUserForm:
		assert.Equal(t, "Add New User", doc.Find("h2").Text())
		assert.Equal(t, 1, doc.Find("input[name=username]").Length())
		assert.Equal(t, 1, doc.Find("input[name=password]").Length())
		assert.Equal(t, 1, doc.Find("input[name=passwordconfirm]").Length())
		assert.Equal(t, 1, doc.Find("select[name=rank]").Length())
		assert.Equal(t, 1, doc.Find("input[value='Create User']").Length())
		assert.Equal(t, 0, doc.Find("input[value=Cancel]").Length())
		assert.Equal(t, "add", hidden.Filter("[name=do]").AttrOr("value", ""))
	case changePasswordForm:
		assert.Equal(t, "Change Password", doc.Find("h2").Text())
		assert.Equal(t, 1, doc.Find("input[name=password]").Length())
		assert.Equal(t, 1, doc.Find("input[name=passwordconfirm]").Length())
		assert.Equal(t, 1, doc.Find("input[value='Update User']").Length())
		assert.Equal(t, 1, doc.Find("input[value=Cancel]").Length())
		assert.Equal(t, "changepass", hidden.Filter("[name=do]").AttrOr("value", ""))
		assert.Equal(t, staff.Username, hidden.Filter("[name=username]").AttrOr("value", ""))
	case changeRankForm:
		assert.Equal(t, "Change User Rank", doc.Find("h2").Text())
		assert.Equal(t, 1, doc.Find("input[value='Update User']").Length())
		assert.Equal(t, 1, doc.Find("input[value=Cancel]").Length())
		hidden := doc.Find("input[type=hidden]")
		assert.Equal(t, "changerank", hidden.Filter("[name=do]").AttrOr("value", ""))
		assert.Equal(t, staff.Username, hidden.Filter("[name=username]").AttrOr("value", ""))
	case noForm:
		assert.Equal(t, 0, doc.Find("h2").Length())
		form := doc.Find("form[action='/manage/staff']")
		assert.Equal(t, 0, form.Length())
	}
}

// manageCallbackTestCase is a generic test case struct for testing the callback functions for /manage/{action}
type manageCallbackTestCase struct {
	desc           string
	path           string
	staff          *gcsql.Staff
	method         string
	header         http.Header
	form           url.Values
	wantsJSON      bool
	expectError    bool
	expectStatus   int
	prepareMock    func(t *testing.T, mock sqlmock.Sqlmock)
	validateOutput func(t *testing.T, output any, writer *httptest.ResponseRecorder, err error)
}

func (tc *manageCallbackTestCase) runTest(t *testing.T, manageCallbackFunc CallbackFunction) {
	infoEv := gcutil.LogInfo()
	errEv := gcutil.LogError(nil)
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()
	if !assert.NoError(t, gcsql.SetTestingDB("mysql", "gochan", "", db)) {
		t.FailNow()
	}

	request, err := http.NewRequest(tc.method, "http://localhost"+tc.path, nil)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if tc.staff == nil {
		tc.staff = &gcsql.Staff{}
	}
	maps.Copy(request.Header, tc.header)
	if tc.method == "POST" {
		request.PostForm = tc.form
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		request.Form = tc.form
	}

	if tc.prepareMock != nil {
		tc.prepareMock(t, mock)
	}

	writer := httptest.NewRecorder()
	output, err := manageCallbackFunc(writer, request, tc.staff, tc.wantsJSON, infoEv, errEv)
	if tc.expectStatus == 0 {
		tc.expectStatus = http.StatusOK
	}
	assert.Equal(t, tc.expectStatus, writer.Code)
	if tc.expectError {
		assert.Error(t, err)
		if !assert.NoError(t, mock.ExpectationsWereMet()) {
			t.FailNow()
		}
		if tc.validateOutput != nil {
			tc.validateOutput(t, output, writer, err)
		}
	} else {
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		if !assert.NoError(t, mock.ExpectationsWereMet()) {
			t.FailNow()
		}
		if tc.validateOutput == nil {
			t.Fatal("validateOutput is nil")
		}
		tc.validateOutput(t, output, writer, err)
	}
}

func setupManageTestSuite(t *testing.T) {
	config.InitConfig()

	_, err := testutil.GoToGochanRoot(t)
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	systemCriticalConfig := config.GetSystemCriticalConfig()
	systemCriticalConfig.TemplateDir = "templates"
	systemCriticalConfig.SiteHost = "localhost"
	config.SetSystemCriticalConfig(systemCriticalConfig)

	gctemplates.InitTemplates()

}

func TestLoginCallback(t *testing.T) {
	setupManageTestSuite(t)

	for _, tc := range loginTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.runTest(t, loginCallback)
		})
	}
}

func TestStaffCallback(t *testing.T) {
	setupManageTestSuite(t)
	for _, tc := range staffTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			tc.runTest(t, staffCallback)
		})
	}
}
