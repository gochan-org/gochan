package gcutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPRangeErrOnInvalidIP(t *testing.T) {
	_, _, err := ParseIPRange("not an IP")
	assert.Error(t, err)
	_, _, err = ParseIPRange("")
	assert.Error(t, err)
	_, _, err = ParseIPRange("192.168.56.0/")
	assert.Error(t, err)
	_, _, err = ParseIPRange("192.168.56.0/24/1")
	assert.Error(t, err)
	_, err = GetIPRangeSubnet("not", "ip")
	assert.Error(t, err)
	_, err = GetIPRangeSubnet("::1", "ip")
	assert.Error(t, err)
	_, err = GetIPRangeSubnet("::1", "127.0.0.1")
	assert.Error(t, err)
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
	ranges := []string{"192.168.56.0/24", "192.168.0.0/16", "192.0.0.0/8"}
	starts := []string{"192.168.56.0", "192.168.0.0", "192.0.0.0"}
	ends := []string{"192.168.56.255", "192.168.255.255", "192.255.255.255"}
	var start, end string
	var err error
	var ipn *net.IPNet
	for i := range ranges {
		start, end, err = ParseIPRange(ranges[i])
		assert.NoError(t, err)
		assert.Equal(t, starts[i], start)
		assert.Equal(t, ends[i], end)
		ipn, err = GetIPRangeSubnet(start, end)
		assert.NoError(t, err)
		assert.Equal(t, ranges[i], ipn.String())
	}
}

func TestIPRangeIPv6Range(t *testing.T) {
	ranges := []string{
		"2607:f8b0:400a:80a::2010/124",
		"2607:f8b0:400a:80a::2000/120",
		"2607:f8b0:400a:80a::2000/116",
		"2607:f8b0:400a:80a::/112",
		"2607:f8b0:400a:80a::/108",
		"2607:f8b0:400a:80a::/104",
		"2607:f8b0:400a:80a::/100",
		"2607:f8b0:400a:80a::/96",
		"2607:f8b0:400a:80a::/92",
		"2607:f8b0:400a:80a::/88",
		"2607:f8b0:400a:80a::/84",
		"2607:f8b0:400a:80a::/80",
		"2607:f8b0:400a:80a::/76",
		"2607:f8b0:400a:80a::/72",
		"2607:f8b0:400a:80a::/68",
		"2607:f8b0:400a:80a::/64",
		"2607:f8b0:400a:800::/60",
		"2607:f8b0:400a:800::/56",
		"2607:f8b0:400a::/52",
		"2607:f8b0:400a::/48",
		"2607:f8b0:4000::/44",
		"2607:f8b0:4000::/40",
		"2607:f8b0:4000::/36",
		"2607:f8b0::/32",
		"2607:f8b0::/28",
		"2607:f800::/24",
		"2607:f000::/20",
		"2607::/16",
		"2600::/12",
		"2600::/8",
		"2000::/4",
	}
	starts := []string{
		"2607:f8b0:400a:80a::2010",
		"2607:f8b0:400a:80a::2000",
		"2607:f8b0:400a:80a::2000",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:80a::",
		"2607:f8b0:400a:800::",
		"2607:f8b0:400a:800::",
		"2607:f8b0:400a::",
		"2607:f8b0:400a::",
		"2607:f8b0:4000::",
		"2607:f8b0:4000::",
		"2607:f8b0:4000::",
		"2607:f8b0::",
		"2607:f8b0::",
		"2607:f800::",
		"2607:f000::",
		"2607::",
		"2600::",
		"2600::",
		"2000::",
	}
	ends := []string{
		"2607:f8b0:400a:80a::201f",
		"2607:f8b0:400a:80a::20ff",
		"2607:f8b0:400a:80a::2fff",
		"2607:f8b0:400a:80a::ffff",
		"2607:f8b0:400a:80a::f:ffff",
		"2607:f8b0:400a:80a::ff:ffff",
		"2607:f8b0:400a:80a::fff:ffff",
		"2607:f8b0:400a:80a::ffff:ffff",
		"2607:f8b0:400a:80a:0:f:ffff:ffff",
		"2607:f8b0:400a:80a:0:ff:ffff:ffff",
		"2607:f8b0:400a:80a:0:fff:ffff:ffff",
		"2607:f8b0:400a:80a:0:ffff:ffff:ffff",
		"2607:f8b0:400a:80a:f:ffff:ffff:ffff",
		"2607:f8b0:400a:80a:ff:ffff:ffff:ffff",
		"2607:f8b0:400a:80a:fff:ffff:ffff:ffff",
		"2607:f8b0:400a:80a:ffff:ffff:ffff:ffff",
		"2607:f8b0:400a:80f:ffff:ffff:ffff:ffff",
		"2607:f8b0:400a:8ff:ffff:ffff:ffff:ffff",
		"2607:f8b0:400a:fff:ffff:ffff:ffff:ffff",
		"2607:f8b0:400a:ffff:ffff:ffff:ffff:ffff",
		"2607:f8b0:400f:ffff:ffff:ffff:ffff:ffff",
		"2607:f8b0:40ff:ffff:ffff:ffff:ffff:ffff",
		"2607:f8b0:4fff:ffff:ffff:ffff:ffff:ffff",
		"2607:f8b0:ffff:ffff:ffff:ffff:ffff:ffff",
		"2607:f8bf:ffff:ffff:ffff:ffff:ffff:ffff",
		"2607:f8ff:ffff:ffff:ffff:ffff:ffff:ffff",
		"2607:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"2607:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"260f:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"26ff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
		"2fff:ffff:ffff:ffff:ffff:ffff:ffff:ffff",
	}
	var start, end string
	var err error
	var ipn *net.IPNet
	for i := range ranges {
		start, end, err = ParseIPRange(ranges[i])
		assert.NoError(t, err)
		assert.Equal(t, starts[i], start, "unequal values at index %d", i)
		assert.Equal(t, ends[i], end, "unequal values at index %d", i)
		ipn, err = GetIPRangeSubnet(start, end)
		assert.NoError(t, err)
		assert.Equal(t, ranges[i], ipn.String())
	}
}
