package v1alpha2

import (
	"fmt"
	"net"
)

// ValidateLoadBalancer validate loadbalancer
func ValidateLoadBalancer(lb *LoadBalancer) error {

	if lb.Spec.Providers.Ipvsdr != nil {
		ipvsdr := lb.Spec.Providers.Ipvsdr
		if net.ParseIP(ipvsdr.Vip) == nil {
			return fmt.Errorf("ipvsdr: vip is invalid")
		}
		switch ipvsdr.Scheduler {
		case IpvsSchedulerRR:
		case IpvsSchedulerWRR:
		case IpvsSchedulerLC:
		case IpvsSchedulerWLC:
		case IpvsSchedulerLBLC:
		case IpvsSchedulerDH:
		case IpvsSchedulerSH:
			break
		default:
			return fmt.Errorf("ipvsdr: scheduler %v is invalid", ipvsdr.Scheduler)
		}
	}

	return nil
}
