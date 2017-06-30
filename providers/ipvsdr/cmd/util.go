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
