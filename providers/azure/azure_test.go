package azure

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/caicloud/clientset/kubernetes"
	lbapi "github.com/caicloud/clientset/pkg/apis/loadbalance/v1alpha2"
	"github.com/caicloud/loadbalancer-provider/providers/azure/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	lbName          = "lb-lbtest-14f93ef434dc35225fa833674749ab98"
	lbResourceGroup = "loadbalancer-test"
)

func getDefaultClient() (*client.Client, error) {
	gopath := os.Getenv("GOPATH")
	gopaths := strings.Split(gopath, ":")
	if len(gopaths) == 0 {
		return nil, fmt.Errorf("get env GOPATH err")
	}
	configFile := gopaths[0] + "/azure.json"
	configBytes, err := ioutil.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	azConfig := &client.Config{}
	err = json.Unmarshal(configBytes, azConfig)
	if err != nil {
		return nil, err
	}

	return client.NewClientWithConfig(azConfig)
}

func TestCreateAzureLoadBalancer_PublicAddress(t *testing.T) {
	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	lb := &lbapi.LoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "lbtest",
		},
		Spec: lbapi.LoadBalancerSpec{
			Providers: lbapi.ProvidersSpec{
				Azure: &lbapi.AzureProvider{
					ResourceGroupName: lbResourceGroup,
					Location:          "chinaeast",
					SKU:               "Standard",
					ClusterID:         "tail",
					IPAddressProperties: lbapi.AzureIPAddressProperties{
						Public: &lbapi.AzurePublicIPAddressProperties{
							PublicIPAddressID: to.StringPtr("/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/loadbalancer-test/providers/Microsoft.Network/publicIPAddresses/public-ip"),
						},
					},
				},
			},
		},
	}

	// test create loadbalancer
	_, err = createAzureLoadBalancer(azClient, lb)
	if err != nil {
		t.Fatalf("createAzureLoadBalancer failed %v", err)
	}
}

func TestCreateAzureLoadBalancer_PrivateAddress(t *testing.T) {
	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	lb := &lbapi.LoadBalancer{
		ObjectMeta: metav1.ObjectMeta{
			Name: "lbtest",
		},
		Spec: lbapi.LoadBalancerSpec{
			Providers: lbapi.ProvidersSpec{
				Azure: &lbapi.AzureProvider{
					ResourceGroupName: "ca-demo",
					Location:          "chinaeast",
					SKU:               "Standard",
					ClusterID:         "hello",
					IPAddressProperties: lbapi.AzureIPAddressProperties{
						// Public: &lbapi.PublicAzureIPAddressProperties{
						// 	PublicIPAddressID: to.StringPtr("/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/loadbalancer-test/providers/Microsoft.Network/publicIPAddresses/public-ip"),
						// },
						Private: &lbapi.AzurePrivateIPAddressProperties{
							IPAllocationMethod: lbapi.AzureStaticIPAllocationMethod,
							SubnetID:           "/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/ca-demo/providers/Microsoft.Network/virtualNetworks/lb-test-zls/subnets/default",
							VPCID:              "/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/loadbalancer-test/providers/Microsoft.Network/virtualNetworks/loadbalancer-test-vnet",
							PrivateIPAddress:   to.StringPtr("172.22.0.4"),
						},
					},
				},
			},
		},
	}

	_, err = createAzureLoadBalancer(azClient, lb)
	if err != nil {
		t.Fatalf("createAzureLoadBalancer failed %v", err)
	}
}

func TestAttachBackendPool(t *testing.T) {

	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	azlb, err := azClient.LoadBalancer.Get(context.TODO(), lbResourceGroup, lbName, "")
	if err != nil {
		t.Fatalf("get LoadBalancer failed : %v", err)
	}
	// test attach backEnd
	networkInterfaces := []string{
		"/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/loadbalancer-test/providers/Microsoft.Network/networkInterfaces/load-privatenet540",
	}
	pool := (*azlb.BackendAddressPools)[0]
	for _, networkInterface := range networkInterfaces {
		t.Logf("start attach network %s to pool %s\n", networkInterface, to.String(pool.ID))
		err = attachNetworkInterfacesAndLoadBalancer(azClient, networkInterface, to.String(pool.ID))
		if err != nil {
			t.Fatalf("attach network %s to pool %s failed %v\n", networkInterface, to.String(pool.ID), err)
		}
		t.Logf("attach network %s to pool %s success\n", networkInterface, to.String(pool.ID))
	}
	t.Logf("OK")
}

func TestDetachBackendPool(t *testing.T) {

	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	azlb, err := azClient.LoadBalancer.Get(context.TODO(), lbResourceGroup, lbName, "")
	if err != nil {
		t.Fatalf("get LoadBalancer failed : %v", err)
	}
	// test detach backEnd
	networkInterfaces := []string{
		"/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/loadbalancer-test/providers/Microsoft.Network/networkInterfaces/load-privatenet540",
	}
	pool := (*azlb.BackendAddressPools)[0]
	for _, networkInterface := range networkInterfaces {
		t.Logf("start detach network %s to pool %s\n", networkInterface, to.String(pool.ID))
		err = detachNetworkInterfacesAndLoadBalancer(azClient, networkInterface, to.String(pool.ID))
		if err != nil {
			t.Fatalf("detach network %s to pool %s failed %v\n", networkInterface, to.String(pool.ID), err)
		}
		t.Logf("detach network %s to pool %s success\n", networkInterface, to.String(pool.ID))
	}
	t.Logf("OK")
}

func TestSyncLoadBalancerRules(t *testing.T) {
	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	azlb, err := azClient.LoadBalancer.Get(context.TODO(), lbResourceGroup, lbName, "")
	if err != nil {
		t.Fatalf("get LoadBalancer failed : %v", err)
	}

	for _, probe := range *azlb.Probes {
		t.Logf("port %d\n", *probe.Port)
	}
	m1 := map[string]string{
		"6060": "default/test1:6060",
	}
	err = syncRules(azClient, &azlb, m1, nil, lbResourceGroup)
	if err != nil {
		t.Errorf("syncRule failed : %v", err)
	}
	m2 := map[string]string{
		"6061": "default/test2:6061",
		"6060": "default/test1:6060",
	}
	err = syncRules(azClient, &azlb, m2, nil, lbResourceGroup)
	if err != nil {
		t.Errorf("syncRule failed : %v", err)
	}

	m3 := map[string]string{
		"6060": "default/test1:6060",
		"6063": "default/test3:6063",
	}
	err = syncRules(azClient, &azlb, m3, nil, lbResourceGroup)
	if err != nil {
		t.Errorf("syncRule failed : %v", err)
	}
}

func TestDeleteAzureLB(t *testing.T) {
	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	err = azClient.LoadBalancer.Delete(context.TODO(), "ca-demo", "lbte")
	if err != nil {
		t.Fatalf("createAzureLoadBalancer failed %v", err)
	}
}

func Test_GetVmNetworkInterfaces(t *testing.T) {
	azClient, err := getDefaultClient()
	if err != nil {
		t.Fatalf("getDefaultClient failed : %v", err)
	}
	nets, err := getNetworkInterfacesFromVM(azClient, "/subscriptions/5d526643-5e89-4956-bbb8-9ad896af0b08/resourceGroups/cps-resource-dev-test/providers/Microsoft.Compute/virtualMachines/compass-vm-pjb3ua-1-bqbpt1n7n4fo-zpbvc")
	t.Logf("nets %v", nets)
	if err != nil {
		t.Fatalf("getNetworkInterfacesFromVM failed %v", err)
	}
}

func TestKubeClient(t *testing.T) {
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		t.Fatalf("BuildConfigFromFlags failed : %v ", err)
	}
	// create clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatalf("Create kubernetes client error:%v", err)
	}
	lb, err := clientset.LoadbalanceV1alpha2().LoadBalancers("kube-system").Get("sda", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("err %v", err)
	}
	provider := &AzureProvider{
		clientset: clientset,
	}
	lb, err = provider.patchLoadBalancerAzureStatus(lb, lbapi.AzureRunningPhase, nil)
	if err != nil {
		t.Fatalf("patch failed %s\n", err)
	}
}
