package azure

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/zoumo/logdog"

	lbapi "github.com/caicloud/clientset/pkg/apis/loadbalance/v1alpha2"
	core "github.com/caicloud/loadbalancer-provider/core/provider"
	"github.com/caicloud/loadbalancer-provider/providers/azure/client"
)

// it will delete default info when recover default azlb
// make sure config the correct default info
func ensureSyncDefaultAzureLBConfig(c *client.Client, azlb *network.LoadBalancer, lb *lbapi.LoadBalancer) (*network.LoadBalancer, error) {
	var err error
	azlb, change := ensureSyncDefaultConfigExceptRules(azlb, lb)
	if change {
		// if probe frontend or backend changed, must clean rules because of rules will invoke
		// the resources reference above all, but the resources can be deleted
		if *azlb.LoadBalancingRules != nil && len(*azlb.LoadBalancingRules) != 0 {
			*azlb.LoadBalancingRules = (*azlb.LoadBalancingRules)[:0]
		}
		*azlb, err = c.LoadBalancer.CreateOrUpdate(context.TODO(), lb.Spec.Providers.Azure.ResourceGroupName, to.String(azlb.Name), *azlb)
		if err != nil {
			log.Errorf("update lb failed error : %v", err)
			return nil, err
		}
	}
	azlb, err = ensureSyncDefaultRules(c, azlb, lb)
	if err != nil {
		return nil, err
	}
	return azlb, nil
}

// getProbesBrief: get brief info from azure api
func getProbesBrief(probes *[]network.Probe) *[]network.Probe {
	if probes == nil || len(*probes) == 0 {
		return nil
	}
	brief := make([]network.Probe, 0, len(*probes))
	for _, probe := range *probes {
		brief = append(brief, network.Probe{
			Name: probe.Name,
			ProbePropertiesFormat: &network.ProbePropertiesFormat{
				Protocol:          probe.Protocol,
				Port:              probe.Port,
				IntervalInSeconds: probe.IntervalInSeconds,
				NumberOfProbes:    probe.NumberOfProbes,
			},
		})
	}
	return &brief
}

// get Azure LoadBalancer FrontendIPConfig from properties
func getAzureLoadBalancerFrontendIPConfigByConfig(properties lbapi.AzureIPAddressProperties) *[]network.FrontendIPConfiguration {
	frontendIPConfigurations := make([]network.FrontendIPConfiguration, 1)
	frontendIPConfiguration := network.FrontendIPConfiguration{
		Name: to.StringPtr(azureLoadBalancerFrontendName),
		FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{},
	}
	// make up front end private address config
	if properties.Private != nil {
		private := properties.Private
		frontendIPConfiguration.FrontendIPConfigurationPropertiesFormat.PrivateIPAllocationMethod = network.IPAllocationMethod(private.IPAllocationMethod)
		frontendIPConfiguration.FrontendIPConfigurationPropertiesFormat.PrivateIPAddress = private.PrivateIPAddress
		frontendIPConfiguration.FrontendIPConfigurationPropertiesFormat.Subnet = &network.Subnet{
			ID: to.StringPtr(private.SubnetID),
		}
	}
	// make up front end public address config
	if properties.Public != nil {
		public := properties.Public
		frontendIPConfiguration.FrontendIPConfigurationPropertiesFormat.PublicIPAddress = &network.PublicIPAddress{
			ID: public.PublicIPAddressID,
		}
	}
	frontendIPConfigurations[0] = frontendIPConfiguration
	return &frontendIPConfigurations
}

func getAzureLoadBalancerFrontendIPConfigBrief(ipConfigs *[]network.FrontendIPConfiguration) *[]network.FrontendIPConfiguration {
	if ipConfigs == nil || len(*ipConfigs) == 0 {
		return nil
	}
	brief := make([]network.FrontendIPConfiguration, 0, len(*ipConfigs))
	for _, config := range *ipConfigs {
		format := config.FrontendIPConfigurationPropertiesFormat
		if format != nil {
			brief = append(brief, network.FrontendIPConfiguration{
				FrontendIPConfigurationPropertiesFormat: &network.FrontendIPConfigurationPropertiesFormat{
					PrivateIPAllocationMethod: format.PrivateIPAllocationMethod,
					PrivateIPAddress:          format.PrivateIPAddress,
					Subnet:                    format.Subnet,
					PublicIPAddress:           format.PublicIPAddress,
				},
			})
		}
	}
	return &brief
}

// check probe backend and frontend in same config except chore for example Etag、ProvisioningState
func ensureSyncDefaultConfigExceptRules(azlb *network.LoadBalancer, lb *lbapi.LoadBalancer) (*network.LoadBalancer, bool) {
	var change bool
	// 1.ensure probe
	probes := defaultClusterAzureLBProbes()
	probesBrief := getProbesBrief(azlb.Probes)
	if !reflect.DeepEqual(probesBrief, &probes) {
		azlb.Probes = &probes
		change = true
	}

	// 2.check backend
	if azlb.BackendAddressPools == nil || len(*azlb.BackendAddressPools) == 0 {
		azlb.BackendAddressPools = &[]network.BackendAddressPool{
			{
				Name: to.StringPtr(azureLoadBalancerBackendName),
			},
		}
		change = true
	}

	// 3.check frontend
	frontsConfig := getAzureLoadBalancerFrontendIPConfigByConfig(lb.Spec.Providers.Azure.IPAddressProperties)
	frontsBrief := getAzureLoadBalancerFrontendIPConfigBrief(azlb.FrontendIPConfigurations)
	if !reflect.DeepEqual(frontsConfig, frontsBrief) {
		azlb.FrontendIPConfigurations = frontsConfig
		change = true
	}
	return azlb, change
}

// create a default azure load balancer
func createAzureLoadBalancer(c *client.Client, lb *lbapi.LoadBalancer) (*network.LoadBalancer, error) {
	// init
	azlb := initClusterAzureLoadBalancerObject(lb, lb.Spec.Providers.Azure.ClusterID)
	log.Infof("create azure loadbalancer %s", to.String(azlb.Name))
	azlb, err := c.LoadBalancer.CreateOrUpdate(context.TODO(), lb.Spec.Providers.Azure.ResourceGroupName, to.String(azlb.Name), azlb)
	if err != nil {
		log.Errorf("create lb failed error : %v", err)
		return nil, err
	}
	log.Infof("create load balancer %s success", to.String(azlb.Name))
	return &azlb, nil
}

// ensure sync default rules
func ensureSyncDefaultRules(c *client.Client, azlb *network.LoadBalancer, lb *lbapi.LoadBalancer) (*network.LoadBalancer, error) {
	frontendIPConfigurationID := to.String((*azlb.FrontendIPConfigurations)[0].ID)
	backendAddressPoolID := to.String((*azlb.BackendAddressPools)[0].ID)
	lbRules := azureLBProbes2Rules(frontendIPConfigurationID, backendAddressPoolID, *azlb.Probes)
	var change bool
	if azlb.LoadBalancingRules != nil && len(*azlb.LoadBalancingRules) != 0 {
		change = patchAzureLoadBalancerDefaultRules(azlb, lbRules)
	} else {
		azlb.LoadBalancingRules = &lbRules
		change = true
	}
	if !change {
		return azlb, nil
	}
	var err error
	*azlb, err = c.LoadBalancer.CreateOrUpdate(context.TODO(), lb.Spec.Providers.Azure.ResourceGroupName, to.String(azlb.Name), *azlb)
	if err != nil {
		log.Errorf(" update LoadBalancer failed, %v", err)
		return azlb, err
	}
	return azlb, nil
}

// patchAzureLoadBalancerDefaultRules
// patch default rules if the rule not exist in azure rules
func patchAzureLoadBalancerDefaultRules(azlb *network.LoadBalancer, defaultRules []network.LoadBalancingRule) bool {
	var change bool
	for _, defaultRule := range defaultRules {
		exist := false
		for _, rule := range *azlb.LoadBalancingRules {
			if equalRule(rule, defaultRule) {
				exist = true
				break
			}
		}
		if !exist {
			*azlb.LoadBalancingRules = append(*azlb.LoadBalancingRules, defaultRule)
			change = true
		}
	}
	return change
}

// check rule in same config except chore for example Etag、ProvisioningState
func equalRule(x, brief network.LoadBalancingRule) bool {
	Xbrief := network.LoadBalancingRule{
		Name: x.Name,
		LoadBalancingRulePropertiesFormat: &network.LoadBalancingRulePropertiesFormat{
			FrontendIPConfiguration: &network.SubResource{
				ID: x.FrontendIPConfiguration.ID,
			},
			BackendAddressPool: &network.SubResource{
				ID: x.BackendAddressPool.ID,
			},
			Probe: &network.SubResource{
				ID: x.Probe.ID,
			},
			Protocol:             x.Protocol,
			LoadDistribution:     x.LoadDistribution,
			FrontendPort:         x.FrontendPort,
			BackendPort:          x.BackendPort,
			IdleTimeoutInMinutes: x.IdleTimeoutInMinutes,
			EnableFloatingIP:     x.EnableFloatingIP,
			DisableOutboundSnat:  x.DisableOutboundSnat,
		},
	}
	return reflect.DeepEqual(Xbrief, brief)
}

func getAzureLoadbalancer(c *client.Client, groupName, lbName string) (*network.LoadBalancer, error) {
	if len(lbName) == 0 {
		return nil, nil
	}
	lb, err := c.LoadBalancer.Get(context.TODO(), groupName, lbName, "")
	if client.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		log.Errorf("get lb %s failed: %v", lbName, err)
		return nil, err
	}
	return &lb, nil
}

// ensureSyncRulesAndBackendPools return true if lb has diff with azureLB
// sync rules and BackendPools
func ensureSyncRulesAndBackendPools(c *client.Client, storeLister *core.StoreLister, azlb *network.LoadBalancer, lb *lbapi.LoadBalancer, tcpMap, udpMap map[string]string) error {

	groupName := getGroupName(lb)
	err := syncRules(c, azlb, tcpMap, udpMap, groupName)
	if err != nil {
		return err
	}
	return syncBackendPools(c, storeLister, azlb, lb)
}

func setProvisioningState(lb *lbapi.LoadBalancer, provisioningState string) {
	if lb.Status.ProvidersStatuses.Azure == nil {
		lb.Status.ProvidersStatuses.Azure = &lbapi.AzureProviderStatus{}
	}
	lb.Status.ProvidersStatuses.Azure.ProvisioningState = provisioningState
}

func syncRules(c *client.Client, azlb *network.LoadBalancer, tcpMap, udpMap map[string]string, groupName string) (err error) {

	log.Infof("sync rules azlb group %s name %s , \ntcp rules :%v \nudp rules: %v", groupName, to.String(azlb.Name), tcpMap, udpMap)
	change := makeUpRules(azlb, tcpMap, udpMap)
	if change {
		*azlb, err = c.LoadBalancer.CreateOrUpdate(context.TODO(), groupName, to.String(azlb.Name), *azlb)
		if err != nil {
			log.Errorf("update azure lb failed:%v", err)
			return err
		}
	}
	return nil
}

// sync backend pools
func syncBackendPools(c *client.Client, storeLister *core.StoreLister, azlb *network.LoadBalancer, lb *lbapi.LoadBalancer) error {
	nodes := lb.Spec.Nodes.Names
	return syncBackendPoolsWithNodes(c, storeLister, azlb, nodes)
}

// sync backend pools with spec nodes
func syncBackendPoolsWithNodes(c *client.Client, storeLister *core.StoreLister, azlb *network.LoadBalancer, nodes []string) error {
	log.Infof("sync backend pools azlb name %s", to.String(azlb.Name))
	// nodes := lb.Spec.Nodes.Names
	detachs, attachs, err := diffBackendPools(c, azlb, nodes, storeLister)
	if err != nil {
		return err
	}
	if azlb.BackendAddressPools == nil || len(*azlb.BackendAddressPools) == 0 {
		return fmt.Errorf("backend pools is empty")
	}
	poolID := to.String((*(azlb.BackendAddressPools))[0].ID)
	for _, detach := range detachs {
		err := detachNetworkInterfacesAndLoadBalancer(c, detach, poolID)
		if err != nil {
			return err
		}
	}
	for _, attach := range attachs {
		err := attachNetworkInterfacesAndLoadBalancer(c, attach, poolID)
		if err != nil {
			return err
		}
	}
	return nil
}

func detachNetworkInterfacesAndLoadBalancer(c *client.Client, detachID string, poolID string) error {
	log.Infof("detach network %s and backend pool %s", detachID, poolID)
	groupName, name, err := getGroupAndResourceNameFromID(detachID, azureNetworkInterfaces)
	if err != nil {
		return err
	}
	nic, err := c.NetworkInterface.Get(context.TODO(), groupName, name, "")
	if err != nil {
		return err
	}
	if nic.IPConfigurations == nil {
		log.Warnf("networkInterface[%s/%s] ipc is zero, skip it", name)
		return nil
	}

	ipcs := *(nic.IPConfigurations)
	needUpdate := false

	for i := range ipcs {
		if ipcs[i].LoadBalancerBackendAddressPools == nil || 0 == len(*(ipcs[i].LoadBalancerBackendAddressPools)) {
			continue
		}
		pools := *(ipcs[i].LoadBalancerBackendAddressPools)
		newPools := make([]network.BackendAddressPool, 0, len(pools))
		found := false
		for _, pool := range pools {
			if strings.ToLower(to.String(pool.ID)) == strings.ToLower(poolID) {
				found = true
			} else {
				newPools = append(newPools, pool)
			}
		}
		if found {
			ipcs[i].LoadBalancerBackendAddressPools = &newPools
			needUpdate = true
		}
	}
	if needUpdate {
		_, err := c.NetworkInterface.CreateOrUpdate(context.TODO(), groupName, name, nic)
		if err != nil {
			log.Errorf("update networkInterfaces error %v", err)
			return err
		}
	}
	return nil
}

func attachNetworkInterfacesAndLoadBalancer(c *client.Client, attachID string, poolID string) error {
	log.Infof("attach network %s and backend pool %s", attachID, poolID)
	groupName, name, err := getGroupAndResourceNameFromID(attachID, azureNetworkInterfaces)
	if err != nil {
		return err
	}
	nic, err := c.NetworkInterface.Get(context.TODO(), groupName, name, "")
	if err != nil {
		return err
	}
	if nic.IPConfigurations == nil {
		log.Warnf("networkInterface[%s/%s] ipc is zero, skip it", name)
		return fmt.Errorf("ipc is zero")
	}

	ipcs := *(nic.IPConfigurations)
	needUpdate := false

	for i := range ipcs {
		found := false
		pools := ipcs[i].LoadBalancerBackendAddressPools
		if pools != nil && len(*pools) != 0 {
			for _, pool := range *pools {
				if strings.ToLower(to.String(pool.ID)) == strings.ToLower(poolID) {
					found = true
					break
				}
			}
		}
		if !found {
			if pools == nil {
				newPools := make([]network.BackendAddressPool, 0, 1)
				pools = &newPools
			}
			*pools = append(*pools, network.BackendAddressPool{
				ID: to.StringPtr(poolID),
			})
			ipcs[i].LoadBalancerBackendAddressPools = pools
			needUpdate = true
		}
	}
	if needUpdate {
		_, err := c.NetworkInterface.CreateOrUpdate(context.TODO(), groupName, name, nic)
		if err != nil {
			log.Errorf("update networkInterfaces error %v", err)
			return err
		}
	}
	return nil
}

// init azure loadbalancer with no probe rules and backends
func initClusterAzureLoadBalancerObject(lb *lbapi.LoadBalancer, clusterID string) network.LoadBalancer {
	lbName := lb.Spec.Providers.Azure.Name
	if len(lbName) == 0 {
		lbName = getAzureLBName(lb.Name, clusterID)
	}
	azlb := network.LoadBalancer{
		Name:     to.StringPtr(lbName),
		Location: to.StringPtr(lb.Spec.Providers.Azure.Location),
		Sku: &network.LoadBalancerSku{
			Name: network.LoadBalancerSkuName(lb.Spec.Providers.Azure.SKU),
		},
		LoadBalancerPropertiesFormat: &network.LoadBalancerPropertiesFormat{},
	}
	frontendIPConfigurations := getAzureLoadBalancerFrontendIPConfigByConfig(lb.Spec.Providers.Azure.IPAddressProperties)
	azlb.FrontendIPConfigurations = frontendIPConfigurations
	return azlb
}

// cleanup
func recoverDefaultAzureLoadBalancer(c *client.Client, groupName, lbName string) (err error) {
	log.Infof("recover default azlb group %s name %s", groupName, lbName)
	defer func() {
		if err != nil {
			log.Errorf("recoverDefaultAzureLoadBalancer failed %v", err)
		}
	}()
	azlb, err := c.LoadBalancer.Get(context.TODO(), groupName, lbName, "")
	if err != nil {
		return err
	}
	// 1.clean all rules
	if azlb.LoadBalancingRules != nil && len(*azlb.LoadBalancingRules) != 0 {
		*azlb.LoadBalancingRules = (*azlb.LoadBalancingRules)[:0]
	}

	// 2.clean probe
	if azlb.Probes != nil && len(*azlb.Probes) != 0 {
		*azlb.Probes = (*azlb.Probes)[:0]
	}

	pools := azlb.BackendAddressPools

	// update azlb
	log.Infof("start recover default azure load lb")
	_, err = c.LoadBalancer.CreateOrUpdate(context.TODO(), groupName, lbName, azlb)
	if err != nil {
		return err
	}

	if pools == nil || len(*pools) == 0 {
		return nil
	}
	// 3.clean backend pools netInterfaces
	pool := (*pools)[0]
	poolID := to.String(pool.ID)

	// 4.detach all netInterfaces with backend pool ID
	if pool.BackendIPConfigurations == nil || len(*pool.BackendIPConfigurations) == 0 {
		return nil
	}
	log.Infof("detach BackendIPConfigurations len %v to poolID %s", len(*pool.BackendIPConfigurations), poolID)
	configs := *pool.BackendIPConfigurations
	for _, config := range configs {
		err = detachNetworkInterfacesAndLoadBalancer(c, to.String(config.ID), poolID)
		if err != nil {
			return err
		}
	}
	return nil
}
