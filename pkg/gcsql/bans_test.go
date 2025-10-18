package gcsql

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

var (
	testCasesGetIPBanByID = []testCaseGetIPBanByID{
		{
			name:         "get existing ban",
			banID:        1,
			expectReturn: &IPBan{ID: 1, IPBanBase: IPBanBase{Message: "Test ban"}},
			expectError:  false,
		},
		{
			name:        "get nonexistent ban",
			banID:       999,
			expectError: true,
		},
	}
)

type testCaseGetIPBanByID struct {
	name         string
	banID        int
	expectReturn *IPBan
	expectError  bool
}

func mockSetupGetIPBanByID(t *testing.T, mock sqlmock.Sqlmock, banID int, driver string, expects *IPBan) {
	t.Helper()

	getBanSQL := `SELECT\s+id, staff_id, board_id, banned_for_post_id, copy_post_text, is_thread_ban,\s*is_active, `
	getBanSQL2 := `\s*issued_at, appeal_at, expires_at,\s*permanent,\s*staff_note, message, can_appeal\s+FROM ip_ban WHERE id = `

	var rangeStartColumn, rangeEndColumn string
	switch driver {
	case "mysql":
		rangeStartColumn = "INET6_NTOA(range_start)"
		rangeEndColumn = "INET6_NTOA(range_end)"
		getBanSQL += `INET6_NTOA\(range_start\), INET6_NTOA\(range_end\),`
		getBanSQL2 += `\?`
	case "sqlite3", "postgres":
		rangeStartColumn = "range_start"
		rangeEndColumn = "range_end"
		getBanSQL += "range_start, range_end,"
		getBanSQL2 += `\$1`
	}

	expectQuery := mock.ExpectPrepare(getBanSQL + getBanSQL2).ExpectQuery().WithArgs(banID)
	if expects != nil {
		expectQuery.WillReturnRows(
			sqlmock.NewRows([]string{"id", "staff_id", "board_id", "banned_for_post_id", "copy_post_text", "is_thread_ban", "is_active",
				rangeStartColumn, rangeEndColumn, "issued_at", "appeal_at", "expires_at", "permanent", "staff_note", "message", "can_appeal"}).
				AddRow(banID, expects.StaffID, expects.BoardID, expects.BannedForPostID, expects.CopyPostText, expects.IsThreadBan, expects.IsActive,
					expects.RangeStart, expects.RangeEnd, expects.IssuedAt, expects.AppealAt, expects.ExpiresAt,
					expects.Permanent, expects.StaffNote, expects.Message, expects.CanAppeal))

	}

}

func TestGetIPBanByID(t *testing.T) {
	config.InitTestConfig()
	tempDir := t.TempDir()
	gcutil.InitLogs(tempDir, &gcutil.LogOptions{
		LogLevel: zerolog.TraceLevel,
	})
	for _, driver := range []string{"mysql", "postgres", "sqlite3"} {
		for _, tC := range testCasesGetIPBanByID {
			t.Run(tC.name+"_"+driver, func(t *testing.T) {
				db, mock, err := sqlmock.New()
				if !assert.NoError(t, err) {
					t.FailNow()
				}
				systemCriticalConfig := config.GetSystemCriticalConfig()
				systemCriticalConfig.DBtype = driver
				systemCriticalConfig.DBname = "gochan"
				systemCriticalConfig.DBprefix = ""
				config.SetSystemCriticalConfig(systemCriticalConfig)
				if !assert.NoError(t, SetTestingDB(driver, systemCriticalConfig.DBname, systemCriticalConfig.DBprefix, db)) {
					t.FailNow()
				}

				mockSetupGetIPBanByID(t, mock, tC.banID, driver, tC.expectReturn)
				ban, err := GetIPBanByID(nil, tC.banID)
				if tC.expectError {
					assert.Error(t, err)
				} else {
					if !assert.NoError(t, err) {
						t.FailNow()
					}
					assert.Equal(t, tC.banID, ban.ID)
					assert.Equal(t, tC.expectReturn.Message, ban.Message)
				}
			})
		}
	}
}
