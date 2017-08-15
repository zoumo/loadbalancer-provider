package sysctl

import (
	"fmt"
	"io/ioutil"
	"path"
	"runtime"
	"strings"
)

const (
	sysctlBase = "/proc/sys"
)

// Interface is An injectable interface for running sysctl commands.
type Interface interface {
	// GetSysctl returns the value for the specified sysctl setting
	GetSysctl(sysctl string) (string, error)
	// SetSysctl modifies the specified sysctl flag to the new value
	SetSysctl(sysctl string, newVal string) error
}

// New returns a new Interface for accessing sysctl
func New() Interface {
	return &procSysctl{}
}

// procSysctl implements Interface by reading and writing files under /proc/sys
type procSysctl struct {
}

// GetSysctl returns the value for the specified sysctl setting
// sysctl could be setting name or path
// name: net.ipv4.ip_nonlocal_bind
// path: net/ipv4/ip_nonlocal_bind
func (s *procSysctl) GetSysctl(sysctl string) (string, error) {
	if runtime.GOOS != "linux" {
		return "", fmt.Errorf("not support on os: %s", runtime.GOOS)
	}
	data, err := ioutil.ReadFile(sysctlPath(sysctl))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// SetSysctl modifies the specified sysctl flag to the new value
func (s *procSysctl) SetSysctl(sysctl string, newVal string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("not support on os: %s", runtime.GOOS)
	}
	return ioutil.WriteFile(sysctlPath(sysctl), []byte(newVal), 0644)
}

func sysctlPath(name string) string {
	return path.Join(sysctlBase, strings.Replace(name, ".", "/", -1))
}
