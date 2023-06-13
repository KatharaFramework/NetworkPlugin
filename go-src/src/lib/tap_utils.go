package katnplib

import (
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
)

const (
	tapPrefix	= "tap"
	tapLen		= 8
)

func randomTapName() string {
	randomUuid, _ := uuid.NewRandom()

	return tapPrefix + strings.Replace(randomUuid.String(), "-", "", -1)[:tapLen]
}

func CreateTap(macAddress net.HardwareAddr) (string, error) {
	tapName := randomTapName()
	
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = tapName
	linkAttrs.HardwareAddr = macAddress

	if err := netlink.LinkAdd(&netlink.Tuntap{
		LinkAttrs: linkAttrs,
		Flags: netlink.TUNTAP_NO_PI,
		Mode: netlink.TUNTAP_MODE_TAP,
	}); err != nil {
		return "", err
	}

	return tapName, nil
}

func DeleteTap(tapIface string) error {
	iface, err := netlink.LinkByName(tapIface)
	if err != nil {
		return err
	}

	if err := netlink.LinkDel(iface); err != nil {
		return err
	}

	return nil
}
