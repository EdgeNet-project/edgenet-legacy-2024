package remoteip

import (
	"log"
	"net"
	"net/http"
	"testing"
)

func TestIpRange(t *testing.T) {
	var tests = []struct {
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

	for _, test := range tests {
		if output := inRange(test.input, test.adress); output != test.expected {
			t.Error("Ip ", test.adress, "doesn't belong to range")
		}
	}
}

func TestIsPrivateSubnet(t *testing.T) {
	var tests = []struct {
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
	for _, test := range tests {
		if output := isPrivateSubnet(test.input); output != test.expected {
			t.Error("Private Ip ", test.input, "doesn't belong to range")
		}
	}
}

func TestGetIPAdress(t *testing.T) {
	request, err := http.NewRequest("GET", "http://localhost:8080", nil)
	if err != nil {
		log.Fatalln(err)
	}
	if GetIPAdress(request) != "" {
		t.Errorf("Problem in GetIpAdress function")
	}
}

func TestGetRecordType(t *testing.T) {
	ipV4 := "98.139.180.149"
	ipV6 := "2607:f0d0:1002:51::4"
	if (GetRecordType(ipV4) != "A") || (GetRecordType(ipV6) != "AAAA") {
		t.Errorf("IP type detection failed")
	}
}
