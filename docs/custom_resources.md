# Multitenancy

Multitenancy is defined in the cloud software context as a single instance of a program serving multiple customers. Each customer is called a *tenant*. These tenants operate in a shared environment.

EdgeNet aims to provide this functionality by adding *tenant* custom resources. CRDs related to multitenancy are explained below.

## Tenant

A tenant is a customer of a multi-tenant cluster. In a shared environment, one or more tenants operate on a multi-tenant cluster. 

In EdgeNet's context, Tenants represent organizations or institutions. They own certain resources which they distribute by creating and assigning resources to subnamespaces. 

## Tenant Resource Quota

Each tenant owns a set of resources. These resources can be CPU, memory, bandwidth, and disk space. The quotas of the tenants are determined by this CRD.

## Subnamespace

A subnamespace is a special type of namespace that allows users of a tenant to create an arbitrary amount of nested namespaces. It allows the inheritance of RBAC and network policy configurations.

## Slice

Slice describes the repartitioning of resources. A slice object claims certain types of resources such as memory, CPU or disk, or a node until the expiration.

## Slice Claim

Slice Claim represents a request for certain resources that a tenant describes. It also contains a node selector where the inapplicable or unwanted nodes can be filtered out before granting the slice to the tenant.

## Tenant Request

Currently, the tenant request represents an organization with a person as a contact detail who wants to join the EdgeNet cluster. The request should contain the contact information of the tenant as well as the address of the establishment or research institution. 

To create a tenant request in EdgeNet refer to the [tutorial](tenant_registration.md).

## Role Request

Roles are composed of operations or actions that the holder can perform. These roles do not have cluster-wide capabilities instead they are only effective on subnamespaces. Role requests of the user with the associated user email address, are sent to the tenant of the user. If the tenant accepts the action space of the user changes accordingly. 

## Cluster Role Request

Cluster roles are composed of operations or actions that the holder can perform. These roles have cluster-wide capabilities. Cluster role requests of the user with the associated user email address, are sent to the EdgeNet administrators. If EdgeNet admins accept the action space of the user changes accordingly. 

# Multiprovider

A multi-provider environment differs from conventional single-provider environments by the number of vendors. There are more than more providers providing node resources to the cluster.

With EdgeNet, it is possible to easily contribute a node to the cluster.

## Node Contribution

EdgeNet is designed in such a way that it allows multiple providers to contribute their nodes in a single cluster. This information is held in the cluster by node contribution objects.

When a new node is added to the cluster which is done by a bootstrap script, it is necessary to configure connection settings. Node Contribution objects are used for setting up the ssh communication channel.

## VPN Peer

EdgeNet nodes are distributed around the world. To connect these nodes in the same network a VPN connection is used. Additionally, VPN is used for overcoming the limitations of NAT.
  

# Locations-Based Node Selection

Edge computing requires low latency. This is why the physical location of the nodes that are running the programs is extremely important. EdgeNet provides this mechanism by allowing location-based deployment.

## Selective Deployment

Selective deployment as the name suggests allows deployments to be run in nodes where the geographic information is specified.

# Cluster Federation

EdgeNet allows the federation of multiple clusters to share and outsource workloads. Currently, this feature is in alpha and hasn't yet been fully implemented.

## Selective Deployment Anchor

When a workload is scheduled to be outsourced, the cluster sends the `selective deployment` information to the federation cluster. This creates a `federation selective deployment anchor` on the federation cluster to indicate the `selective deployment` to be scheduled on the new working cluster.