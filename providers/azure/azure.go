package azure

import (
	"context"
	"fmt"
	"reflect"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-01-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	log "github.com/zoumo/logdog"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/caicloud/clientset/kubernetes"
	lbapi "github.com/caicloud/clientset/pkg/apis/loadbalance/v1alpha2"
	core "github.com/caicloud/loadbalancer-provider/core/provider"
	"github.com/caicloud/loadbalancer-provider/pkg/version"
	"github.com/caicloud/loadbalancer-provider/providers/azure/client"
)

// AzureProvider azure lb provider
type AzureProvider struct {
	storeLister           core.StoreLister
	clientset             *kubernetes.Clientset
	loadBalancerNamespace string
	loadBalancerName      string

	// old azure lb azure spec
	oldAzureProvider *lbapi.AzureProvider
	nodes            []string

	// load balancer rules cache
	tcpRuleMap map[string]string
	udpRuleMap map[string]string
}

// New creates a new azure LoadBalancer Provider.
func New(clientset *kubernetes.Clientset, name, namespace string) (*AzureProvider, error) {
	azure := &AzureProvider{
		clientset:             clientset,
		loadBalancerName:      name,
		loadBalancerNamespace: namespace,
	}
	return azure, nil
}

// SetListers set store lister
func (l *AzureProvider) SetListers(storeLister core.StoreLister) {
	l.storeLister = storeLister
}

func (l *AzureProvider) setCacheAzureLoadbalancer(azure *lbapi.AzureProvider) {
	if l.oldAzureProvider == nil {
		l.oldAzureProvider = &lbapi.AzureProvider{}
	}
	l.oldAzureProvider.Name = azure.Name
	l.oldAzureProvider.ResourceGroupName = azure.ResourceGroupName
}

func (l *AzureProvider) setCacheReserveStatus(reserve *bool) {
	if l.oldAzureProvider == nil {
		l.oldAzureProvider = &lbapi.AzureProvider{}
	}
	l.oldAzureProvider.ReserveAzure = reserve
}

// OnUpdate update loadbalancer
func (l *AzureProvider) OnUpdate(lb *lbapi.LoadBalancer) error {

	log.Infof("OnUpdate......")
	if lb.Spec.Providers.Azure == nil {
		return l.cleanupAzureLB()
	}
	// ignore change of azure's name groupName and reserve status
	l.setCacheAzureLoadbalancer(lb.Spec.Providers.Azure)
	l.setCacheReserveStatus(lb.Spec.Providers.Azure.ReserveAzure)

	// tell if change of load balancer
	tcp, udp, ruleChange, err := l.getProxyConfigMapAndCompare(lb)
	if err != nil {
		return err
	}
	// ignore change of other providers
	if reflect.DeepEqual(lb.Spec.Providers.Azure, l.oldAzureProvider) &&
		reflect.DeepEqual(l.nodes, lb.Spec.Nodes.Names) &&
		!ruleChange {
		return nil
	}

	// check lb exist in cluster
	_, err = l.storeLister.LoadBalancer.LoadBalancers(lb.Namespace).Get(lb.Name)
	if errors.IsNotFound(err) {
		return l.cleanupAzureLB()
	}
	if err != nil {
		log.Errorf("get lb falied %v", err)
		return err
	}

	nlb := lb.DeepCopy()

	azlb, err := l.ensureSync(nlb, tcp, udp)

	l.updateLoadBalancerAzureStatus(azlb, lb, err)
	if err == nil {
		log.Infof("update cache data %v", nlb.Spec.Providers.Azure)
		l.updateCacheData(nlb, tcp, udp)
	}
	return err
}

func (l *AzureProvider) updateCacheData(lb *lbapi.LoadBalancer, tcp, udp map[string]string) {
	l.oldAzureProvider = lb.Spec.Providers.Azure
	l.nodes = lb.Spec.Nodes.Names
	l.tcpRuleMap = tcp
	l.udpRuleMap = udp
}

// Start ...
func (l *AzureProvider) Start() {
	log.Infof("Startting azure provider ns %s name %s", l.loadBalancerNamespace, l.loadBalancerName)
	return
}

// Stop ...
func (l *AzureProvider) Stop() error {
	log.Infof("end provider azure...")
	_, err := l.storeLister.LoadBalancer.LoadBalancers(l.loadBalancerNamespace).Get(l.loadBalancerName)
	if errors.IsNotFound(err) {
		return l.cleanupAzureLB()
	}
	return nil
}

// WaitForStart waits for
func (l *AzureProvider) WaitForStart() bool {
	// err := wait.Poll(time.Second, 60*time.Second, func() (bool, error) {
	// 	//
	// 	return true, nil
	// })

	// if err != nil {
	// 	return false
	// }
	return true
}

// Info information about the provider.
func (l *AzureProvider) Info() core.Info {
	info := version.Get()
	return core.Info{
		Name:      "azure",
		Version:   info.Version,
		GitCommit: info.GitCommit,
		GitRemote: info.GitRemote,
	}
}

func (l *AzureProvider) updateLoadBalancerAzureStatus(azlb *network.LoadBalancer, lb *lbapi.LoadBalancer, err error) {
	if azlb != nil {
		setProvisioningState(lb, to.String(azlb.ProvisioningState))
	}
	if err == nil {
		l.patchLoadBalancerAzureStatus(lb, lbapi.AzureRunningPhase, nil)
	} else {
		l.patchLoadBalancerAzureStatus(lb, lbapi.AzureErrorPhase, err)
	}
}

// make sure azure lb config stay in same with compass lb
func (l *AzureProvider) ensureSync(lb *lbapi.LoadBalancer, tcp, udp map[string]string) (*network.LoadBalancer, error) {

	azureSpec := lb.Spec.Providers.Azure
	log.Infof("start sync azlb group %s name %s", azureSpec.ResourceGroupName, azureSpec.Name)

	// update status
	_, err := l.patchLoadBalancerAzureStatus(lb, lbapi.AzureUpdatingPhase, nil)
	if err != nil {
		return nil, err
	}

	c, err := client.NewClient(&l.storeLister)
	if err != nil {
		log.Errorf("init client error %v", err)
		return nil, err
	}

	// get a valid azure load balancer
	azlb, err := l.ensureAzureLoadbalancer(c, lb)
	if err != nil {
		return nil, err
	}

	azlb, err = ensureSyncDefaultAzureLBConfig(c, azlb, lb)

	if err != nil {
		return nil, err
	}

	err = ensureSyncRulesAndBackendPools(c, &l.storeLister, azlb, lb, tcp, udp)

	return azlb, err
}

// get compass lb proxy info and compare with cache data
func (l *AzureProvider) getProxyConfigMapAndCompare(lb *lbapi.LoadBalancer) (map[string]string, map[string]string, bool, error) {
	tcpCm, err := l.storeLister.ConfigMap.ConfigMaps(lb.Namespace).Get(lb.Status.ProxyStatus.TCPConfigMap)
	if err != nil {
		log.Errorf("get namespace %s cm %s failed err : %v", lb.Namespace, lb.Status.ProxyStatus.TCPConfigMap, err)
		return nil, nil, false, client.NewServiceError("K8SStore", err.Error())
	}
	udpCm, err := l.storeLister.ConfigMap.ConfigMaps(lb.Namespace).Get(lb.Status.ProxyStatus.UDPConfigMap)
	if err != nil {
		log.Errorf("get namespace %s cm %s failed err : %v", lb.Namespace, lb.Status.ProxyStatus.TCPConfigMap, err)
		return nil, nil, false, client.NewServiceError("K8SStore", err.Error())
	}
	if len(l.tcpRuleMap) != len(tcpCm.Data) || len(l.udpRuleMap) != len(udpCm.Data) {
		return tcpCm.Data, udpCm.Data, true, nil
	}
	for key, value := range l.tcpRuleMap {
		v, ok := tcpCm.Data[key]
		if !ok {
			return tcpCm.Data, udpCm.Data, true, nil
		}
		if v != value {
			return tcpCm.Data, udpCm.Data, true, nil
		}
	}
	for key, value := range l.udpRuleMap {
		v, ok := udpCm.Data[key]
		if !ok {
			return tcpCm.Data, udpCm.Data, true, nil
		}
		if v != value {
			return tcpCm.Data, udpCm.Data, true, nil
		}
	}
	return tcpCm.Data, udpCm.Data, false, nil
}

// get a valid azure load balancer
func (l *AzureProvider) ensureAzureLoadbalancer(c *client.Client, lb *lbapi.LoadBalancer) (*network.LoadBalancer, error) {
	azureSpec := lb.Spec.Providers.Azure
	azlb, err := getAzureLoadbalancer(c, azureSpec.ResourceGroupName, azureSpec.Name)
	if err != nil {
		return nil, err
	}
	if azlb == nil {
		azlb, err = createAzureLoadBalancer(c, lb)
		if err != nil {
			return nil, err
		}
		if len(azureSpec.Name) == 0 {
			azureSpec.Name = to.String(azlb.Name)
			err = l.pathLoadBalancerName(lb, azureSpec.Name)
			if err != nil {
				return nil, err
			}
		}
	}
	return azlb, nil
}

func (l *AzureProvider) pathLoadBalancerName(lb *lbapi.LoadBalancer, name string) error {
	lb.Spec.Providers.Azure.Name = name
	l.setCacheAzureLoadbalancer(lb.Spec.Providers.Azure)
	patch := fmt.Sprintf(`{"spec":{"providers":{"azure":{"name":"%s"}}}}`, name)
	_, err := l.clientset.LoadbalanceV1alpha2().LoadBalancers(lb.Namespace).Patch(lb.Name, types.MergePatchType, []byte(patch))
	if err != nil {
		log.Errorf("patch lb %s failed %v", lb.Name, err)
		return err
	}
	return nil
}

// patch load balancer azure status
func (l *AzureProvider) patchLoadBalancerAzureStatus(lb *lbapi.LoadBalancer, phase lbapi.AzureProviderPhase, result error) (*lbapi.LoadBalancer, error) {
	var reason, message string
	var serviceError *client.ServiceError
	switch t := result.(type) {
	case autorest.DetailedError:
		serviceError = client.ParseServiceError(result)
		if serviceError != nil {
			reason = serviceError.Code
			message = serviceError.Message
		}
	case *client.ServiceError:
		reason = t.Code
		message = t.Message
	default:
		if result != nil {
			reason = "Unknown"
			message = result.Error()
		}
	}

	var provisioningState string
	if lb.Status.ProvidersStatuses.Azure != nil {
		provisioningState = lb.Status.ProvidersStatuses.Azure.ProvisioningState
	}
	patch := fmt.Sprintf(azureProviderStatusFormat, phase, reason, message, provisioningState)
	lb, err := l.clientset.LoadbalanceV1alpha2().LoadBalancers(lb.Namespace).Patch(lb.Name, types.MergePatchType, []byte(patch))
	if err != nil {
		log.Errorf("patch lb %s failed %v", lb.Name, err)
		return nil, err
	}
	return lb, nil
}

// clean up azure lb info and make oldAzureProvider nil
func (l *AzureProvider) cleanupAzureLB() error {
	if l.oldAzureProvider == nil || len(l.oldAzureProvider.Name) == 0 {
		log.Errorf("old azure info nil")
		return nil
	}
	c, err := client.NewClient(&l.storeLister)
	if err != nil {
		log.Errorf("init client error %v", err)
		return err
	}

	defer func() {
		if err == nil {
			l.oldAzureProvider = nil
		}
	}()

	if to.Bool(l.oldAzureProvider.ReserveAzure) {
		err = recoverDefaultAzureLoadBalancer(c, l.oldAzureProvider.ResourceGroupName, l.oldAzureProvider.Name)
		return err
	}
	log.Infof("delete azure lb group %s name %s", l.oldAzureProvider.ResourceGroupName, l.oldAzureProvider.Name)
	err = c.LoadBalancer.Delete(context.TODO(), l.oldAzureProvider.ResourceGroupName, l.oldAzureProvider.Name)
	log.Infof("delete result %v", err)
	return err
}
