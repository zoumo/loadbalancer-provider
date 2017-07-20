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
	"fmt"
	"net"
	"strconv"
	"time"

	netv1alpha1 "github.com/caicloud/loadbalancer-controller/pkg/apis/networking/v1alpha1"
	"github.com/caicloud/loadbalancer-controller/pkg/util/validation"
	"github.com/caicloud/loadbalancer-provider/core/pkg/arp"
	core "github.com/caicloud/loadbalancer-provider/core/provider"
	"github.com/caicloud/loadbalancer-provider/providers/ipvsdr/version"
	log "github.com/zoumo/logdog"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/flowcontrol"
	utildbus "k8s.io/kubernetes/pkg/util/dbus"
	k8sexec "k8s.io/kubernetes/pkg/util/exec"
	utiliptables "k8s.io/kubernetes/pkg/util/iptables"
	"k8s.io/kubernetes/pkg/util/sysctl"
)

var _ core.Provider = &IpvsdrProvider{}

var (
	// sysctl changes required by keepalived
	sysctlAdjustments = map[string]int{
		// allows processes to bind() to non-local IP addresses
		"net/ipv4/ip_nonlocal_bind": 1,
		// enable connection tracking for LVS connections
		"net/ipv4/vs/conntrack": 1,
		// Reply only if the target IP address is local address configured on the incoming interface.
		"net/ipv4/conf/all/arp_ignore": 1,
		// Always use the best local address for ARP requests sent on interface.
		"net/ipv4/conf/all/arp_announce": 2,
		// Reply only if the target IP address is local address configured on the incoming interface.
		"net/ipv4/conf/lo/arp_ignore": 1,
		// Always use the best local address for ARP requests sent on interface.
		"net/ipv4/conf/lo/arp_announce": 2,
	}
)

// IpvsdrProvider ...
type IpvsdrProvider struct {
	nodeInfo          *nodeInfo
	reloadRateLimiter flowcontrol.RateLimiter
	keepalived        *keepalived
	storeLister       core.StoreLister
	sysctlDefault     map[string]int
	vip               string
	cfgMD5            string
	ipt               utiliptables.Interface
	neighbors         []ipmac
}

// NewIpvsdrProvider creates a new ipvs-dr LoadBalancer Provider.
func NewIpvsdrProvider(nodeIP net.IP, lb *netv1alpha1.LoadBalancer, unicast bool) (*IpvsdrProvider, error) {
	nodeInfo, err := getNetworkInfo(nodeIP.String())
	if err != nil {
		log.Error("get node info err", log.Fields{"err": err})
		return nil, err
	}

	execer := k8sexec.New()
	dbus := utildbus.New()
	iptInterface := utiliptables.New(execer, dbus, utiliptables.ProtocolIpv4)

	ipvs := &IpvsdrProvider{
		nodeInfo:          nodeInfo,
		reloadRateLimiter: flowcontrol.NewTokenBucketRateLimiter(10.0, 10),
		vip:               lb.Spec.Providers.Ipvsdr.Vip,
		sysctlDefault:     make(map[string]int, 0),
		ipt:               iptInterface,
		neighbors:         make([]ipmac, 0),
	}

	// neighbors := getNodeNeighbors(nodeInfo, clusterNodes)
	ipvs.keepalived = &keepalived{
		nodeInfo:   nodeInfo,
		useUnicast: unicast,
		ipt:        iptInterface,
	}

	err = ipvs.keepalived.loadTemplate()
	if err != nil {
		return nil, err
	}

	return ipvs, nil
}

// OnUpdate ...
func (p *IpvsdrProvider) OnUpdate(lb *netv1alpha1.LoadBalancer) error {
	p.reloadRateLimiter.Accept()

	if err := validation.ValidateLoadBalancer(lb); err != nil {
		log.Error("invalid loadbalancer", log.Fields{"err": err})
		return nil
	}

	// filtered
	if lb.Spec.Type != netv1alpha1.LoadBalancerTypeExternal || lb.Spec.Providers.Ipvsdr == nil {
		return nil
	}

	log.Notice("Updating config")

	// get selected nodes' ip
	selectedNodes := p.getNodesIP(lb.Spec.Nodes.Names)
	if len(selectedNodes) == 0 {
		return nil
	}

	svc := virtualServer{
		VIP:        lb.Spec.Providers.Ipvsdr.Vip,
		Scheduler:  string(lb.Spec.Providers.Ipvsdr.Scheduler),
		RealServer: selectedNodes,
	}

	neighbors := p.resolveNeighbors(getNeighbors(p.nodeInfo.ip, selectedNodes))

	err := p.keepalived.UpdateConfig(
		[]virtualServer{svc},
		neighbors,
		getNodePriority(p.nodeInfo.ip, selectedNodes),
		*lb.Status.ProvidersStatuses.Ipvsdr.Vrid,
	)
	if err != nil {
		return err
	}

	// check md5
	md5, err := checksum(keepalivedCfg)
	if err == nil && md5 == p.cfgMD5 {
		return nil
	}

	p.ensureIptablesMark(neighbors)

	p.cfgMD5 = md5
	err = p.keepalived.Reload()
	if err != nil {
		log.Error("reload keepalived error", log.Fields{"err": err})
		return err
	}

	return nil
}

// Start ...
func (p *IpvsdrProvider) Start() {
	log.Info("Startting ipvs dr provider")

	p.changeSysctl()
	p.setLoopbackVIP()
	go p.keepalived.Start()
	return
}

// WaitForStart waits for ipvsdr fully run
func (p *IpvsdrProvider) WaitForStart() bool {
	err := wait.Poll(time.Second, 60*time.Second, func() (bool, error) {
		if p.keepalived.started && p.keepalived.cmd != nil && p.keepalived.cmd.Process != nil {
			return true, nil
		}
		return false, nil
	})

	if err != nil {
		return false
	}
	return true
}

// Stop ...
func (p *IpvsdrProvider) Stop() error {
	log.Info("Shutting down ipvs dr provider")

	err := p.resetSysctl()
	if err != nil {
		log.Error("reset sysctl error", log.Fields{"err": err})
	}

	err = p.removeLoopbackVIP()
	if err != nil {
		log.Error("remove loopback vip error", log.Fields{"err": err})
	}

	p.flushIptablesMark()

	p.keepalived.Stop()

	return nil
}

// Info ...
func (p *IpvsdrProvider) Info() core.Info {
	return core.Info{
		Name:       "ipvsdr",
		Release:    version.RELEASE,
		Build:      version.COMMIT,
		Repository: version.REPO,
	}
}

// SetListers sets the configured store listers in the generic ingress controller
func (p *IpvsdrProvider) SetListers(lister core.StoreLister) {
	p.storeLister = lister
}

func (p *IpvsdrProvider) getNodesIP(names []string) []string {
	ips := make([]string, 0)
	if names == nil {
		return ips
	}

	for _, name := range names {
		node, err := p.storeLister.Node.Get(name)
		if err != nil {
			continue
		}
		ip, err := GetNodeHostIP(node)
		if err != nil {
			continue
		}
		ips = append(ips, ip.String())
	}

	return ips
}

// changeSysctl changes the required network setting in /proc to get
// keepalived working in the local system.
func (p *IpvsdrProvider) changeSysctl() error {
	sys := sysctl.New()
	for k, v := range sysctlAdjustments {
		defVar, err := sys.GetSysctl(k)
		if err != nil {
			return err
		}
		p.sysctlDefault[k] = defVar

		if err := sys.SetSysctl(k, v); err != nil {
			return err
		}
	}
	return nil
}

// resetSysctl resets the network setting
func (p *IpvsdrProvider) resetSysctl() error {
	log.Info("reset sysctl to original value", log.Fields{"defaults": p.sysctlDefault})
	sys := sysctl.New()
	for k, v := range p.sysctlDefault {
		if err := sys.SetSysctl(k, v); err != nil {
			return err
		}
	}
	return nil
}

// setLoopbackVIP sets vip to dev lo
func (p *IpvsdrProvider) setLoopbackVIP() error {

	if p.vip == "" {
		return nil
	}

	lo, err := getLoopBackInfo()
	if err != nil {
		return err
	}

	out, err := k8sexec.New().Command("ip", "addr", "add", p.vip+"/32", "dev", lo.iface).CombinedOutput()
	if err != nil {
		return fmt.Errorf("set VIP %s to dev lo error: %v\n%s", p.vip, err, out)
	}
	return nil
}

// removeLoopbackVIP removes vip from dev lo
func (p *IpvsdrProvider) removeLoopbackVIP() error {
	log.Info("remove vip from dev lo", log.Fields{"vip": p.vip})

	if p.vip == "" {
		return nil
	}

	lo, err := getLoopBackInfo()
	if err != nil {
		return err
	}

	out, err := k8sexec.New().Command("ip", "addr", "del", p.vip+"/32", "dev", lo.iface).CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing configured VIP from dev lo error: %v\n%s", err, out)
	}
	return nil
}

func (p *IpvsdrProvider) resolveNeighbors(neighbors []string) []ipmac {

	resolvedNeighbors := make([]ipmac, 0)

	for _, neighbor := range neighbors {
		hwAddr, err := arp.Resolve(p.nodeInfo.iface, neighbor)
		if err != nil {
			log.Errorf("failed to resolve hardware address for %v", neighbor)
			continue
		}
		resolvedNeighbors = append(resolvedNeighbors, ipmac{IP: neighbor, MAC: hwAddr})
	}
	return resolvedNeighbors
}

func (p *IpvsdrProvider) setIptablesMark(protocol, mac string, mark int) (bool, error) {
	if mac == "" {
		return p.ipt.EnsureRule(utiliptables.Append, "mangle", "PREROUTING", "-i", p.nodeInfo.iface, "-d", p.vip, "-p", protocol, "-j", "MARK", "--set-mark", strconv.Itoa(mark))
	}
	return p.ipt.EnsureRule(utiliptables.Append, "mangle", "PREROUTING", "-i", p.nodeInfo.iface, "-d", p.vip, "-p", protocol, "-m", "mac", "--mac-source", mac, "-j", "MARK", "--set-mark", strconv.Itoa(mark))
}

func (p *IpvsdrProvider) deleteIptablesMark(protocol, mac string, mark int) error {
	if mac == "" {
		return p.ipt.DeleteRule("mangle", "PREROUTING", "-i", p.nodeInfo.iface, "-d", p.vip, "-p", protocol, "-j", "MARK", "--set-mark", strconv.Itoa(mark))
	}
	return p.ipt.DeleteRule("mangle", "PREROUTING", "-i", p.nodeInfo.iface, "-d", p.vip, "-p", protocol, "-m", "mac", "--mac-source", mac, "-j", "MARK", "--set-mark", strconv.Itoa(mark))

}

func (p *IpvsdrProvider) ensureIptablesMark(neighbors []ipmac) {
	if p.neighbors == nil {
		p.neighbors = make([]ipmac, 0)
	}

	p.flushIptablesMark()

	// this two rules should be appended firstly
	// they mark all", "tcp and ", "udp traffics with 1
	p.setIptablesMark("tcp", "", acceptMark)
	p.setIptablesMark("udp", "", acceptMark)

	// all neighbors' rules should be under the basic rule, to override it
	// make sure that all traffics which come from the neighbors will be marked with 2
	// an than lvs will ignore it
	for _, neighbor := range neighbors {
		_, err := p.setIptablesMark("tcp", neighbor.MAC.String(), dropMark)
		if err != nil {
			log.Error("failed to ensure iptables tcp rule for", log.Fields{"ip": neighbor.IP, "mac": neighbor.MAC.String(), "mark": dropMark, "err": err})
		}
		_, err = p.setIptablesMark("udp", neighbor.MAC.String(), dropMark)
		if err != nil {
			log.Error("failed to ensure iptables udp rule for", log.Fields{"ip": neighbor.IP, "mac": neighbor.MAC.String(), "mark": dropMark, "err": err})
		}
	}

	p.neighbors = neighbors

}

func (p *IpvsdrProvider) flushIptablesMark() {
	log.Info("flush all marked rules")

	p.deleteIptablesMark("tcp", "", acceptMark)
	p.deleteIptablesMark("udp", "", acceptMark)

	for _, neighbor := range p.neighbors {
		p.deleteIptablesMark("tcp", neighbor.MAC.String(), dropMark)
		p.deleteIptablesMark("udp", neighbor.MAC.String(), dropMark)
	}

}
