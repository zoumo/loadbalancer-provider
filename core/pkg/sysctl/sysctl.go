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
