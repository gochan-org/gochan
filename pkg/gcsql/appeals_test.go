package gcsql

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

var (
	testCasesGetAppeals = []testCaseGetAppeals{
		{
			name:         "single appeal, no results",
			args:         argsGetAppeals{1, 1},
			expectReturn: nil,
		},
		{
			name: "single appeal, with result",
			args: argsGetAppeals{1, 1},
			expectReturn: []IPBanAppeal{
				{ID: 1},
			},
		},
		{
			name:         "all appeals, no results",
			args:         argsGetAppeals{0, 1},
			expectReturn: nil,
		},
		{
			name:         "all appeals, with results",
			args:         argsGetAppeals{0, 10},
			expectReturn: []IPBanAppeal{{}, {}, {}},
		},
	}
	testCasesApproveAppeals = []testCaseApproveAppeals{
		{
			name: "approve nonexistent appeal",
			args: argsApproveAppeal{1, 1},
		},
	}
)

type testCaseGetAppeals struct {
	name         string
	args         argsGetAppeals
	expectReturn []IPBanAppeal
}

type testCaseApproveAppeals struct {
	name string
	args argsApproveAppeal
}

type argsGetAppeals struct {
	banID int
	limit int
}

type argsApproveAppeal struct {
	appealID int
	staffID  int
}

func testRunnerGetAppeals(t *testing.T, tC *testCaseGetAppeals, driver string) {
	t.Helper()
	mock, err := setupMockDB(t, driver, "gochan")
	if !assert.NoError(t, err) {
		return
	}

	query := `SELECT id, staff_id, ip_ban_id, appeal_text, staff_response, is_denied FROM ip_ban_appeals`
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
	if tC.args.limit > 0 {
		query += " LIMIT " + strconv.Itoa(tC.args.limit)
	}
	expectQuery := mock.ExpectPrepare(query).ExpectQuery()
	if tC.args.banID > 0 {
		expectQuery.WithArgs(tC.args.banID)
	}

	expectedRows := sqlmock.NewRows([]string{"id", "staff_id", "ip_ban_id", "appeal_text", "staff_response", "is_denied"})
	if len(tC.expectReturn) > 0 {
		for _, expectedBan := range tC.expectReturn {
			expectedRows.AddRow(
				expectedBan.ID, expectedBan.StaffID, expectedBan.IPBanID, expectedBan.AppealText,
				expectedBan.StaffResponse, expectedBan.IsDenied,
			)
		}
	}
	expectQuery.WillReturnRows(expectedRows)

	got, err := GetAppeals(tC.args.banID, tC.args.limit)
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
				testRunnerGetAppeals(t, &tC, driver)
			})
		}
	}
}

func testRunnerApproveAppeal(t *testing.T, tC *testCaseApproveAppeals, sqlDriver string) {
	t.Helper()
	mock, err := setupMockDB(t, sqlDriver, "gochan")
	if !assert.NoError(t, err) {
		return
	}

	deactivateQuery := `UPDATE ip_ban SET is_active = FALSE WHERE id = \(\s+` +
		`SELECT ip_ban_id FROM ip_ban_appeals WHERE id = `
	deactivateAppealQuery := `INSERT INTO ip_ban_audit\s*\(\s*ip_ban_id, timestamp, ` +
		`staff_id, is_active, is_thread_ban, permanent, staff_note, message, can_appeal\)\s*VALUES\(\(` +
		`SELECT ip_ban_id FROM ip_ban_appeals WHERE id = `
	deleteAppealQuery := `DELETE FROM ip_ban_appeals WHERE id = `

	switch sqlDriver {
	case "mysql":
		deactivateQuery += `\?\)`
		deactivateAppealQuery += `\?\),\s*CURRENT_TIMESTAMP, \?, FALSE, FALSE, FALSE, '', '', TRUE\)`
		deleteAppealQuery += `\?`
	case "sqlite3":
		fallthrough
	case "postgres":
		deactivateQuery += `\$1\)`
		deactivateAppealQuery += `\$1\),\s+CURRENT_TIMESTAMP, \$2, FALSE, FALSE, FALSE, '', '', TRUE\)`
		deleteAppealQuery += `\$1`
	}
	mock.ExpectBegin()
	mock.ExpectPrepare(deactivateQuery).ExpectExec().
		WithArgs(tC.args.appealID).WillReturnResult(driver.ResultNoRows)

	mock.ExpectPrepare(deactivateAppealQuery).ExpectExec().
		WithArgs(tC.args.appealID, tC.args.staffID).
		WillReturnResult(driver.ResultNoRows)

	mock.ExpectPrepare(deleteAppealQuery).ExpectExec().
		WithArgs(tC.args.appealID).
		WillReturnResult(driver.ResultNoRows)
	mock.ExpectCommit()

	assert.NoError(t, ApproveAppeal(tC.args.appealID, tC.args.staffID))
	assert.NoError(t, mock.ExpectationsWereMet())

	closeMock(t, mock)
}

func TestApproveAppeal(t *testing.T) {
	for _, tC := range testCasesApproveAppeals {
		for _, sqlDriver := range testingDBDrivers {
			t.Run(fmt.Sprintf("%s (%s)", tC.name, sqlDriver), func(t *testing.T) {
				testRunnerApproveAppeal(t, &tC, sqlDriver)
			})
		}
	}
}
