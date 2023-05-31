<p align="center">
    <img src="/assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text_300dpi_10pct.png" alt="Welcome to EdgeNet" width="400">
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/EdgeNet-project/edgenet)](https://goreportcard.com/report/github.com/EdgeNet-project/edgenet)
[![Build Status](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml/badge.svg?branch=main)](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml)
[![Coverage Status](https://coveralls.io/repos/github/EdgeNet-project/edgenet/badge.svg?branch=main)](https://coveralls.io/github/EdgeNet-project/edgenet?branch=main)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/EdgeNet-project/edgenet)](https://github.com/EdgeNet-project/edgenet/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/EdgeNet-project/edgenet.svg)](https://pkg.go.dev/github.com/EdgeNet-project/edgenet)

## What is EdgeNet?
EdgeNet is Kubernetes adapted for the network edge. It allows you to deploy applications to hundreds of nodes that are scattered across the internet, rather than just one or a small number of centralized data centers.

EdgeNet software runs all around the world and it makes edgenet-test bed possible. We encourage people to contribute to nodes and join the EdgeNet-Testbed. For more information visit the [website](https://edge-net.org).

## Features
To extend, and adapt Kubernetes into edge computing, EdgeNet employs various features. 
* Multitenancy: EdgeNet enables the utilization of a shared cluster by multiple tenants who lack trust in each other. Tenants can allocate resource quotas or slices, and they also have the ability to offer their resources to other tenants. This functionality empowers tenants to function both as providers and consumers, operating in both vendor and consumer modes

* Multiprovider: By accommodating the collaboration of diverse providers, EdgeNet encourages numerous entities to contribute to nodes, thus fostering a rich and expansive ecosystem that thrives on heterogeneity. With the power of multitenancy, contributors with different hardware can easily lend their hardware.

* Selective deployment: While the involvement of multiple providers in EdgeNet extends beyond hardware vending, the possibilities encompass a broader spectrum. Node contributions can originate from individuals across the globe, and by leveraging a selective deployment mechanism, EdgeNet empowers the targeted deployment of resources to specific geographical regions, thereby augmenting localization capabilities and enabling efficient utilization of computing power where it is most needed.

* Federation support: EdgeNet envisions the federation of Kubernetes clusters worldwide, starting from the edge. By granting clusters the ability to assume worker or federation roles, EdgeNet enables the outsourcing of workloads to these clusters, fostering a seamless and globally interconnected network of distributed computing resources.


<!-- Tenant resource quota:  -->

<!-- Variable slice granularity: -->

## Code layout and architecture
If you are familiar with the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) used
by other Kubernetes-related projects, you will easily be able to navigate this repository.

EdgeNet extends Kubernetes via [custom controllers](https://kubernetes.io/docs/concepts/architecture/controller/). They check for state changes of custom EdgeNet resources and try to converge the current state with the desired state. The best practice is to deploy the controllers in the clusters. For installation please refer to the [installation tutorial](/docs/installation/README.md).

The architecture is described extensively in the [architecture document](/docs/architecture) and can be seen in the diagram below.

![EdgeNet Architecture Diagram](/docs/architecture/architecture.png)

## Tutorials and Documentation
EdgeNet provides extensive documentation and tutorials to make the adoption easier and faster. All of the documentation currently resides in the doc folder and can be accessed by clicking [here](./docs/README.md).

You can directly access all of the tutorials by clicking [here](./docs/tutorials/README.md).

## Support

To chat with a member of the EdgeNet team live, please [open our tawk.to window](https://tawk.to/edgenet).

## Contributing

The EdgeNet software is free and open source, licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0); we invite you to contribute.
