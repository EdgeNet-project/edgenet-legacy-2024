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

# Create an EdgeNet Cluster

To create an EdgeNet cluster you need to have access to a Kubernetes cluster. If you want to create one, you can see the Kubernetes cluster created with [minikube](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/) or [kubeadm](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/) from the official documentation.


You first need to download the `all-in-one.yaml` file to install all of the features. If you only want to use a specific set of features you can check out our [advanced installation guide](/docs/tutorials/deploy_edgenet_to_kube.md). To download, you need to specify the branch of EdgeNet. The default is currently `release-1.0`.

```bash
RELEASE=release-1.0

curl -so all-in-one.yaml https://raw.githubusercontent.com/EdgeNet-project/edgenet/$RELEASE/build/yamls/kubernetes/all-in-one.yaml
```

Then edit the secrets for using the external API features, such as Slack, email, geological ip database, etc. If you leave them blank, you won't be able to use specific features but other features will continue to work. Don't forget to encode the tokens in base64. You can do it by using this command: `echo "<token-or-secret>" | base64`.

```yaml

  headnode.yaml: |
    # dns: "<Root domain>"
    # ip: "<IP address of the control plane node>"
  smtp.yaml: |
    # host: "<Hostname of the smtp server>"
    # port: "<Port of the smtp client>"
    # from: "<Mail address of the sender of notifications>"
    # username : "<Username of the account>"
    # password : "<Password of the account>"
    # to: "<Mail address of the administrator>"
  console.yaml: |
    # url: "<URL of the console>"
  namecheap.yaml: |
    # Provide the namecheap credentials for DNS records.
    # app: "<App name>"
    # apiUser : "<API user>"
    # apiToken : "<API Token>"
    # username : "<Username>"
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"

```

Then simply apply the changes with `kubectl`.

```bash
kubectl apply -f all-in-one.yaml
```

You are done! you just need to wait for Kubernetes to spin the EdgeNet controllers.

# Features
To extend, and adapt Kubernetes into edge computing, EdgeNet employs various features. You can click on them to go through detailed documentation. 
    
* [Multitenancy](/docs/custom_resources.md#multitenancy): EdgeNet enables the utilization of a shared cluster by multiple tenants who lack trust in each other. Tenants can allocate resource quotas or slices, and they also have the ability to offer their resources to other tenants. This functionality empowers tenants to function both as providers and consumers, operating in both vendor and consumer modes

* [Multiprovider](/docs/custom_resources.md#multiprovider): By accommodating the collaboration of diverse providers, EdgeNet encourages numerous entities to contribute to nodes, thus fostering a rich and expansive ecosystem that thrives on heterogeneity. With the power of multitenancy, contributors with different hardware can easily lend their hardware.

* [Selective deployment](/docs/custom_resources.md#selective-deployment): While the involvement of multiple providers in EdgeNet extends beyond hardware vending, the possibilities encompass a broader spectrum. Node contributions can originate from individuals across the globe, and by leveraging a selective deployment mechanism, EdgeNet empowers the targeted deployment of resources to specific geographical regions, thereby augmenting localization capabilities and enabling efficient utilization of computing power where it is most needed.

* [Federation support](/docs/custom_resources.md#cluster-federation): EdgeNet envisions the federation of Kubernetes clusters worldwide, starting from the edge. By granting clusters the ability to assume worker or federation roles, EdgeNet enables the outsourcing of workloads to these clusters, fostering a seamless and globally interconnected network of distributed computing resources.

# Tutorials and Documentation
If you are planning to use EdgeNet software in your Kubernetes cluster, we highly encourage you to check out the [EdgeNet's documentation](/docs/README.md).

You can access all of [EdgeNet's tutorials](./docs/README.md#tutorials) with the main documentation or by navigating to the `doc` folder in the main repository.

# Support

To chat with a member of the EdgeNet team live, please [open our tawk.to window](https://tawk.to/edgenet).

# Contributing

The EdgeNet software is free and open source, licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0); we invite you to contribute. You can access [contribution guide](/docs/guides/contribution_guides.md) for more information on how to contribute.
