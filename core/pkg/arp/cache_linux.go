// +build linux

package arp

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

const (
	fIPAddr int = iota
	fHWType
	fFlags
	fHWAddr
	fMask
	fDevice
)

const (
	cachefile = "/proc/net/arp"
)

func loadCache() (Caches, error) {
	f, err := ioutil.ReadFile(cachefile)
	if err != nil {
		return nil, err
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(f))
	// skip first line, it is descriptions
	scanner.Scan()

	var caches = make(Caches, 0)

	for scanner.Scan() {
		c, err := parse(scanner.Text(), ifaces)
		if err != nil {
			continue
		}
		caches = append(caches, c)
	}
	return caches, nil
}

func parse(line string, ifaces []net.Interface) (*Cache, error) {
	fields := strings.Fields(line)

	ip := net.ParseIP(fields[fIPAddr])
	if ip == nil {
		return nil, fmt.Errorf("failed to parse IP addr: %v", fields[fIPAddr])
	}
	hwType, err := strconv.ParseInt(fields[fHWType], 0, 32)
	if err != nil {
		return nil, err
	}
	flag, err := strconv.ParseInt(fields[fFlags], 0, 32)
	if err != nil {
		return nil, err
	}
	hwAddr, err := net.ParseMAC(fields[fHWAddr])
	if err != nil {
		return nil, err
	}
	d := fields[fDevice]
	var dev *net.Interface
	for _, iface := range ifaces {
		if iface.Name == d {
			dev = &iface
			break
		}
	}

	return &Cache{
		IP:           ip,
		HardwareAddr: hwAddr,
		HardwareType: hwType,
		Flags:        flag,
		Interface:    dev,
	}, nil
}
