# EdgeNet Documentation

## What is EdgeNet?
EdgeNet, the open-source globally distributed edge cloud for Internet researchers, is based on industry-standard Cloud software, using [Docker](https://www.docker.com/) for containerization and [Kubernetes](https://kubernetes.io/) for deployment and node management. The source code of EdgeNet can be found [here](https://github.com/EdgeNet-project/edgenet). 

Additionally, we provide a testbed for all researchers. To open an account please refer to the [Registering a tenant into EdgeNet](tenant_registration.md). For getting more info about the project visit the official [EdgeNet-Project](https://www.edge-net.org/) website.

## Components of EdgeNet
EdgeNet adds [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRDs) to Kubernetes to extend its capabilities to edge computing. It adds the following list of components as CRDs:
* [Cluster Role Request](custom_resources.md#cluster-role-request)
* [Node Contribution](custom_resources.md#node-contribution)
* [Role Request](custom_resources.md#role-request)
* [Selective Deployment](custom_resources.md#selective-deployment)
* [Slice](custom_resources.md#slice)
* [Slice Claim](custom_resources.md#slice-claim)
* [Subnamespace](custom_resources.md#subnamespace)
* [Tenant](custom_resources.md#tenant)
* [Tenant Request](custom_resources.md#tenant-request)
* [Tenant Resource Quota](custom_resources.md#tenant-resource-quota)
* [VPN Peer](custom_resources.md#vpn-peer)

<!-- Individual explanations of the CRDs can be found [here](custom_resources.md). -->

EdgeNet also provides custom [controllers](https://kubernetes.io/docs/concepts/architecture/controller/) for these resources. These controllers check the states of the CRDs in a loop and try to make them closer to their specs. In Kubernetes world, status represents the current state of the objects and specs represent the desired states.

These controllers usually run inside the cluster and communicate with the kube-api server to fulfill certain functionalities.

## How to use EdgeNet?
EdgeNet is an open-source Kubernetes cluster extension. You can install EdgeNet to your local cluster for testing or use the global EdgeNet testbed. We have provided tutorials for different purposes below:

* [Create my own EdgeNet cluster](cluster_creation.md)
* [Registering a tenant into EdgeNet](tenant_registration.md)
* [Creating a node-level slice for your tenant](slice_creation.md)
* [Subnamespace creation for your tenant](subnamespace_creation.md)
* [Configuring use permissions](user_permissions.md)
* [Role request](role_request.md)
* [Cluster role request](cluster_role_request.md)

## Contributing

We welcome contributions to EdgeNet of any kind, including documentation, suggestions, bug reports,
pull requests, etc. You are in the right place for documentation-related contributions.
<!-- Also check out our [contribution guide](). --> 

Spelling fixes are most welcomed, as are contributions and edits to longer sections of the documentation.

## Branches

* The `master` branch reflects the currently-deployed version of EdgeNet.
* The current `release` branch is where we prepare the next EdgeNet release. Please make your changes to this branch.
