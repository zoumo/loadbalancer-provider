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
	"os"
	"os/exec"
	"syscall"
	"text/template"

	log "github.com/zoumo/logdog"

	k8sexec "k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/iptables"
)

const (
	iptablesChain  = "LOADBALANCER-IPVS-DR"
	keepalivedCfg  = "/etc/keepalived/keepalived.conf"
	keepalivedTmpl = "/root/keepalived.tmpl"

	acceptMark = 1
	dropMark   = 0
	mask       = "0x00000001"
)

type ipmac struct {
	IP  string
	MAC net.HardwareAddr
}

type virtualServer struct {
	VIP        string
	Scheduler  string
	RealServer []string
}

type keepalived struct {
	started    bool
	useUnicast bool
	nodeInfo   *netInterface
	ipt        iptables.Interface
	cmd        *exec.Cmd
	tmpl       *template.Template
	vips       []string
}

// WriteCfg creates a new keepalived configuration file.
// In case of an error with the generation it returns the error
func (k *keepalived) UpdateConfig(vss []virtualServer, neighbors []ipmac, priority int, vrid int) error {
	w, err := os.Create(keepalivedCfg)
	if err != nil {
		return err
	}
	defer w.Close()

	// save vips for release when shutting down
	k.vips = getVIPs(vss)

	conf := make(map[string]interface{})
	conf["iptablesChain"] = iptablesChain
	conf["iface"] = k.nodeInfo.name
	conf["myIP"] = k.nodeInfo.ip
	conf["netmask"] = k.nodeInfo.netmask
	conf["vss"] = vss
	conf["vips"] = k.vips
	conf["neighbors"] = neighbors
	conf["priority"] = priority
	conf["useUnicast"] = k.useUnicast
	conf["vrid"] = vrid
	conf["acceptMark"] = acceptMark

	return k.tmpl.Execute(w, conf)
}

// getVIPs returns a list of the virtual IP addresses to be used in keepalived
// without duplicates (a service can use more than one port)
func getVIPs(svcs []virtualServer) []string {
	result := []string{}
	for _, svc := range svcs {
		result = appendIfMissing(result, svc.VIP)
	}

	return result
}

// Start starts a keepalived process in foreground.
// In case of any error it will terminate the execution with a fatal error
func (k *keepalived) Start() {
	ae, err := k.ipt.EnsureChain(iptables.TableFilter, iptables.Chain(iptablesChain))
	if err != nil {
		log.Fatalf("unexpected error: %v", err)
	}
	if ae {
		log.Infof("chain %v already existed", iptablesChain)
	}

	k.cmd = exec.Command("keepalived",
		"--dont-fork",
		"--log-console",
		"--release-vips",
		"--pid", "/keepalived.pid")

	k.cmd.Stdout = os.Stdout
	k.cmd.Stderr = os.Stderr

	k.started = true

	if err := k.cmd.Start(); err != nil {
		log.Errorf("keepalived error: %v", err)
	}

	if err := k.cmd.Wait(); err != nil {
		log.Fatalf("keepalived error: %v", err)
	}
}

// Reload sends SIGHUP to keepalived to reload the configuration.
func (k *keepalived) Reload() error {
	if !k.started {
		// TODO: add a warning indicating that keepalived is not started?
		log.Warn("keepalived is not started, skip the reload")
		return nil
	}

	log.Info("reloading keepalived")
	err := syscall.Kill(k.cmd.Process.Pid, syscall.SIGHUP)
	if err != nil {
		return fmt.Errorf("error reloading keepalived: %v", err)
	}

	return nil
}

// Stop stop keepalived process
func (k *keepalived) Stop() {
	for _, vip := range k.vips {
		k.removeVIP(vip)
	}

	log.Info("flush iptables chain", log.Fields{"table": iptables.TableFilter, "chain": iptablesChain})
	err := k.ipt.FlushChain(iptables.TableFilter, iptables.Chain(iptablesChain))
	if err != nil {
		log.Errorf("unexpected error flushing iptables chain %v: %v", err, iptablesChain)
	}

	log.Info("kill keepalived process", log.Fields{"pid": k.cmd.Process.Pid})
	err = syscall.Kill(k.cmd.Process.Pid, syscall.SIGTERM)
	if err != nil {
		log.Errorf("error stopping keepalived: %v", err)
	}

}

func (k *keepalived) removeVIP(vip string) error {
	log.Info("removing configured VIP %v from dev %v", vip, k.nodeInfo.name)
	out, err := k8sexec.New().Command("ip", "addr", "del", vip+"/32", "dev", k.nodeInfo.name).CombinedOutput()
	if err != nil {
		return fmt.Errorf("error reloading keepalived: %v\n%s", err, out)
	}
	return nil
}

func (k *keepalived) loadTemplate() error {
	tmpl, err := template.ParseFiles(keepalivedTmpl)
	if err != nil {
		return err
	}
	k.tmpl = tmpl
	return nil
}
