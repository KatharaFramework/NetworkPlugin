package katnplib

import (
	"reflect"
	"testing"
)

func TestParseSwitchMode(t *testing.T) {
	cases := map[string]SwitchMode{
		"":		SwitchModeHub,
		"hub":		SwitchModeHub,
		"switch":	SwitchModeSwitch,
		"Managed":	SwitchModeManaged,
		" managed ":	SwitchModeManaged,
	}
	for value, expected := range cases {
		mode, err := ParseSwitchMode(value)
		if err != nil || mode != expected {
			t.Errorf("ParseSwitchMode(%q) = %q, %v; expected %q", value, mode, err, expected)
		}
	}

	if _, err := ParseSwitchMode("router"); err == nil {
		t.Errorf("ParseSwitchMode(\"router\") must fail")
	}
}

func TestSwitchArgs(t *testing.T) {
	cases := map[SwitchMode][]string{
		SwitchModeHub:		{"-n", "65535", "-x", "-d", "-s", "ctl", "-p", "pid"},
		SwitchModeSwitch:	{"-n", "65535", "-d", "-s", "ctl", "-p", "pid"},
		SwitchModeManaged:	{"-n", "65535", "-d", "-s", "ctl", "-p", "pid", "-M", "mgmt"},
	}
	for mode, expected := range cases {
		if args := switchArgs(mode, "ctl", "pid", "mgmt"); !reflect.DeepEqual(args, expected) {
			t.Errorf("switchArgs(%q) = %v; expected %v", mode, args, expected)
		}
	}
}

func TestParsePortVlans(t *testing.T) {
	vlans, err := ParsePortVlans("10", "30, 20,30")
	if err != nil || !vlans.Equal(PortVlans{Untagged: 10, Tagged: []int{20, 30}}) {
		t.Errorf("ParsePortVlans = %v, %v", vlans, err)
	}

	vlans, err = ParsePortVlans("", "40 30")
	if err != nil || !vlans.Equal(PortVlans{Tagged: []int{30, 40}}) {
		t.Errorf("ParsePortVlans with spaces = %v, %v", vlans, err)
	}

	vlans, err = ParsePortVlans("", "")
	if err != nil || !vlans.Equal(PortVlans{}) {
		t.Errorf("ParsePortVlans without VLAN = %v, %v", vlans, err)
	}

	for _, invalid := range [][2]string{{"0", ""}, {"4095", ""}, {"abc", ""}, {"", "10,x"}, {"10", "10"}} {
		if _, err := ParsePortVlans(invalid[0], invalid[1]); err == nil {
			t.Errorf("ParsePortVlans(%q, %q) must fail", invalid[0], invalid[1])
		}
	}
}

func TestParseMgmtReply(t *testing.T) {
	code, message, lines, err := parseMgmtReply("0000 DATA END WITH '.'\nPort 0003\n.\n1000 Success\n")
	if err != nil || code != 1000 || message != "Success" || !reflect.DeepEqual(lines, []string{"Port 0003"}) {
		t.Errorf("parseMgmtReply = %d, %q, %v, %v", code, message, lines, err)
	}

	port, err := parseCreatedPort(lines)
	if err != nil || port != 3 {
		t.Errorf("parseCreatedPort = %d, %v", port, err)
	}

	code, message, lines, err = parseMgmtReply("1017 File exists\n")
	if err != nil || code != 1017 || message != "File exists" || len(lines) != 0 {
		t.Errorf("parseMgmtReply = %d, %q, %v, %v", code, message, lines, err)
	}

	if _, _, _, err = parseMgmtReply("garbage\n"); err == nil {
		t.Errorf("parseMgmtReply without status must fail")
	}
}
