<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [LoadBalancer Provider](#loadbalancer-provider)
  - [About the project](#about-the-project)
    - [Status](#status)
    - [Design](#design)
    - [See also](#see-also)
  - [Getting stated](#getting-stated)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# LoadBalancer Provider

## About the project

This repo is the sub project of [loadbalancer-controller](https://github.com/caicloud/loadbalancer-controller) containing providers of loadbalancer.

### Status

**Working in process**, still in alpha version

### Design

Learn more about loadbalancer, design doc on [google drive](https://docs.google.com/document/d/1TfiO_AFgEV1V_1T10Cg08b1iqMG1J6foQhvqXYAkCMw/edit#heading=h.y5pdkaud6h8o)

### See also

-   [loadbalancer-controller](https://github.com/caicloud/loadbalancer-controller)

## Getting stated

### Layout

```
├── core
│   ├── options
│   ├── pkg
│   │   ├── arp
│   │   ├── net
│   │   ├── node
│   │   └── sysctl
│   └── provider
└── providers
    ├── ingress
    │   ├── cmd
    │   ├── provider
    │   └── version
    └── ipvsdr
        ├── cmd
        ├── provider
        └── version
```

-   `core/pkg` contains generic pkgs, etc, arp, sysctl, net, node
-   `core/options` contains generic options
-   `core/provider` contains generic loadbalancer provider
-   `providers/ingress` contains ingress sidecar for loadbalancer proxy
-   `providers/ipvsdr` contains IPVS DR mode provider backend