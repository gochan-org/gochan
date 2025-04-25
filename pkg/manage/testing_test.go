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
	loginTestCases = []manageCallbackTestCase{
		{
			desc:         "GET login",
			path:         "/manage/login",
			method:       "GET",
			expectStatus: http.StatusOK,
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder) {
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
				mock.ExpectPrepare(loginQueryRE).ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", "$2a$10$EdXlrHd/vKQo9COSpxRdgOpjzEQ7As5mW4N5P4R4KrqaI8j3jO2PW", 1, time.Now(), time.Now(), true),
				)
				mock.ExpectBegin()
				mock.ExpectPrepare(insertSessionRE).ExpectExec().WithArgs(1, sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectPrepare(updateStaffLoginRE).ExpectExec().WithArgs(1).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			validateOutput: func(t *testing.T, output any, writer *httptest.ResponseRecorder) {
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
				mock.ExpectPrepare(loginQueryRE).ExpectQuery().WithArgs("admin").WillReturnRows(
					sqlmock.NewRows([]string{"id", "username", "password_checksum", "global_rank", "added_on", "last_login", "is_active"}).
						AddRow(1, "admin", "$2a$10$EdXlrHd/vKQo9COSpxRdgOpjzEQ7As5mW4N5P4R4KrqaI8j3jO2PW", 1, time.Now(), time.Now(), true),
				)
			},
		},
	}
)

// manageCallbackTestCase is a generic test case struct for testing the callback functions for /manage/{action}
type manageCallbackTestCase struct {
	desc string
	// writer         *httptest.ResponseRecorder
	path           string
	staff          *gcsql.Staff
	method         string
	header         http.Header
	form           url.Values
	wantsJSON      bool
	expectError    bool
	expectStatus   int
	prepareMock    func(t *testing.T, mock sqlmock.Sqlmock)
	validateOutput func(t *testing.T, output any, writer *httptest.ResponseRecorder)
}

func TestLoginCallback(t *testing.T) {
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

	infoEv := gcutil.LogInfo()
	errEv := gcutil.LogError(nil)

	for _, tc := range loginTestCases {
		t.Run(tc.desc, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if !assert.NoError(t, err) {
				t.FailNow()
			}
			defer db.Close()
			if !assert.NoError(t, gcsql.SetTestingDB("mysql", "gochan", "", db)) {
				t.FailNow()
			}
			defer assert.NoError(t, mock.ExpectationsWereMet())

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
			output, err := loginCallback(writer, request, tc.staff, tc.wantsJSON, infoEv, errEv)
			assert.Equal(t, tc.expectStatus, writer.Code)
			if tc.expectError {
				assert.Error(t, err)
				if tc.validateOutput != nil {
					tc.validateOutput(t, output, writer)
				}
			} else {
				assert.NoError(t, err)
				if tc.validateOutput == nil {
					t.Fatal("validateOutput is nil")
				}
				tc.validateOutput(t, output, writer)
			}
		})
	}
}
