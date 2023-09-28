package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"crypto/md5"
)

var (
	XTABLES_LOCK_PATH = "/run/xtables.lock"
	IPTABLES_PATH = "/sbin/iptables"
	IP6TABLES_PATH = "/sbin/ip6tables"
	NFT_SUFFIX = "-nft"
	LEGACY_SUFFIX = "-legacy"
)

func detectIpTables() error {
	useNft := false

	stat, err := os.Stat(XTABLES_LOCK_PATH)
	if err != nil {
		if os.IsNotExist(err) {
			useNft = true
		} else {
			return err
		}
	}

	if stat.IsDir() {
		useNft = true
	}

	ipTablesVersion := ""
	ip6TablesVersion := ""
	if useNft {
		ipTablesVersion = IPTABLES_PATH + NFT_SUFFIX
		ip6TablesVersion = IP6TABLES_PATH + NFT_SUFFIX
	} else {
		ipTablesVersion = IPTABLES_PATH + LEGACY_SUFFIX
		ip6TablesVersion = IP6TABLES_PATH + LEGACY_SUFFIX
	}

	_, err = exec.Command("update-alternatives", "--set", "iptables", ipTablesVersion).CombinedOutput()
	if err != nil {
		return err
	}
	_, err = exec.Command("update-alternatives", "--set", "ip6tables", ip6TablesVersion).CombinedOutput()
	return err
}

func generateMacAddressFromID(macAddressID string) string {
	// Generate an hash from the previous string and truncate it to 6 bytes (48 bits = MAC Length)
	hasher := md5.New()
	hasher.Write([]byte(macAddressID))
	macAddressBytes := hasher.Sum(nil)[:6]

	// Convert the byte array into an hex encoded string separated by `:`
	// This will be the MAC Address of the interface
	macAddressString := []string{}

	for _, element := range macAddressBytes {
		macAddressString = append(macAddressString, fmt.Sprintf("%02x", element))
	}

	// Steps to obtain a locally administered unicast MAC
	// See http://www.noah.org/wiki/MAC_address
	firstByteInt, _ := strconv.ParseInt(macAddressString[0], 16, 32)
	macAddressString[0] = fmt.Sprintf("%02x", (firstByteInt|0x02)&0xfe)

	return strings.Join(macAddressString, ":")
}
