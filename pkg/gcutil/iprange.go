package gcutil

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var (
	ErrInvalidIP     = errors.New("invalid IP address")
	ErrInvalidSubnet = errors.New("invalid IP address or subnet mask")
)

// ParseIPRange takes a single IP address or an IP range of the form "networkIP/netmaskbits" and
// gives the starting IP and ending IP in the subnet
//
// More info: https://en.wikipedia.org/wiki/Subnet
func ParseIPRange(ipOrCIDR string) (string, string, error) {
	parts := strings.Split(ipOrCIDR, "/")
	ip := net.ParseIP(parts[0])
	if ip == nil {
		return "", "", ErrInvalidIP
	}
	ipv4 := ip.To4()
	if ipv4 != nil {
		ip = ipv4
	}
	if len(parts) == 1 {
		// single IP
		return ipOrCIDR, ipOrCIDR, nil
	} else if len(parts) == 2 {
		// IP/mask
		netBits, err := strconv.Atoi(parts[1])
		if err != nil {
			return "", "", err
		}
		mask := net.CIDRMask(netBits, len(ip)*8)
		broadcast := net.IP(make([]byte, len(ip)))
		for i := range ip {
			broadcast[i] = ip[i] | ^mask[i]
		}
		return ip.Mask(mask).String(), broadcast.String(), nil
	}
	return "", "", ErrInvalidSubnet
}

// GetIPRangeSubnet returns the smallest subnet that contains the start and end
// IP addresses, and any errors that occured
func GetIPRangeSubnet(start string, end string) (*net.IPNet, error) {
	startIP := net.ParseIP(start)
	endIP := net.ParseIP(end)
	if startIP == nil {
		return nil, fmt.Errorf("invalid IP address %s", start)
	}
	if endIP == nil {
		return nil, fmt.Errorf("invalid IP address %s", end)
	}
	if len(startIP) != len(endIP) {
		return nil, errors.New("ip addresses must both be IPv4 or IPv6")
	}

	if startIP.To4() != nil {
		startIP = startIP.To4()
		endIP = endIP.To4()
	}

	bits := 0
	var ipn *net.IPNet
	for b := range startIP {
		if startIP[b] == endIP[b] {
			bits += 8
			continue
		}
		for i := 7; i >= 0; i-- {
			if startIP[b]&(1<<i) == endIP[b]&(1<<i) {
				bits++
				continue
			}
			ipn = &net.IPNet{IP: startIP, Mask: net.CIDRMask(bits, len(startIP)*8)}
			return ipn, nil
		}
	}
	return &net.IPNet{IP: startIP, Mask: net.CIDRMask(bits, len(startIP)*8)}, nil
}
