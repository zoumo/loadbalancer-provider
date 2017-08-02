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

package provider

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/golang/glog"
	"k8s.io/client-go/pkg/api/v1"
	k8sexec "k8s.io/kubernetes/pkg/util/exec"
)

var (
	invalidIfaces = []string{"lo", "docker0", "flannel.1", "cbr0"}
	nsSvcLbRegex  = regexp.MustCompile(`(.*)/(.*):(.*)|(.*)/(.*)`)
	vethRegex     = regexp.MustCompile(`^veth.*`)
	lvsRegex      = regexp.MustCompile(`NAT`)
)

// GetNodeHostIP returns the provided node's IP, based on the priority:
// 1. NodeExternalIP
// 2. NodeInternalIP
func GetNodeHostIP(node *v1.Node) (net.IP, error) {
	addresses := node.Status.Addresses
	addressMap := make(map[v1.NodeAddressType][]v1.NodeAddress)
	for i := range addresses {
		addressMap[addresses[i].Type] = append(addressMap[addresses[i].Type], addresses[i])
	}
	if addresses, ok := addressMap[v1.NodeExternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	if addresses, ok := addressMap[v1.NodeInternalIP]; ok {
		return net.ParseIP(addresses[0].Address), nil
	}
	return nil, fmt.Errorf("host IP unknown; known addresses: %v", addresses)
}

func appendIfMissing(slice []string, item string) []string {
	for _, elem := range slice {
		if elem == item {
			return slice
		}
	}
	return append(slice, item)
}

func resetIPVS() error {
	glog.Info("cleaning ipvs configuration")
	_, err := k8sexec.New().Command("ipvsadm", "-C").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error removing ipvs configuration: %v", err)
	}

	return nil
}

type netInterface struct {
	name    string
	ip      string
	netmask int
	mac     net.HardwareAddr
}

func getLoopBackInfo() (*netInterface, error) {

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
					info := &netInterface{
						name:    iface.Name,
						ip:      ipnet.IP.String(),
						netmask: mask,
						mac:     iface.HardwareAddr,
					}
					return info, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("Can not find loopback interface")
}

// getNetworkInfo returns information of the node where the pod is running
func getNetworkInfo(ip string) (*netInterface, error) {
	niface := interfaceByIP(ip)
	if niface.name == "" {
		return nil, fmt.Errorf("Can not find net interface for ip:%v", ip)
	}
	return &niface, nil
}

func getNeighbors(ip string, nodes []string) (neighbors []string) {
	for _, neighbor := range nodes {
		if ip != neighbor {
			neighbors = append(neighbors, neighbor)
		}
	}
	return
}

// netInterfaces returns a slice containing the local network interfaces
// excluding lo, docker0, flannel.1 and veth interfaces.
func netInterfaces() []net.Interface {
	validIfaces := []net.Interface{}
	ifaces, err := net.Interfaces()
	if err != nil {
		return validIfaces
	}

	for _, iface := range ifaces {
		if !vethRegex.MatchString(iface.Name) && stringSlice(invalidIfaces).pos(iface.Name) == -1 {
			validIfaces = append(validIfaces, iface)
		}
	}

	return validIfaces
}

// interfaceByIP returns the local network interface name that is using the
// specified IP address. If no interface is found returns an empty string.
func interfaceByIP(ip string) netInterface {
	for _, iface := range netInterfaces() {
		niface, err := ipByInterface(iface.Name)
		if err == nil && ip == niface.ip {
			return niface
		}
	}

	return netInterface{}
}

func ipByInterface(name string) (netInterface, error) {
	ret := netInterface{
		name:    name,
		ip:      "",
		netmask: 32,
		mac:     net.HardwareAddr(""),
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
				ret.ip = ipnet.IP.String()
				ones, _ := ipnet.Mask.Size()
				ret.netmask = ones
				ret.mac = iface.HardwareAddr
				return ret, nil
			}
		}
	}

	return ret, errors.New("Found no IPv4 addresses")
}

type stringSlice []string

// pos returns the position of a string in a slice.
// If it does not exists in the slice returns -1.
func (slice stringSlice) pos(value string) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}

	return -1
}

// getPriority returns the priority of one node using the
// IP address as key. It starts in 100
func getNodePriority(ip string, nodes []string) int {
	return 100 + stringSlice(nodes).pos(ip)
}

func checksum(filename string) (string, error) {
	var result []byte
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(result)), nil
}
