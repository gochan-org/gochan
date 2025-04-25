package manage

import (
	"database/sql/driver"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/gcutil/testutil"
	"github.com/stretchr/testify/assert"
)

const (
	ipBanInsertRE = `INSERT INTO ip_ban\s*\(staff_id,\s*board_id,\s*banned_for_post_id,\s*copy_post_text,\s*is_thread_ban,\s*is_active,` +
		`\s*range_start,\s*range_end,\s*appeal_at,\s*expires_at,\s*permanent,\s*staff_note,\s*message,\s*can_appeal\)\s+` +
		`VALUES\(\?, \?, \?, \?, \?, \?, INET6_ATON\(\?\), INET6_ATON\(\?\), \?, \?, \?, \?, \?, \?\)`
)

var (
	newIPBanFromRequestTestCases = []banTestCase{
		{
			desc: "single IP, 1 hour total ban, no appeals",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now(),
					Message:   "reason",
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.1"},
				"duration":  {"1h"},
				"noappeals": {"on"},
				"reason":    {"reason"},
			},
		},
		{
			desc: "single IP, 1 hour thread ban, no appeals",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt:   time.Now().Add(time.Hour),
					AppealAt:    time.Now(),
					Message:     "reason",
					IsThreadBan: true,
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.1"},
				"duration":  {"1h"},
				"noappeals": {"on"},
				"reason":    {"reason"},
				"threadban": {"on"},
			},
		},
		{
			desc: "single IP, 1 hour total ban, immediate appeal",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now(),
					Message:   "reason",
					CanAppeal: true,
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.1"},
				"duration":  {"1h"},
				"noappeals": {"off"},
				"reason":    {"reason"},
			},
		},
		{
			desc: "single IP, 1 hour total ban, appeal in 30 minutes",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now().Add(30 * time.Minute),
					Message:   "reason",
					CanAppeal: true,
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":         {"add"},
				"ip":         {"192.168.56.1"},
				"duration":   {"1h"},
				"noappeals":  {"off"},
				"appealwait": {"30m"},
				"reason":     {"reason"},
			},
		},
		{
			desc: "single IP, permaban, appeal in 30 minutes",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					AppealAt:  time.Now().Add(30 * time.Minute),
					Message:   "reason",
					CanAppeal: true,
					Permanent: true,
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":         {"add"},
				"ip":         {"192.168.56.1"},
				"duration":   {""},
				"appealwait": {"30m"},
				"reason":     {"reason"},
			},
		},
		{
			desc: "single IP, 1 hour total ban, no appeals, with staff note",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now(),
					Message:   "reason",
					StaffNote: "staff note",
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.1"},
				"duration":  {"1h"},
				"noappeals": {"on"},
				"reason":    {"reason"},
				"staffnote": {"staff note"},
			},
		},
		{
			desc: "IP subnet ban, 1 hour total ban, no appeals",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now(),
					Message:   "reason",
				},
				RangeStart: "192.168.56.0",
				RangeEnd:   "192.168.56.255",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.0/24"},
				"duration":  {"1h"},
				"noappeals": {"on"},
				"reason":    {"reason"},
				"boardid":   {"1"},
			},
			boardID: 1,
		},
		{
			desc: "Board ban, 1 hour total ban, no appeals",
			expectBan: gcsql.IPBan{
				IPBanBase: gcsql.IPBanBase{
					ExpiresAt: time.Now().Add(time.Hour),
					AppealAt:  time.Now(),
					Message:   "reason",
				},
				RangeStart: "192.168.56.1",
				RangeEnd:   "192.168.56.1",
			},
			form: url.Values{
				"do":        {"add"},
				"ip":        {"192.168.56.1"},
				"duration":  {"1h"},
				"noappeals": {"on"},
				"reason":    {"reason"},
				"boardid":   {"1"},
			},
			boardID: 1,
		},
	}
)

type banTestCase struct {
	desc      string
	expectBan gcsql.IPBan
	exptError bool
	method    string
	form      url.Values
	boardID   int
}

func TestIPBanFromRequest(t *testing.T) {
	mock, err := gcsql.SetupMockDB("mysql")
	if err != nil {
		t.Fatalf("Failed to setup mock DB: %v", err)
	}

	for _, tc := range newIPBanFromRequestTestCases {
		tc.method = "POST"
		t.Run(tc.desc, func(t *testing.T) {
			request := &http.Request{
				Method:   tc.method,
				PostForm: tc.form,
			}
			infoEv := gcutil.LogInfo()
			errEv := gcutil.LogError(nil)
			var ban gcsql.IPBan
			ban.IPBanBase.StaffID = 1
			if tc.boardID > 0 {
				tc.expectBan.BoardID = &tc.boardID
			}
			ban.IssuedAt = time.Now()
			tc.expectBan.IssuedAt = ban.IssuedAt

			mock.ExpectBegin()
			mock.ExpectPrepare(ipBanInsertRE).ExpectExec().
				WithArgs(1, tc.expectBan.BoardID, nil, "", tc.expectBan.IsThreadBan, true,
					tc.expectBan.RangeStart, tc.expectBan.RangeEnd,
					testutil.FuzzyTime(tc.expectBan.AppealAt), testutil.FuzzyTime(tc.expectBan.ExpiresAt),
					tc.expectBan.Permanent, tc.expectBan.StaffNote, tc.expectBan.Message, tc.expectBan.CanAppeal,
				).WillReturnResult(driver.ResultNoRows)
			mock.ExpectCommit()

			err := ipBanFromRequest(&ban, request, infoEv, errEv)
			if tc.exptError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectBan.Permanent, ban.Permanent)
				assert.Equal(t, tc.expectBan.CanAppeal, ban.CanAppeal)
				assert.Equal(t, tc.expectBan.RangeStart, ban.RangeStart)
				assert.Equal(t, tc.expectBan.RangeEnd, ban.RangeEnd)
				assert.Equal(t, tc.expectBan.ExpiresAt.Truncate(time.Minute), ban.ExpiresAt.Truncate(time.Minute))
				assert.Equal(t, tc.expectBan.AppealAt.Truncate(time.Minute), ban.AppealAt.Truncate(time.Minute))
				if tc.expectBan.BoardID == nil {
					assert.Nil(t, ban.BoardID)
				} else {
					assert.Equal(t, *tc.expectBan.BoardID, *ban.BoardID)
				}
			}
		})
	}
}
