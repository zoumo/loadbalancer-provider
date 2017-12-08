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

package ipvsdr

import (
	"html/template"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	tmpl, _ := template.ParseFiles("../../build/ipvsdr/keepalived.tmpl")
	conf := make(map[string]interface{})

	conf["iptablesChain"] = iptablesChain
	conf["iface"] = "en0"
	conf["myIP"] = "192.168.1.1"
	conf["netmask"] = "255.255.255.255"
	conf["vss"] = []virtualServer{
		{
			VIP:       "192.168.99.200",
			Scheduler: "rr",
			RealServer: []string{
				"192.168.1.1",
				"192.168.1.2",
			},
		},
	}
	conf["vips"] = []string{"192.168.99.200"}
	conf["neighbors"] = []ipmac{{IP: "192.168.1.2"}}
	conf["priority"] = 100
	conf["useUnicast"] = true
	conf["vrid"] = 100
	conf["acceptMark"] = acceptMark
	assert.Nil(t, tmpl.Execute(ioutil.Discard, conf))
}
