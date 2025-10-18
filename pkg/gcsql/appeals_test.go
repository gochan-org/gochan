package gcsql

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	testCasesGetAppeals = []testCaseGetAppeals{
		{
			name:         "single appeal, no results",
			args:         argsGetAppeals{banID: 1, limit: 1},
			expectReturn: nil,
		},
		{
			name: "single appeal, with result",
			args: argsGetAppeals{banID: 1, limit: 1},
			expectReturn: []Appeal{
				{IPBanAppeal: IPBanAppeal{ID: 1}},
			},
		},
		{
			name:         "all appeals, no results",
			args:         argsGetAppeals{limit: 1},
			expectReturn: nil,
		},
		{
			name:         "all appeals, with results",
			args:         argsGetAppeals{limit: 10},
			expectReturn: []Appeal{{}, {}, {}},
		},
	}
	testCasesApproveAppeals = []testCaseApproveAppeals{
		{
			name:        "approve nonexistent appeal",
			appealID:    1,
			staffID:     1,
			ban:         nil,
			expectID:    1,
			expectError: true,
		},
		{
			name:     "approve appeal",
			appealID: 1,
			staffID:  1,
			ban: &IPBan{
				ID: 1,
				IPBanBase: IPBanBase{
					StaffID:   1,
					Message:   "Test ban",
					CanAppeal: true,
					ExpiresAt: time.Now().Add(time.Hour),
					IsActive:  true,
				},
				IssuedAt: time.Now(),
			},
			expectID:     1,
			expectActive: true,
		},
	}
)

type testCaseGetAppeals struct {
	name         string
	args         argsGetAppeals
	expectReturn []Appeal
}

type testCaseApproveAppeals struct {
	name         string
	appealID     int
	staffID      int
	ban          *IPBan
	expectID     int
	expectActive bool
	expectError  bool
}

type argsGetAppeals struct {
	banID     int
	limit     int
	orderDesc bool
}

func testRunnerGetAppeals(t *testing.T, tC *testCaseGetAppeals, driver string) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, SetTestingDB(driver, "gochan", "", db)) {
		return
	}

	query := `SELECT id, staff_id, staff_username, ip_ban_id, appeal_text, staff_response, is_denied, timestamp FROM v_appeals`
	if tC.args.banID > 0 {
		switch driver {
		case "mysql":
			query += ` WHERE ip_ban_id = \?`
		case "sqlite3":
			fallthrough
		case "postgres":
			query += ` WHERE ip_ban_id = \$1`
		}
	}
	if tC.args.orderDesc {
		query += " ORDER BY id DESC"
	} else {
		query += " ORDER BY id ASC"
	}
	if tC.args.limit > 0 {
		query += " LIMIT " + strconv.Itoa(tC.args.limit)
	}
	expectQuery := mock.ExpectPrepare(query).ExpectQuery()
	if tC.args.banID > 0 {
		expectQuery.WithArgs(tC.args.banID)
	}

	expectedRows := sqlmock.NewRows([]string{"id", "staff_id", "staff_username", "ip_ban_id", "appeal_text", "staff_response", "is_denied", "timestamp"})
	if len(tC.expectReturn) > 0 {
		for _, expectedBan := range tC.expectReturn {
			expectedRows.AddRow(
				expectedBan.ID, expectedBan.StaffID, expectedBan.StaffUsername, expectedBan.IPBanID, expectedBan.AppealText,
				expectedBan.StaffResponse, expectedBan.IsDenied, expectedBan.Timestamp,
			)
		}
	}
	expectQuery.WillReturnRows(expectedRows)

	got, err := GetAppeals(tC.args.banID, tC.args.limit, tC.args.orderDesc)
	if !assert.NoError(t, err) {
		return
	}
	assert.NoError(t, mock.ExpectationsWereMet())

	assert.LessOrEqual(t, len(got), tC.args.limit)
	assert.Equal(t, tC.expectReturn, got)
	if tC.args.banID > 0 && tC.expectReturn != nil {
		assert.Equal(t, tC.args.banID, tC.expectReturn[0].ID)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
	closeMock(t, mock)

}

func TestGetAppeals(t *testing.T) {
	for _, tC := range testCasesGetAppeals {
		for _, driver := range testingDBDrivers {
			t.Run(fmt.Sprintf("%s (%s)", tC.name, driver), func(t *testing.T) {
				config.SetTestDBConfig(driver, "localhost", "gochan", "gochan", "gochan", "")
				testRunnerGetAppeals(t, &tC, driver)
			})
		}
	}
}

func testRunnerApproveAppeal(t *testing.T, tC *testCaseApproveAppeals, sqlDriver string) {
	db, mock, err := sqlmock.New()
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, SetTestingDB(sqlDriver, "gochan", "", db)) {
		return
	}

	checkAppealsSQL := `SELECT ip_ban_id, is_ban_active FROM v_appeals WHERE id = `
	deactivateSQL := `UPDATE ip_ban SET is_active = FALSE WHERE id = `
	insertBanAudit := `INSERT INTO ip_ban_audit\s*\(ip_ban_id, staff_id, is_active, is_thread_ban, expires_at, appeal_at, permanent, ` +
		`staff_note, message, can_appeal\)\s*VALUES\(`
	insertAppealsAudit := `INSERT INTO ip_ban_appeals_audit \(appeal_id, appeal_text, staff_id, staff_response, is_denied\)\s*VALUES\(`
	switch sqlDriver {
	case "mysql":
		checkAppealsSQL += `\?`
		deactivateSQL += `\?`
		insertBanAudit += `\?, \?, FALSE, \?, \?, \?, \?, \?, \?, \?\)`
		insertAppealsAudit += `\?, \(SELECT appeal_text FROM ip_ban_appeals WHERE id = \?\), \?, 'Appeal approved, ban deactivated.', FALSE\)`
	case "sqlite3", "postgres":
		checkAppealsSQL += `\$1`
		deactivateSQL += `\$1`
		insertBanAudit += `\$1, \$2, FALSE, \$3, \$4, \$5, \$6, \$7, \$8, \$9\)`
		insertAppealsAudit += `\$1, \(SELECT appeal_text FROM ip_ban_appeals WHERE id = \$2\), \$3, 'Appeal approved, ban deactivated.', FALSE\)`
	}
	checkAppealsSQL += " AND is_denied = FALSE"

	mock.ExpectBegin()
	mock.ExpectPrepare(checkAppealsSQL).ExpectQuery().WithArgs(tC.appealID).
		WillReturnRows(sqlmock.NewRows([]string{"ip_ban_id", "is_ban_active"}).AddRow(tC.expectID, tC.expectActive))
	if tC.expectActive {
		mockSetupGetIPBanByID(t, mock, tC.expectID, gcdb.driver, tC.ban)

		mock.ExpectPrepare(deactivateSQL).ExpectExec().
			WithArgs(tC.appealID).WillReturnResult(driver.ResultNoRows)

		mock.ExpectPrepare(insertBanAudit).ExpectExec().
			WithArgs(tC.ban.ID, tC.ban.StaffID, tC.ban.IsThreadBan, tC.ban.ExpiresAt, tC.ban.AppealAt,
				tC.ban.Permanent, tC.ban.StaffNote, tC.ban.Message, tC.ban.CanAppeal).
			WillReturnResult(driver.ResultNoRows)

		mock.ExpectPrepare(insertAppealsAudit).ExpectExec().
			WithArgs(tC.appealID, tC.appealID, tC.staffID).
			WillReturnResult(driver.ResultNoRows)

		mock.ExpectCommit()
	} else {
		mock.ExpectRollback()
	}

	err = ApproveAppeal(tC.appealID, tC.staffID)
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	if tC.expectError {
		assert.Error(t, err)
		return
	} else {
		assert.NoError(t, err)
	}

	closeMock(t, mock)
}

func TestApproveAppeal(t *testing.T) {
	config.InitTestConfig()

	tempDir := t.TempDir()
	gcutil.InitLogs(tempDir, &gcutil.LogOptions{
		LogLevel: zerolog.TraceLevel,
	})
	for _, tC := range testCasesApproveAppeals {
		for _, sqlDriver := range testingDBDrivers {
			t.Run(fmt.Sprintf("%s (%s)", tC.name, sqlDriver), func(t *testing.T) {
				config.SetTestDBConfig(sqlDriver, "localhost", "gochan", "gochan", "gochan", "")
				testRunnerApproveAppeal(t, &tC, sqlDriver)
			})
		}
	}
}
