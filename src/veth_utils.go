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
	uuid, _ := uuid.NewRandom()

	return vethPrefix + strings.Replace(uuid.String(), "-", "", -1)[:vethLen]
}

func createVethPair(macAddress net.HardwareAddr) (string, string, error) {
	vethName1 := randomVethName()
	vethName2 := randomVethName()

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = vethName1
	linkAttrs.HardwareAddr = macAddress

	err := netlink.LinkAdd(&netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  vethName2,
	})
	if err != nil {
		return "", "", err
	}

	return vethName1, vethName2, nil
}

func deleteVethPair(vethOutside string) error {
	iface, err := netlink.LinkByName(vethOutside)
	if err != nil {
		return err
	}

	netlink.LinkDel(iface)

	return nil
}
