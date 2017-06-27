package main

import (
	"fmt"
	"net"
	"os"

	"github.com/caicloud/loadbalancer-provider/providers/ipvsdr/provider"
	log "github.com/zoumo/logdog"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sexec "k8s.io/kubernetes/pkg/util/exec"
)

func getNodeIP(client kubernetes.Interface, podName, podNamespace string) (net.IP, error) {
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

	ip, err := provider.GetNodeHostIP(node)
	if err != nil {
		return nil, err
	}

	return ip, nil
}

// loadIPVSModule load module require to use keepalived
func loadIPVSModule() error {
	out, err := k8sexec.New().Command("modprobe", "ip_vs").CombinedOutput()
	if err != nil {
		log.Infof("Error loading ip_vip: %s, %v", string(out), err)
		return err
	}

	_, err = os.Stat("/proc/net/ip_vs")
	return err
}

func resetIPVS() error {
	log.Info("cleaning ipvs configuration")
	_, err := k8sexec.New().Command("ipvsadm", "-C").CombinedOutput()
	if err != nil {
		return fmt.Errorf("error removing ipvs configuration: %v", err)
	}

	return nil
}
