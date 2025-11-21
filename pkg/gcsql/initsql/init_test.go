package initsql

import (
	"database/sql"
	"net"
	"testing"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/stretchr/testify/assert"
)

type banMaskTestCase struct {
	desc       string
	rangeStart string
	rangeEnd   string
	expects    string
	banID      int
}

type inetConversionTestCase[I any, E any] struct {
	desc         string
	input        I
	inetExpects  E
	inet6Expects E
}

func TestBanMaskTmplFunc(t *testing.T) {
	testCases := []banMaskTestCase{
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

func TestInetNtoA(t *testing.T) {
	testCases := []inetConversionTestCase[any, sql.NullString]{
		{
			desc:         "convert valid IP number to IPv4 address",
			input:        3232249859,
			inet6Expects: sql.NullString{String: "192.168.56.3", Valid: true},
			inetExpects:  sql.NullString{String: "192.168.56.3", Valid: true},
		},
		{
			desc:         "convert valid IPv6 address bytes to string",
			input:        net.ParseIP("2601::1"),
			inet6Expects: sql.NullString{String: "2601::1", Valid: true},
		},
		{
			desc:  "convert invalid IP address bytes to string",
			input: []byte{1, 2, 3},
		},
	}
	db, err := sql.Open("sqlite3-inet6", ":memory:")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()
	var ipStr sql.NullString
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if !assert.NoError(t, db.QueryRow("SELECT INET_NTOA(?)", tC.input).Scan(&ipStr)) {
				t.FailNow()
			}
			assert.Equal(t, tC.inetExpects, ipStr)
			if !assert.NoError(t, db.QueryRow("SELECT INET6_NTOA(?)", tC.input).Scan(&ipStr)) {
				t.FailNow()
			}
			assert.Equal(t, tC.inet6Expects, ipStr)
		})
	}
	assert.NoError(t, db.Close())
}

func TestInetAtoN(t *testing.T) {
	testCases := []inetConversionTestCase[string, []byte]{
		{
			desc:         "convert valid IPv4 address string to bytes",
			input:        "192.168.56.3",
			inetExpects:  net.ParseIP("192.168.56.3").To16(),
			inet6Expects: net.ParseIP("192.168.56.3").To16(),
		},
		{
			desc:         "convert valid IPv6 address string to bytes",
			input:        "2601::1",
			inetExpects:  nil,
			inet6Expects: net.ParseIP("2601::1").To16(),
		},
		{
			desc:         "invalid IP address returns null",
			input:        "hmm",
			inetExpects:  nil,
			inet6Expects: nil,
		},
	}
	db, err := sql.Open("sqlite3-inet6", ":memory:")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()
	var ip []byte
	for _, tC := range testCases {
		t.Run(tC.desc, func(t *testing.T) {
			if !assert.NoError(t, db.QueryRow("SELECT INET_ATON(?)", tC.input).Scan(&ip)) {
				t.FailNow()
			}
			assert.Equal(t, tC.inetExpects, ip)
			if !assert.NoError(t, db.QueryRow("SELECT INET6_ATON(?)", tC.input).Scan(&ip)) {
				t.FailNow()
			}
			assert.Equal(t, tC.inet6Expects, ip)
		})
	}
	assert.NoError(t, db.Close())
}

func TestIPCmp(t *testing.T) {
	db, err := sql.Open("sqlite3-inet6", ":memory:")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer db.Close()

	setup := []string{
		"create table addrs(id integer primary key autoincrement, ip varbinary(16))",
		"insert into addrs(ip) values (inet_aton('192.168.56.1')), (inet6_aton('192.168.56.2')), (inet6_aton('2601:8000::1')), (inet6_aton('2601:f000::'))",
	}
	for _, stmt := range setup {
		_, err := db.Exec(stmt)
		if !assert.NoError(t, err) {
			t.FailNow()
		}
	}
	var result []net.IP
	rows, err := db.Query("SELECT ip FROM addrs WHERE ip_cmp(ip, '192.168.56.1') > 0")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	for rows.Next() {
		var ip net.IP
		if !assert.NoError(t, rows.Scan(&ip)) {
			t.FailNow()
		}
		result = append(result, ip)
		t.Log(ip.String())
	}
	assert.Equal(t, 1, len(result))
	result = []net.IP{}
	rows, err = db.Query("SELECT ip FROM addrs WHERE ip_cmp(ip, '2601:9000::1') >= 0")
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	for rows.Next() {
		var ip net.IP
		if !assert.NoError(t, rows.Scan(&ip)) {
			t.FailNow()
		}
		result = append(result, ip)
		// t.Log(ip)
	}

	assert.Equal(t, 1, len(result))
	assert.NoError(t, db.Close())
}
