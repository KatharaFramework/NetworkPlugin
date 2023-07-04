package katnplib

import (
	"fmt"
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
	"github.com/containernetworking/plugins/pkg/ns"
)

const (
	tapPrefix	= "tap"
	tapLen		= 8
)

func randomTapName() string {
	randomUuid, _ := uuid.NewRandom()

	return tapPrefix + strings.Replace(randomUuid.String(), "-", "", -1)[:tapLen]
}

func CreateTap(macAddress net.HardwareAddr) (string, int, error) {
	tapName := randomTapName()
	
	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = tapName
	linkAttrs.HardwareAddr = macAddress

	if err := netlink.LinkAdd(&netlink.Tuntap{
		LinkAttrs: linkAttrs,
		Flags: netlink.TUNTAP_NO_PI,
		Mode: netlink.TUNTAP_MODE_TAP,
	}); err != nil {
		return "", -1, err
	}

	iface, err := netlink.LinkByName(tapName)
	if err != nil {
		return "", -1, fmt.Errorf("failed to lookup %q: %v", tapName, err)
	}

	return tapName, iface.Attrs().Index, nil
}

func DeleteTap(tapIface string, tapIfaceIdx int, nsPath string) error {
	netns, err := ns.GetNS(nsPath)
	if err != nil {
		return fmt.Errorf("failed to open netns %q: %v", nsPath, err)
	}
	defer netns.Close()

	err = netns.Do(func(hostNS ns.NetNS) error {
		iface, err := netlink.LinkByIndex(tapIfaceIdx)
		if err != nil {
			return fmt.Errorf("failed to lookup %q in %q: %v", tapIface, hostNS.Path(), err)
		}

		if err := netlink.LinkDel(iface); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
