/*
Copyright 2017 Caicloud authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package net

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
)

var (
	invalidIfaces = []string{"lo", "docker0", "flannel.1", "cbr0"}
	nsSvcLbRegex  = regexp.MustCompile(`(.*)/(.*):(.*)|(.*)/(.*)`)
	vethRegex     = regexp.MustCompile(`^veth.*`)
	lvsRegex      = regexp.MustCompile(`NAT`)
)

// Interface represents the local network interface
type Interface struct {
	Name    string
	IP      string
	Netmask int
	MAC     net.HardwareAddr
}

// interfaces returns a slice containing the local network interfaces
// excluding lo, docker0, flannel.1 and veth interfaces.
func interfaces() []net.Interface {
	validIfaces := []net.Interface{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return validIfaces
	}

	for _, iface := range ifaces {
		if !vethRegex.MatchString(iface.Name) && !StringInSlice(iface.Name, invalidIfaces) {
			validIfaces = append(validIfaces, iface)
		}
	}

	return validIfaces
}

// InterfaceByLoopback returns the loopback interface
func InterfaceByLoopback() (*Interface, error) {

	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if strings.HasPrefix(iface.Name, "lo") {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, a := range addrs {
				// get loopback
				if ipnet, ok := a.(*net.IPNet); ok && ipnet.IP.IsLoopback() {
					mask, _ := ipnet.Mask.Size()
					info := &Interface{
						Name:    iface.Name,
						IP:      ipnet.IP.String(),
						Netmask: mask,
						MAC:     iface.HardwareAddr,
					}
					return info, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Can not find loopback interface")
}

// InterfaceByIP returns the local network interface that is using the
// specified IP address.
func InterfaceByIP(ip string) (*Interface, error) {
	for _, iface := range interfaces() {
		niface, err := InterfaceByName(iface.Name)
		if err == nil && ip == niface.IP {
			return &niface, nil
		}
	}
	return nil, fmt.Errorf("Can not find net interface for ip:%v", ip)
}

// InterfaceByName returns the local network interface specified by name.
func InterfaceByName(name string) (Interface, error) {
	ret := Interface{
		Name:    name,
		IP:      "",
		Netmask: 32,
		MAC:     net.HardwareAddr(""),
	}

	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ret, err
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return ret, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ret.IP = ipnet.IP.String()
				ones, _ := ipnet.Mask.Size()
				ret.Netmask = ones
				ret.MAC = iface.HardwareAddr
				return ret, nil
			}
		}
	}

	return ret, errors.New("Found no IPv4 addresses")
}

// StringInSlice check whether string a is a member of slice.
func StringInSlice(a string, slice []string) bool {
	for _, b := range slice {
		if b == a {
			return true
		}
	}
	return false
}
