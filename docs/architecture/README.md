# EdgeNet's Architecture Documentation

Following image is the architecture diagram of the EdgeNet. It shows how multitenancy and slices functions. 

![EdgeNet Architecture Diagram](/docs/architecture/architecture.png)

## Edge Computing
With cloud computing, providers can offer their computational resources to clients and bill according to their usage. One of the key concepts of cloud computing is to allow a pay-as-you-go model. With the maintenance burden of the hardware on some of the software stack being handled by the provider, clients can only use the services the provider offers and cut expenses. 

Different service models are offered by different cloud providers such as [IaaS](https://www.ibm.com/topics/iaas) (Infrastructure as a Service), [PaaS](https://www.ibm.com/topics/paas) (Platform as a Service), and [SaaS](https://www.ibm.com/topics/saas) (Software as a Service). There are also other alternatives for more specific cases. For instance, in recent years it has been seen that the overhead of creating VMs is not tolerable for edge computing cases. Thus a new term [CaaS](https://www.redhat.com/en/topics/cloud-computing/what-is-caas) (Container as a Service) is started to be used in the industry. 

CaaS enabled lower overhead when creating and running workloads since container technology does not require a hypervisor and is more agile than VMs. This is why EdgeNet, which has been designed to be compatible with edge environments, is developed with the CaaS service model in mind. So it can run on the edge and in the cloud with lower overhead.

## Multi-Tenancy
Generally, the need for multi-tenancy arises when more than one user wants to use the service that is offered by the provider at the same time. To make sure the tenants do not harm others they need to be isolated from each other. Different isolation techniques are used to enable multi-tenancy by different projects. For example, Virtual Kubelet based frameworks, such as [Liqo](https://github.com/liqotech/liqo), [Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet), and [tensile-kube](https://github.com/virtual-kubelet/tensile-kube) enables multi-tenancy by creating multiple separate clusters. Other's such as [Virtual Cluster](https://github.com/kubernetes-sigs/cluster-api-provider-nested/tree/main/virtualcluster), [k3v](https://github.com/ibuildthecloud/k3v), [vcluster](https://github.com/loft-sh/vcluster), and [Kamaji](https://github.com/clastix/kamaji) runs a separate control plane for each tenant. EdgeNet's multi-tenancy approach is to have only one control plane and separate the tenants logically. This approach is also named single-instance native. Other projects enable this kind of multi-tenancy such as [HNC](https://github.com/kubernetes-sigs/hierarchical-namespaces), [Capsule](https://github.com/clastix/capsule), [kiosk](https://github.com/loft-sh/kiosk), and [Arktos](https://github.com/CentaurusInfra/arktos).

In EdgeNet a tenant is the fundamental entity that can manipulate workloads.

## Consumer and Vendor Tenancy
In general cloud services support two types of tenancy. The first is called Consumer Mode, in which the tenant is the user. It can create, delete, and update workloads. The second is called Vendor Mode. In this mode, the tenant can resell the access to the resources to others.

EdgeNet supports both of these modes of tenancy. So that tenants can resell their resources and use them at the same time.

## Tenant Resource Quota
To bill their customers and prevent excessive use, cloud providers put resource quotas on their tenants and limit their usage. In the Kubernetes, resource quotas can be put on namespaces. EdgeNet also supports hierarchical namespaces. Combining these two EdgeNet allows tenant's resources to be propagated hierarchically.

The [HNC](https://github.com/kubernetes-sigs/hierarchical-namespaces) (Hierarchical Namespace Controller) project also implements this functionality however, there is no requirement for a quota to be attributed to each namespace. Since it creates logical problems in multi-tenant environments, EdgeNet makes it compulsatory to assign resource quotas to namespaces.

## Variable Slice Granularity
Slicing in EdgeNet context, refers to the allocation of a larger pool of resources into smaller portions. Each portion is exclusively assigned to a tenant. This resource is generally a node in the cluster exclusively designated for a tenant. This is called Node-level-slicing. However, in some cases, a node might be too large to be used efficiently. For instance, a node can be underutilized by a tenant where the whole node is allocated. For such cases, EdgeNet implements Sub-node-level-slicing. Which exclusively divides and allocates resources for tenants. 

EdgeNet implements an automatic mechanism to create slices for tenants. These slices can be at the node level or sub-node level.

## Kubernetes Custom Resources
EdgeNet extends the Kubernetes API server instead of modifying a fork of it. In this way, the EdgeNet can work with different versions of Kubernetes and the repository doesn't need to be updated for every change in Kubernetes' main repository. This is done by having CRDs (Custom Resource Definitions) and custom controllers. CRDs in Kubernetes are the most straightforward methods of adding extra functionalities.

