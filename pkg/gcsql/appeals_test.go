package gcsql

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/stretchr/testify/assert"
)

type argsGetAppeals struct {
	banID int
	limit int
}

func TestGetAppeals(t *testing.T) {
	_, err := goToGochanRoot(t)
	if !assert.NoError(t, err) {
		return
	}
	config.SetVersion("3.10.1")
	config.SetRandomSeed("test")

	testCases := []struct {
		name         string
		args         argsGetAppeals
		expectReturn []IPBanAppeal
		wantErr      bool
	}{
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
	var mock sqlmock.Sqlmock
	for _, tC := range testCases {
		for _, driver := range testingDBDrivers {
			t.Run(fmt.Sprintf("%s (%s)", tC.name, driver), func(t *testing.T) {
				gcdb, err = setupDBConn("localhost", driver, "gochan", "gochan", "gochan", "")
				if !assert.NoError(t, err) {
					return
				}
				gcdb.db, mock, err = sqlmock.New()
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
				if tC.wantErr {
					assert.Error(t, err)
					return
				} else {
					if !assert.NoError(t, err) {
						return
					}
				}
				assert.NoError(t, mock.ExpectationsWereMet())

				assert.LessOrEqual(t, len(got), tC.args.limit)
				assert.Equal(t, tC.expectReturn, got)
				if tC.args.banID > 0 && tC.expectReturn != nil {
					assert.Equal(t, tC.args.banID, tC.expectReturn[0].ID)
				}
				assert.NoError(t, mock.ExpectationsWereMet())
				closeMock(t, mock)
			})
		}
	}
}
