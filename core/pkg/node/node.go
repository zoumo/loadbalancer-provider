package node

import (
	"fmt"
	"net"

	"k8s.io/client-go/pkg/api/v1"
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
