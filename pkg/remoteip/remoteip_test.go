package remoteip

import (
	"fmt"
	"math/rand"
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

/*
   --- FAIL: FuzzIpRange (0.00s)
       remoteip_test.go:126: ip range is {190.9.7.16, 190.9.7.16}, ipAddress is 190.9.7.16
       remoteip_test.go:126: ip range is {190.9.7.16, 190.9.7.16}, ipAddress is 144.9.7.16
*/
func FuzzIpRange(f *testing.F) {
	a1 := uint8(192)
	b1 := uint8(168)
	c1 := uint8(17)
	d1 := uint8(0)
	a2 := uint8(195)
	b2 := uint8(111)
	c2 := uint8(19)
	d2 := uint8(3)
	x4 := uint8(193)
	getOrder := func(a, b uint8) (int, int) {
		if a < b {
			return int(a), int(b)
		} else {
			return int(b), int(a)
		}
	}
	f.Add(a1, b1, c1, d1, a2, b2, c2, d2, x4)
	f.Fuzz(func(t *testing.T, a1, b1, c1, d1, a2, b2, c2, d2, x4 uint8) {
		x1, x2 := getOrder(a1, a2)
		y1, y2 := getOrder(b1, b2)
		z1, z2 := getOrder(c1, c2)
		n1, n2 := getOrder(d1, d2)
		start := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", x1, y1, z1, n1))
		end := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", x2, y2, z2, n2))
		var x3, y3, z3, n3 int
		if x2 > x1 {
			x3 = rand.Intn(x2-x1) + x1
		} else {
			x3 = x1
		}
		if y2 > y1 {
			y3 = rand.Intn(y2-y1) + y1
		} else {
			y3 = y1
		}
		if z2 > z1 {
			z3 = rand.Intn(z2-z1) + z1
		} else {
			z3 = z1
		}
		if n2 > n1 {
			n3 = rand.Intn(n2-n1) + n1
		} else {
			n3 = n1
		}
		ip1 := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", x3, y3, z3, n3))
		ip2 := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", int(x4), y3, z3, n3))
		cases := []struct {
			input    ipRange
			adress   net.IP
			expected bool
		}{
			{ipRange{start, end},
				ip1,
				true},
			{ipRange{start, end},
				ip2,
				inBetween(x4, uint8(x1), uint8(x2))},
		}
		for _, tc := range cases {
			output := inRange(tc.input, tc.adress)
			util.Equals(t, tc.expected, output)
			t.Logf("ip range is {%v, %v}, ipAddress is %v", tc.input.start, tc.input.end, tc.adress)
		}
	})
}

// TODO: wierd, run it failed, re-run success
func FuzzIsPrivateSubnet(f *testing.F) {
	a := uint8(192)
	b := uint8(168)
	c := uint8(17)
	d := uint8(87)
	expected := false
	f.Add(a, b, c, d)
	f.Fuzz(func(t *testing.T, a, b, c, d uint8) {
		ipv4 := net.ParseIP(fmt.Sprintf("%d.%d.%d.%d", a, b, c, d))
		t.Logf("%d.%d.%d.%d", a, b, c, d)
		switch a {
		case 10:
			expected = true
		case 100:
			if inBetween(b, 64, 127) {
				expected = true
			} else {
				expected = false
			}
		case 172:
			if inBetween(b, 16, 31) {
				expected = true
			} else {
				expected = false
			}
		case 192:
			if b == 0 && c == 0 || b == 168 {
				expected = true
			} else {
				expected = false
			}
		case 198:
			if inBetween(b, 18, 19) {
				expected = true
			} else {
				expected = false
			}
		default:
			expected = false
		}
		output := isPrivateSubnet(ipv4)
		util.Equals(t, expected, output)
	})
}

func inBetween(i, min, max uint8) bool {
	if (i >= min) && (i <= max) {
		return true
	} else {
		return false
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
