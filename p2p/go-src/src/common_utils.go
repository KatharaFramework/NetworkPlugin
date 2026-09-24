package main

import (
	"fmt"
	"strconv"
	"strings"

	"crypto/md5"
)

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
