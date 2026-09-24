package main

import (
	"errors"
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

func createVethPair() (string, string, error) {
	vethName1 := randomVethName()
	vethName2 := randomVethName()

	linkAttrs := netlink.NewLinkAttrs()
	linkAttrs.Name = vethName1

	if err := netlink.LinkAdd(&netlink.Veth{
		LinkAttrs: linkAttrs,
		PeerName:  vethName2,
	}); err != nil {
		return "", "", err
	}

	return vethName1, vethName2, nil
}

func vethExists(vethName string) bool {
	_, err := netlink.LinkByName(vethName)

	return err == nil
}

func assignVethMac(vethName string, macAddress net.HardwareAddr) error {
	if len(macAddress) == 0 {
		return nil
	}

	iface, err := netlink.LinkByName(vethName)
	if err != nil {
		return err
	}

	return netlink.LinkSetHardwareAddr(iface, macAddress)
}

/* Deleting one end of a veth pair also deletes its peer. Missing interfaces are ignored. */
func deleteVethPair(vethNames ...string) error {
	for _, vethName := range vethNames {
		if vethName == "" {
			continue
		}

		iface, err := netlink.LinkByName(vethName)
		if err != nil {
			var notFound netlink.LinkNotFoundError
			if errors.As(err, &notFound) {
				continue
			}
			return err
		}

		return netlink.LinkDel(iface)
	}

	return nil
}
