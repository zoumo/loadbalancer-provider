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
	"reflect"
	"sync"

	log "github.com/zoumo/logdog"

	netv1alpha1 "github.com/caicloud/loadbalancer-controller/pkg/apis/networking/v1alpha1"
	"github.com/caicloud/loadbalancer-controller/pkg/informers"
	netlisters "github.com/caicloud/loadbalancer-controller/pkg/listers/networking/v1alpha1"
	controllerutil "github.com/caicloud/loadbalancer-controller/pkg/util/controller"
	"github.com/caicloud/loadbalancer-controller/pkg/util/validation"

	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// GenericProvider holds the boilerplate code required to build an LoadBalancer Provider.
type GenericProvider struct {
	cfg *Configuration

	queue    workqueue.RateLimitingInterface
	factory  informers.SharedInformerFactory
	lbLister netlisters.LoadBalancerLister

	helper *controllerutil.Helper

	// stopLock is used to enforce only a single call to Stop is active.
	// Needed because we allow stopping through an http endpoint and
	// allowing concurrent stoppers leads to stack traces.
	stopLock *sync.Mutex
	stopCh   chan struct{}
	shutdown bool
}

// NewLoadBalancerProvider returns a configured LoadBalancer controller
func NewLoadBalancerProvider(cfg *Configuration) *GenericProvider {

	gp := &GenericProvider{
		cfg:      cfg,
		factory:  informers.NewSharedInformerFactory(cfg.KubeClient, cfg.TPRClient, 0),
		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "loadbalancer"),
		stopLock: &sync.Mutex{},
		stopCh:   make(chan struct{}),
	}

	lbinformer := gp.factory.Networking().V1alpha1().LoadBalancer()
	cminformer := gp.factory.Core().V1().ConfigMaps()
	lbinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    gp.addLoadBalancer,
		UpdateFunc: gp.updateLoadBalancer,
		DeleteFunc: gp.deleteLoadBalancer,
	})
	cminformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		UpdateFunc: gp.updateConfigMap,
	})

	// sync nodes
	nodeinformer := gp.factory.Core().V1().Nodes()
	nodeinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{})

	gp.cfg.Backend.SetListers(StoreLister{
		Node:         nodeinformer.Lister(),
		LoadBalancer: lbinformer.Lister(),
		ConfigMap:    cminformer.Lister(),
	})

	gp.helper = controllerutil.NewHelper(&netv1alpha1.LoadBalancer{}, gp.queue, gp.syncLoadBalancer)
	gp.lbLister = lbinformer.Lister()

	return gp
}

// Start starts the LoadBalancer Provider.
func (p *GenericProvider) Start() {
	defer utilruntime.HandleCrash()
	log.Info("Startting provider")

	p.factory.Start(p.stopCh)

	// wait cache synced
	log.Info("Wait for all caches synced")
	synced := p.factory.WaitForCacheSync(p.stopCh)
	for tpy, sync := range synced {
		if !sync {
			log.Error("Wait for cache sync timeout", log.Fields{"type": tpy})
			return
		}
	}
	log.Info("All caches have synced, Running LoadBalancer Controller ...")

	// start backend
	p.cfg.Backend.Start()
	if !p.cfg.Backend.WaitForStart() {
		log.Error("Wait for backend start timeout")
		return
	}

	// start worker
	p.helper.Run(1, p.stopCh)

	<-p.stopCh

}

// Stop stops the LoadBalancer Provider.
func (p *GenericProvider) Stop() error {
	log.Info("Shutting down provider")
	p.stopLock.Lock()
	defer p.stopLock.Unlock()
	// Only try draining the workqueue if we haven't already.
	if !p.shutdown {
		p.shutdown = true
		log.Info("close channel")
		close(p.stopCh)
		// stop backend
		log.Info("stop backend")
		p.cfg.Backend.Stop()
		// stop syncing
		log.Info("shutting down controller queue")
		p.helper.ShutDown()
		return nil
	}

	return fmt.Errorf("shutdown already in progress")
}

func (p *GenericProvider) addLoadBalancer(obj interface{}) {
	lb := obj.(*netv1alpha1.LoadBalancer)
	if p.filterLoadBalancer(lb) {
		return
	}
	log.Info("Adding LoadBalancer ")
	p.helper.Enqueue(lb)
}

func (p *GenericProvider) updateLoadBalancer(oldObj, curObj interface{}) {
	old := oldObj.(*netv1alpha1.LoadBalancer)
	cur := curObj.(*netv1alpha1.LoadBalancer)

	if old.ResourceVersion == cur.ResourceVersion {
		// Periodic resync will send update events for all known LoadBalancer.
		// Two different versions of the same LoadBalancer will always have different RVs.
		return
	}

	if p.filterLoadBalancer(cur) {
		return
	}
	log.Info("Updating LoadBalancer")

	p.helper.Enqueue(cur)

}

func (p *GenericProvider) deleteLoadBalancer(obj interface{}) {
	lb, ok := obj.(*netv1alpha1.LoadBalancer)

	if !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Couldn't get object from tombstone %#v", obj))
			return
		}
		lb, ok = tombstone.Obj.(*netv1alpha1.LoadBalancer)
		if !ok {
			utilruntime.HandleError(fmt.Errorf("Tombstone contained object that is not a LoadBalancer %#v", obj))
			return
		}
	}

	if p.filterLoadBalancer(lb) {
		return
	}

	log.Info("Deleting LoadBalancer")

	p.helper.Enqueue(lb)
}

func (p *GenericProvider) updateConfigMap(oldObj, curObj interface{}) {
	old := oldObj.(*v1.ConfigMap)
	cur := curObj.(*v1.ConfigMap)

	if old.ResourceVersion == cur.ResourceVersion {
		// Periodic resync will send update events for all known LoadBalancer.
		// Two different versions of the same LoadBalancer will always have different RVs.
		return
	}

	// namespace and name can not change, so we check one of them is enough
	if p.filterConfigMap(cur) {
		return
	}

	if reflect.DeepEqual(old.Data, cur.Data) {
		// nothing changed
		return
	}

	p.helper.Enqueue(cache.ExplicitKey(p.cfg.LoadBalancerNamespace + "/" + p.cfg.LoadBalancerName))

}

func (p *GenericProvider) filterLoadBalancer(lb *netv1alpha1.LoadBalancer) bool {
	if lb.Namespace == p.cfg.LoadBalancerNamespace && lb.Name == p.cfg.LoadBalancerName {
		return false
	}

	return true
}

func (p *GenericProvider) filterConfigMap(cm *v1.ConfigMap) bool {
	if cm.Namespace == p.cfg.LoadBalancerNamespace && (cm.Name == p.cfg.TCPConfigMap || cm.Name == p.cfg.UDPConfigMap) {
		return false
	}
	return true
}

func (p *GenericProvider) syncLoadBalancer(obj interface{}) error {
	key := obj.(string)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	lb, err := p.lbLister.LoadBalancers(namespace).Get(name)
	if errors.IsNotFound(err) {
		log.Warn("LoadBalancer has been deleted", log.Fields{"lb": key})
		// deleted
		// TODO shutdown?
		return nil
	}
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Unable to retrieve LoadBalancer %v from store: %v", key, err))
		return err
	}

	if err := validation.ValidateLoadBalancer(lb); err != nil {
		log.Debug("invalid loadbalancer scheme", log.Fields{"err": err})
		return err
	}

	return p.cfg.Backend.OnUpdate(lb)
}
