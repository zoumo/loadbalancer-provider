package provider

import "k8s.io/client-go/pkg/api/v1"

var (
	// ReservedTCPPorts represents the reserved tcp ports
	ReservedTCPPorts = []string{"80", "443", "18080", "8181", "8282"}
	// ReservedUDPPorts represents the reserved udp ports
	ReservedUDPPorts = []string{}
)

// GetExportedPorts get exported ports from tcp and udp ConfigMap
func GetExportedPorts(tcpcm, udpcm *v1.ConfigMap) ([]string, []string) {
	tcpPorts := make([]string, 0)
	udpPorts := make([]string, 0)
	tcpPorts = append(tcpPorts, ReservedTCPPorts...)
	udpPorts = append(udpPorts, ReservedUDPPorts...)

	for port := range tcpcm.Data {
		tcpPorts = append(tcpPorts, port)
	}
	for port := range udpcm.Data {
		udpPorts = append(udpPorts, port)
	}
	return tcpPorts, udpPorts

}
