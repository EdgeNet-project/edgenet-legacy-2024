# EdgeNet Documentation

## What is EdgeNet?
EdgeNet is an open-source edge cloud orchestration software that is built on top of industry-standard cloud software [Kubernetes](https://kubernetes.io/) and utilizes [Docker](https://www.docker.com/) for containerization. The source code can be found [here](https://github.com/EdgeNet-project/edgenet).

In the EdgeNet Project, we provide [EdgeNet Testbed](https://edge-net.org) which is a globally distributed edge cloud for Internet researchers. We encourage everybody to contribute a node to the testbed. For more info please visit the [website](https://edge-net.org).

## Components of EdgeNet
EdgeNet adds [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRDs) to Kubernetes to extend its capabilities to edge computing. We have divided these components into 3 groups. Please refer to the following list of components:
* Multitenancy
    * [Tenant](custom_resources.md#tenant)
    * [Tenant Resource Quota](custom_resources.md#tenant-resource-quota)
    * [Subnamespace](custom_resources.md#subnamespace)
    * [Slice](custom_resources.md#slice)
    * [Slice Claim](custom_resources.md#slice-claim)
    * [Tenant Request](custom_resources.md#tenant-request)
    * [Role Request](custom_resources.md#role-request)
    * [Cluster Role Request](custom_resources.md#cluster-role-request)
* Multi-provider
    * [Node Contribution](custom_resources.md#node-contribution)
    * [VPN Peer](custom_resources.md#vpn-peer)
* Locations-based node selection
    * [Selective Deployment](custom_resources.md#selective-deployment)

EdgeNet also provides custom [controllers](https://kubernetes.io/docs/concepts/architecture/controller/) for these resources. These controllers check the states of the CRDs in a loop and try to make them closer to their specs. In Kubernetes world, status represents the current state of the objects and specs represent the desired states.

These controllers usually run inside the cluster and communicate with the kube-api server to fulfill certain functionalities.

## How to use EdgeNet?
EdgeNet is an open-source extension for Kubernetes clusters. You can install EdgeNet to your Kubernetes cluster and start using the functionalities. We have provided tutorials for different purposes below:

* [Deploy EdgeNet on a Kubernetes cluster](getting_started.md)
* [Registering a tenant](tenant_registration.md)
* [Creating a node-level slice for your tenant](slice_creation.md)
* [Subnamespace creation for your tenant](subnamespace_creation.md)
* [Configure user permissions](user_permissions.md)
* [Role request](role_request.md)
* [Cluster role request](cluster_role_request.md)

## Contributing

We welcome contributions to EdgeNet of any kind, including documentation, suggestions, bug reports,
pull requests, etc. You are in the right place for documentation-related contributions.
<!-- Also check out our [contribution guide](). --> 

Spelling fixes are most welcomed, as are contributions and edits to longer sections of the documentation.


### Unit tests

To make sure the code works correctly it is important to have high-quality unit tests. [Here](unit_tests.md) you can find the guide for creating unit tests.

## Branches

* The `master` branch reflects the currently-deployed version of EdgeNet.
* The current `release` branch is where we prepare the next EdgeNet release. Please make your changes to this branch.
