package arp

import (
	"fmt"
	"net"

	arpClient "github.com/mdlayher/arp"
)

var caches Caches

// Caches represents a list of ARP caches.
type Caches []*Cache

// Cache represents an entry in the ARP cache.
type Cache struct {
	IP           net.IP
	HardwareAddr net.HardwareAddr
	HardwareType int64
	Interface    *net.Interface
	Flags        int64
}

// Resolve resolves the hardware address of the given ip on the net interface
// 1. it try to get hardware address from local ARP cache
// 2. If the hardware address is not in the cache, then It performs an ARP request,
// attempting to retrieve the hardware address of  the machine using its IPv4 address
// through the given net interface
func Resolve(iface, ip string) (net.HardwareAddr, error) {

	refreshCache()

	dev, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return nil, fmt.Errorf("failed to parse ip addr: %v", ipAddr)
	}

	hwaddr, ok := caches.resolve(iface, ip)
	if ok {
		return hwaddr, nil
	}

	client, err := arpClient.Dial(dev)
	if err != nil {
		return nil, err
	}

	return client.Resolve(ipAddr)
}

func refreshCache() {
	var err error
	caches, err = loadCache()
	if err != nil {
		caches = make(Caches, 0)
	}
}

func (c Caches) resolve(iface, ip string) (net.HardwareAddr, bool) {
	if len(c) == 0 {
		return nil, false
	}

	for _, cache := range c {
		if cache.Interface.Name == iface && cache.IP.String() == ip {
			return cache.HardwareAddr, true
		}
	}
	return nil, false
}
