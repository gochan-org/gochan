package gcutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

type startEndCidr struct {
	cidr  string
	start string
	end   string
}

func TestIPRangeErrOnInvalidIP(t *testing.T) {
	rangeTestCases := []string{
		"not an IP",
		"",
		"192.168.56.0/",
		"192.168.56.0/24/1",
	}
	var err error
	for _, input := range rangeTestCases {
		t.Run(input, func(tr *testing.T) {
			_, _, err = ParseIPRange(input)
			assert.Error(tr, err)
		})
	}

	subnetTestCases := []startEndCidr{
		{start: "not", end: "ip"},
		{start: "::1", end: "ip"},
		{start: "ip", end: "::1"},
		{start: "127.0.0.1", end: "::1"},
		{start: "::1", end: "127.0.0.1"},
	}
	for _, tC := range subnetTestCases {
		t.Run("start = "+tC.start+", end = "+tC.end, func(tr *testing.T) {
			_, err = GetIPRangeSubnet(tC.start, tC.end)
			assert.Error(tr, err)
		})
	}
}

func TestIPRangeSingleIP(t *testing.T) {
	start, end, err := ParseIPRange("192.168.56.1")
	assert.NoError(t, err)
	assert.Equal(t, start, end)
	start, end, err = ParseIPRange("2801::")
	assert.NoError(t, err)
	assert.Equal(t, start, end)
}

func TestIPRangeIPv4Range(t *testing.T) {
	testCases := []startEndCidr{
		{cidr: "192.168.56.0/24", start: "192.168.56.0", end: "192.168.56.255"},
		{cidr: "192.168.0.0/16", start: "192.168.0.0", end: "192.168.255.255"},
		{cidr: "192.0.0.0/8", start: "192.0.0.0", end: "192.255.255.255"},
	}
	var start, end string
	var err error
	var ipn *net.IPNet
	for _, tC := range testCases {
		t.Run(tC.cidr, func(tr *testing.T) {
			start, end, err = ParseIPRange(tC.cidr)
			assert.NoError(tr, err)
			assert.Equal(tr, tC.start, start)
			assert.Equal(tr, tC.end, end)
			ipn, err = GetIPRangeSubnet(start, end)
			assert.NoError(tr, err)
			assert.Equal(tr, tC.cidr, ipn.String())
		})
	}
}

func TestIPRangeIPv6Range(t *testing.T) {
	testCases := []startEndCidr{
		{cidr: "2607:f8b0:400a:80a::2010/124", start: "2607:f8b0:400a:80a::2010", end: "2607:f8b0:400a:80a::201f"},
		{cidr: "2607:f8b0:400a:80a::2000/120", start: "2607:f8b0:400a:80a::2000", end: "2607:f8b0:400a:80a::20ff"},
		{cidr: "2607:f8b0:400a:80a::2000/116", start: "2607:f8b0:400a:80a::2000", end: "2607:f8b0:400a:80a::2fff"},
		{cidr: "2607:f8b0:400a:80a::/112", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a::ffff"},
		{cidr: "2607:f8b0:400a:80a::/108", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a::f:ffff"},
		{cidr: "2607:f8b0:400a:80a::/104", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a::ff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/100", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a::fff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/96", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a::ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/92", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:0:f:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/88", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:0:ff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/84", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:0:fff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/80", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:0:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/76", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:f:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/72", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:ff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/68", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:fff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:80a::/64", start: "2607:f8b0:400a:80a::", end: "2607:f8b0:400a:80a:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:800::/60", start: "2607:f8b0:400a:800::", end: "2607:f8b0:400a:80f:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a:800::/56", start: "2607:f8b0:400a:800::", end: "2607:f8b0:400a:8ff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a::/52", start: "2607:f8b0:400a::", end: "2607:f8b0:400a:fff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:400a::/48", start: "2607:f8b0:400a::", end: "2607:f8b0:400a:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:4000::/44", start: "2607:f8b0:4000::", end: "2607:f8b0:400f:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:4000::/40", start: "2607:f8b0:4000::", end: "2607:f8b0:40ff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0:4000::/36", start: "2607:f8b0:4000::", end: "2607:f8b0:4fff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0::/32", start: "2607:f8b0::", end: "2607:f8b0:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f8b0::/28", start: "2607:f8b0::", end: "2607:f8bf:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f800::/24", start: "2607:f800::", end: "2607:f8ff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607:f000::/20", start: "2607:f000::", end: "2607:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2607::/16", start: "2607::", end: "2607:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2600::/12", start: "2600::", end: "260f:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2600::/8", start: "2600::", end: "26ff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
		{cidr: "2000::/4", start: "2000::", end: "2fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
	}
	var start, end string
	var err error
	var ipn *net.IPNet
	for _, tC := range testCases {
		t.Run(tC.cidr, func(tr *testing.T) {
			start, end, err = ParseIPRange(tC.cidr)
			assert.NoError(tr, err)
			assert.Equal(tr, tC.start, start)
			assert.Equal(tr, tC.end, end)

			ipn, err = GetIPRangeSubnet(start, end)
			assert.NoError(t, err)
			assert.Equal(t, tC.cidr, ipn.String())
		})
	}
}
