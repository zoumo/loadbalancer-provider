package node

import (
	"fmt"
	"net"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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

// GetNodeIPForPod returns the node ip where the pod is located
func GetNodeIPForPod(client kubernetes.Interface, podNamespace, podName string) (net.IP, error) {
	if podName == "" || podNamespace == "" {
		return nil, fmt.Errorf("Please check the manifest (for missing POD_NAME or POD_NAMESPACE env variables)")
	}

	pod, err := client.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to get pod: %s", err)
	}

	node, err := client.CoreV1().Nodes().Get(pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to get node: %s", err)
	}

	ip, err := GetNodeHostIP(node)
	if err != nil {
		return nil, err
	}

	return ip, nil
}
