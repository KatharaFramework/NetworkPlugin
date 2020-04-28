package main

import (
	"net"
	"strings"

	"github.com/google/uuid"
	"github.com/vishvananda/netlink"
)

const (
	vethPrefix	= "veth"
	vethLen		= 8
)

func randomVethName() string {
	randomUuid, _ := uuid.NewRandom()

	return vethPrefix + strings.Replace(randomUuid.String(), "-", "", -1)[:vethLen]
}

func createVethPair(macAddress net.HardwareAddr) (string, string, error) {
	vethName1 := randomVethName()
	vethName2 := randomVethName()

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = vethName1
	linkAttrs.HardwareAddr = macAddress

	if err := netlink.LinkAdd(&netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  vethName2,
	}); err != nil {
		return "", "", err
	}

	return vethName1, vethName2, nil
}

func deleteVethPair(vethOutside string) error {
	iface, err := netlink.LinkByName(vethOutside)
	if err != nil {
		return err
	}

	if err := netlink.LinkDel(iface); err != nil {
		return err
	}

	return nil
}
