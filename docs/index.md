# Edgenet Documentation

## What is EdgeNet?
EdgeNet, the open-source globally distributed edge cloud for Internet researchers, is based on industry-standard Cloud software, using [Docker](https://www.docker.com/) for containerization and [Kubernetes](https://kubernetes.io/) for deployment and node management. The source code of EdgeNet can be found in [here](https://github.com/EdgeNet-project/edgenet). 

Additionally, we provide a testbed for all researchers. To open an account please refer to the [Registering a tenant into EdgeNet](tenant_registration.md). For getting more info about the project visit the official [EdgeNet-Project](https://www.edge-net.org/) website.

## Components of EdgeNet
EdgeNet adds [custom resource definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (CRDs) to Kubernetes to extend its capabilities to edge computing. Its adds the folowing list of components as CRDs:
* Cluster Role Request
* Node Contribution
* Role Request
* Selective Deployment
* Slice
* Slice Claim
* Subnamespace
* Tenant
* Tenant Request
* Tenant Resource Quota
* VPN Peer

<!-- Individual explainations of the CRDs can be found [here](custom_resources.md). -->

EdgeNet also provides custom [controllers](https://kubernetes.io/docs/concepts/architecture/controller/) for these resources. These controllers checks the states of the CRDs in a loop and tries to make it closer to their specs. In kubernetes world status represents the current state of the objects and specs represents the desired states.

These controllers usually run inside the cluster and communicates with the kube-api server to fulfill certain functionalities.

## How to use EdgeNet?
EdgeNet is an open source kuber
To use EdgeNet, we have provided tutorials for different putposes. 

* [Create my own EdgeNet cluster](cluster_creation.md)
* [Registering a tenant into EdgeNet](tenant_registration.md)
* [Creating a node level slice for your tenant](slice_creation.md)
* [Subnamespace creation for your tenant](subnamespace_creation.md)
* [Configuring use permissions](user_permissions.md)
* [Role request](role_request.md)
* [Cluster role request](cluster_role_request.md)

