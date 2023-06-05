<p align="center">
    <img src="assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text_300dpi_10pct.png" alt="Welcome to EdgeNet" width="400">
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/EdgeNet-project/edgenet)](https://goreportcard.com/report/github.com/EdgeNet-project/edgenet)
[![Build Status](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml/badge.svg?branch=main)](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml)
[![Coverage Status](https://coveralls.io/repos/github/EdgeNet-project/edgenet/badge.svg?branch=main)](https://coveralls.io/github/EdgeNet-project/edgenet?branch=main)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/EdgeNet-project/edgenet)](https://github.com/EdgeNet-project/edgenet/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/EdgeNet-project/edgenet.svg)](https://pkg.go.dev/github.com/EdgeNet-project/edgenet)

# What is EdgeNet?

EdgeNet is an open-source Kubernetes extension that is designed for the network edge. It allows you to deploy applications to hundreds of nodes that are scattered across the internet, rather than just one or a small number of centralized data centers.

If you are looking for [EdgeNet-Testbed](https://edge-net.org), a globally distributed test cloud running EdgeNet software, we encourage you to contribute nodes and join the testbed and help scientific research.

This repository contains the source code and documentation for EdgeNet software.

# Using EdgeNet
## Getting Started

EdgeNet can be deployed in any Kubernetes cluster with a couple of simple steps. To deploy EdgeNet to your private Kubernetes cluster please refer to the [deploying EdgeNet to Kubernetes tutorial](/docs/tutorials/deploy_edgenet_to_kube.md). 

## Features
To extend, and adapt Kubernetes into edge computing, EdgeNet employs various features. You can click on them to go through detailed documentation. 
    
* [Multitenancy](/docs/custom_resources.md#multitenancy): EdgeNet enables the utilization of a shared cluster by multiple tenants who lack trust in each other. Tenants can allocate resource quotas or slices, and they also have the ability to offer their resources to other tenants. This functionality empowers tenants to function both as providers and consumers, operating in both vendor and consumer modes

* [Multiprovider](/docs/custom_resources.md#multiprovider): By accommodating the collaboration of diverse providers, EdgeNet encourages numerous entities to contribute to nodes, thus fostering a rich and expansive ecosystem that thrives on heterogeneity. With the power of multitenancy, contributors with different hardware can easily lend their hardware.

* [Selective deployment](/docs/custom_resources.md#selective-deployment): While the involvement of multiple providers in EdgeNet extends beyond hardware vending, the possibilities encompass a broader spectrum. Node contributions can originate from individuals across the globe, and by leveraging a selective deployment mechanism, EdgeNet empowers the targeted deployment of resources to specific geographical regions, thereby augmenting localization capabilities and enabling efficient utilization of computing power where it is most needed.

* [Federation support](/docs/custom_resources.md#cluster-federation): EdgeNet envisions the federation of Kubernetes clusters worldwide, starting from the edge. By granting clusters the ability to assume worker or federation roles, EdgeNet enables the outsourcing of workloads to these clusters, fostering a seamless and globally interconnected network of distributed computing resources.

<!-- ## About the Source Code
If you are familiar with the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) used by other Kubernetes-related projects, you will easily be able to navigate this repository.

EdgeNet extends Kubernetes via [custom controllers](https://kubernetes.io/docs/concepts/architecture/controller/). They check for state changes of custom EdgeNet resources and try to converge the current state with the desired state. We EdgeNet source code contains these controllers' source code. You can find the docker controller images in [EdgeNet's DockerHub](https://hub.docker.com/u/edgenetio).


The architecture of EdgeNet is described in the [architecture document](/docs/architecture/README.md). -->

## Tutorials and Documentation
If you are planning to use EdgeNet software in your Kubernetes cluster, we highly encourage you to check out the [documentation](/docs/README.md).

You can directly access all of the [tutorials](./docs/tutorials/README.md) under the `doc` folder in the main repository.

## Support

To chat with a member of the EdgeNet team live, please [open our tawk.to window](https://tawk.to/edgenet).

## Contributing

The EdgeNet software is free and open source, licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0); we invite you to contribute. You can access [contribution guide](/docs/guides/contribution_guides.md) for more information on how to contribute.
