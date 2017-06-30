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
	netv1alpha1 "github.com/caicloud/loadbalancer-controller/pkg/apis/networking/v1alpha1"
	netlisters "github.com/caicloud/loadbalancer-controller/pkg/listers/networking/v1alpha1"
	v1listers "k8s.io/client-go/listers/core/v1"
)

// Provider holds the methods to handle an Provider backend
type Provider interface {
	// Info returns information about the loadbalancer provider
	Info() Info
	// SetListers allows the access of store listers present in the generic controller
	// This avoid the use of the kubernetes client.
	SetListers(StoreLister)
	// OnUpdate callback invoked when loadbalancer changed
	OnUpdate(*netv1alpha1.LoadBalancer) error
	// Start starts the loadbalancer provider
	Start()
	// WaitForStart waits for provider fully run
	WaitForStart() bool
	// Stop shuts down the loadbalancer provider
	Stop() error
}

// Info returns information about the provider.
// This fields contains information that helps to track issues or to
// map the running loadbalancer provider to source code
type Info struct {
	// Name returns the name of the backend implementation
	Name string `json:"name"`
	// Release returns the running version (semver)
	Release string `json:"release"`
	// Build returns information about the git commit
	Build string `json:"build"`
	// Repository return information about the git repository
	Repository string `json:"repository"`
}

// StoreLister returns the configured store for loadbalancers, nodes
type StoreLister struct {
	LoadBalancer netlisters.LoadBalancerLister
	Node         v1listers.NodeLister
}
