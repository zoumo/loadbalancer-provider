// +build darwin

package arp

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"strings"

	k8sexec "k8s.io/kubernetes/pkg/util/exec"
)

const (
	fIPAddr int = iota
	fHWAddr
	fExpireO
	fExpirtI
	fNetif
	fRefs
	fPrbs
)

func loadCache() (Caches, error) {
	output, err := k8sexec.New().Command("arp", "-anl").CombinedOutput()
	if err != nil {
		return nil, err
	}
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(bytes.NewReader(output))
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
	hwAddr, err := net.ParseMAC(fields[fHWAddr])
	if err != nil {
		return nil, err
	}
	d := fields[fNetif]
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
		Interface:    dev,
	}, nil
}
