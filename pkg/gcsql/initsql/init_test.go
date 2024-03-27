package initsql

import (
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

func TestBanMaskTmplFunc(t *testing.T) {
	testCases := []struct {
		desc       string
		rangeStart string
		rangeEnd   string
		expects    string
		banID      int
	}{
		{
			desc: "expect empty string if either value is enpty",
		},
		{
			desc:       "expect rangeStart if banID is 0 and rangeStart == rangEnd",
			rangeStart: "192.168.56.1",
			rangeEnd:   "192.168.56.1",
			expects:    "192.168.56.1",
		},
		{
			desc:       `expect "?" if an error is received and banID > 0`,
			banID:      1,
			rangeStart: "lol",
			rangeEnd:   "lmao",
			expects:    "?",
		},
		{
			desc:       "expect CIDR if ban exists, comparison is valid, and IPs differ (IPv4)",
			banID:      1,
			rangeStart: "192.168.56.0",
			rangeEnd:   "192.168.56.255",
			expects:    "192.168.56.0/24",
		},
		{
			desc:       "expect CIDR if ban exists, comparison is valid, and IPs differ (IPv6)",
			banID:      1,
			rangeStart: "2801::",
			rangeEnd:   "2801::ffff",
			expects:    "2801::/112",
		},
		{
			desc:       "expect IP if ban exists, comparison is valid, and IPs are the same (IPv4)",
			banID:      1,
			rangeStart: "192.168.56.1",
			rangeEnd:   "192.168.56.1",
			expects:    "192.168.56.1",
		},
	}
	var ban gcsql.IPBan
	for _, tC := range testCases {
		t.Run(tC.desc, func(tr *testing.T) {
			ban = gcsql.IPBan{
				ID:         tC.banID,
				RangeStart: tC.rangeStart,
				RangeEnd:   tC.rangeEnd,
			}
			result := banMaskTmplFunc(ban)
			assert.Equal(tr, tC.expects, result)
		})
	}
}
