package multiprovider

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/EdgeNet-project/edgenet/pkg/util"
)

func TestIpRange(t *testing.T) {
	cases := []struct {
		input    ipRange
		adress   net.IP
		expected bool
	}{
		{ipRange{net.ParseIP("0.0.0.0"), net.ParseIP("255.255.255.255")}, net.ParseIP("128.128.128.128"), true},
		{ipRange{net.ParseIP("0.0.0.0"), net.ParseIP("128.128.128.128")}, net.ParseIP("255.255.255.255"), false},
		{ipRange{net.ParseIP("74.50.153.0"), net.ParseIP("74.50.153.4")}, net.ParseIP("74.50.153.0"), true},
		{ipRange{net.ParseIP("74.50.153.0"), net.ParseIP("74.50.153.4")}, net.ParseIP("74.50.153.5"), false},
		{ipRange{start: net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), end: net.ParseIP("74.50.153.4")}, net.ParseIP("74.50.153.2"), false},
		{ipRange{net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:8334")}, net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), true},
		{ipRange{net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7334"), net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:8334")}, net.ParseIP("2001:0db8:85a3:0000:0000:8a2e:0370:7350"), true},
		{ipRange{net.ParseIP("::ffff:192.0.2.128"), net.ParseIP("::ffff:192.0.2.250")}, net.ParseIP("::ffff:192.0.2.127"), false},
		{ipRange{net.ParseIP("::ffff:192.0.2.128"), net.ParseIP("::ffff:192.0.2.250")}, net.ParseIP("::ffff:192.0.2.128"), true},
		{ipRange{net.ParseIP("::ffff:192.0.2.128"), net.ParseIP("::ffff:192.0.2.250")}, net.ParseIP("::ffff:192.0.2.129"), true},
		{ipRange{net.ParseIP("::ffff:192.0.2.128"), net.ParseIP("::ffff:192.0.2.250")}, net.ParseIP("::ffff:192.0.2.251"), false},
		{ipRange{net.ParseIP("::ffff:192.0.2.128"), net.ParseIP("::ffff:192.0.2.250")}, net.ParseIP("192.0.2.130"), true},
		{ipRange{net.ParseIP("192.0.2.128"), net.ParseIP("192.0.2.250")}, net.ParseIP("::ffff:192.0.2.130"), true},
		{ipRange{net.ParseIP("192.0.2.128"), net.ParseIP("192.0.2.250")}, net.ParseIP("::ffff:192.0.2.130"), true},
	}
	for _, tc := range cases {
		output := inRange(tc.input, tc.adress)
		util.Equals(t, tc.expected, output)
	}
}

func TestIsPrivateSubnet(t *testing.T) {
	cases := []struct {
		input    net.IP
		expected bool
	}{
		{net.ParseIP("10.0.0.54"), true},
		{net.ParseIP("100.64.0.1"), true},
		{net.ParseIP("172.32.45.53"), false},
		{net.ParseIP("192.0.0.0"), true},
		{net.ParseIP("192.168.0.0"), true},
		{net.ParseIP("224.43.65.67"), false},
		{net.ParseIP("192.168.17.87"), true},
	}
	for _, tc := range cases {
		output := isPrivateSubnet(tc.input)
		util.Equals(t, tc.expected, output)
	}
}

func TestGetIPAdress(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"60.30.210.210", "60.30.210.210"},
		{"100.64.0.1", ""},
		{"172.32.45.53", "172.32.45.53"},
		{"192.0.0.0", ""},
		{"192.168.0.0", ""},
		{"localhost", ""},
	}
	for _, tc := range cases {
		request, err := http.NewRequest("GET", fmt.Sprintf("http://%s:8080", tc.input), nil)
		util.OK(t, err)
		request.Header.Add("X-Real-Ip", tc.input)
		util.Equals(t, tc.expected, getIPAdress(request))
	}
}

func TestGetRecordType(t *testing.T) {
	ipV4 := "98.139.180.149"
	ipV6 := "2607:f0d0:1002:51::4"
	util.Equals(t, "A", GetRecordType(ipV4))
	util.Equals(t, "AAAA", GetRecordType(ipV6))
}
