package provider

import (
	"html/template"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	tmpl, _ := template.ParseFiles("../keepalived.tmpl")
	conf := make(map[string]interface{})

	conf["iptablesChain"] = iptablesChain
	conf["iface"] = "en0"
	conf["myIP"] = "127.0.0.1"
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
	conf["neighbors"] = []string{"192.168.1.1", "192.168.1.2"}
	conf["priority"] = 100
	conf["useUnicast"] = false
	conf["vrid"] = 100
	conf["acceptMark"] = acceptMark

	assert.Nil(t, tmpl.Execute(ioutil.Discard, conf))
}
