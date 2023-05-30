# EdgeNet Documentation

## What is EdgeNet?
EdgeNet is an open-source edge cloud orchestration software extension that is built on top of industry-standard cloud software [Kubernetes](https://kubernetes.io/) and utilizes [Docker](https://www.docker.com/) for containerization. The source code can be found [here](https://github.com/EdgeNet-project/edgenet).

The EdgeNet software can be installed on any Kubernetes cluster. The documentation of EdgeNet software is distinct from the [EdgeNet Testbed](https://edge-net.org) which is a globally distributed edge cloud for Internet researchers. We encourage everybody to contribute a node to the testbed. To participate in the testbed and learn more please visit the [edgenet website](https://edge-net.org).

## EdgeNet's Features
### Edge Computing
With cloud computing, providers can offer their computational resources to clients and bill according to their usage. One of the key concepts of cloud computing is to allow a pay-as-you-go model. With the maintenance burden of the hardware on some of the software stack being handled by the provider, clients can only use the services the provider offers and cut expenses. 

Different service models are offered by different cloud providers such as [IaaS](https://www.ibm.com/topics/iaas) (Infrastructure as a Service), [PaaS](https://www.ibm.com/topics/paas) (Platform as a Service), and [SaaS](https://www.ibm.com/topics/saas) (Software as a Service). There are also other alternatives for more specific cases. For instance, in recent years it has been seen that the overhead of creating VMs is not tolerable for edge computing cases. Thus a new term [CaaS](https://www.redhat.com/en/topics/cloud-computing/what-is-caas) (Container as a Service) is started to be used in the industry. 

CaaS enabled lower overhead when creating and running workloads since container technology does not require a hypervisor and is more agile than VMs. This is why EdgeNet, which has been designed to be compatible with edge environments, is developed with the CaaS service model in mind. So it can run on the edge and in the cloud with lower overhead.

### Multi-Tenancy
Generally, the need for multi-tenancy arises when more than one user wants to use the service that is offered by the provider at the same time. To make sure the tenants do not harm others they need to be isolated from each other. Different isolation techniques are used to enable multi-tenancy by different projects. For example, Virtual Kubelet based frameworks, such as [Liqo](https://github.com/liqotech/liqo), [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet), and [tensile-kube](https://github.com/virtual-kubelet/tensile-kube) enables multi-tenancy by creating multiple separate clusters. Other's such as [Virtual Cluster](https://github.com/kubernetes-sigs/cluster-api-provider-nested/tree/main/virtualcluster), [k3v](https://github.com/ibuildthecloud/k3v), [vcluster](https://github.com/loft-sh/vcluster), and [Kamaji](https://github.com/clastix/kamaji) runs a separate control plane for each tenant. EdgeNet's multi-tenancy approach is to have only one control plane and separate the tenants logically. This approach is also named single-instance native. Other projects enable this kind of multi-tenancy such as [HNC](https://github.com/kubernetes-sigs/hierarchical-namespaces), [Capsule](https://github.com/clastix/capsule), [kiosk](https://github.com/loft-sh/kiosk), and [Arktos](https://github.com/CentaurusInfra/arktos).

In EdgeNet a tenant is the fundamental entity that can manipulate workloads.

### Consumer and Vendor Tenancy
In general cloud services support two types of tenancy. The first is called Consumer Mode, in which the tenant is the user. It can create, delete, and update workloads. The second is called Vendor Mode. In this mode, the tenant can resell the access to the resources to others.

EdgeNet supports both of these modes of tenancy. So that tenants can resell their resources and use them at the same time.

### Tenant Resource Quota
To bill their customers and prevent excessive use, cloud providers put resource quotas on their tenants and limit their usage. In the Kubernetes, resource quotas can be put on namespaces. EdgeNet also supports hierarchical namespaces. Combining these two EdgeNet allows tenant's resources to be propagated hierarchically.

The [HNC](https://github.com/kubernetes-sigs/hierarchical-namespaces) (Hierarchical Namespace Controller) project also implements this functionality however, there is no requirement for a quota to be attributed to each namespace. Since it creates logical problems in multi-tenant environments, EdgeNet makes it compulsatory to assign resource quotas to namespaces.

### Variable Slice Granularity
Slicing in EdgeNet context, refers to the allocation of a larger pool of resources into smaller portions. Each portion is exclusively assigned to a tenant. This resource is generally a node in the cluster exclusively designated for a tenant. This is called Node-level-slicing. However, in some cases, a node might be too large to be used efficiently. For instance, a node can be underutilized by a tenant where the whole node is allocated. For such cases, EdgeNet implements Sub-node-level-slicing. Which exclusively divides and allocates resources for tenants. 

EdgeNet implements an automatic mechanism to create slices for tenants. These slices can be at the node level or sub-node level.

### Kubernetes Custom Resources
EdgeNet extends the Kubernetes API server instead of modifying a fork of it. In this way, the EdgeNet can work with different versions of Kubernetes and the repository doesn't need to be updated for every change in Kubernetes' main repository. This is done by having CRDs (Custom Resource Definitions) and custom controllers. CRDs in Kubernetes are the most straightforward methods of adding extra functionalities. EdgeNet comes with different CRDs for functioning which will be explained in the next chapter. 

## Components of EdgeNet
As discussed to flexibly add functionalities to the Kubernetes API server without the burden of updating the codebase, EdgeNet introduces novel  [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). We have devised 4 major categories for EdgeNet and assigned the CRDs to their places. However, this classification does not fully differentiate CRDs. For instance, the Selective Deployment CRD is used in both federation and location-based node selection categories. 
Here are the categories and their associated CRDs:
* CRDs for multi-tenancy features
    * [Tenant](custom_resources.md#tenant)
    * [Tenant Resource Quota](custom_resources.md#tenant-resource-quota)
    * [Subnamespace](custom_resources.md#subnamespace)
    * [Slice](custom_resources.md#slice)
    * [Slice Claim](custom_resources.md#slice-claim)
    * [Tenant Request](custom_resources.md#tenant-request)
    * [Role Request](custom_resources.md#role-request)
    * [Cluster Role Request](custom_resources.md#cluster-role-request)
* CRDs for multi-provider features
    * [Node Contribution](custom_resources.md#node-contribution)
    * [VPN Peer](custom_resources.md#vpn-peer)
* CRDs for locations-based node selection features
    * [Selective Deployment](custom_resources.md#selective-deployment)
* CRDs for federation features
    * [Selective Deployment Anchor]()

EdgeNet also provides custom [controllers](https://kubernetes.io/docs/concepts/architecture/controller/) for these resources. These controllers check the states of the CRDs in a loop and try to make them closer to their specs. In the Kubernetes world, status represents the current state of the objects and specs represent the desired states.

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
pull requests, etc. You are in the right place for documentation-related contributions. You can start how open-source works from this [guide](https://opensource.guide/how-to-contribute/#how-to-submit-a-contribution).

Spelling fixes are most welcomed, as are contributions and edits to longer sections of the documentation.

For issues and pull requests please refer to open-source [contribution guidelines](contribution_guidelines.md). 

### Unit tests

To make sure the code works correctly it is important to have high-quality unit tests. [Here](unit_tests.md) you can find the guide for creating unit tests.

## Branches

* The `master` branch reflects the currently-deployed version of EdgeNet.
* The current `release` branch is where we prepare the next EdgeNet release. Please make your changes to this branch.