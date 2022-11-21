# EdgeNet Documentation for Custom Resources

## Multitenancy

Multitenancy means in cloud computing context, multiple customers of a provider in a cluster shares resources. This is a difficult task since generally the programs of different customers runs on same physical machines. EdgeNet aims to provide multitenancy by adding custom resources. They are explained below.

### Tenant 
A tenant in EdgeNet is a party that owns certain resources and has users who have access to these resources. Tenants give roles to users that enable them to restrict or allow access to certain operations. The users may access different subnamespaces depending on their roles.

A tenant also holds the institutions' addresses and the administrator's contact data.

### Tenant Resource Quota
In EdgeNet each tenant owns a set of resources. This means tenants shares resources in a multi-tenant cluster. each tenant resource quota describes a set of resources to be added and removed from the access of the tenant.

### Subnamespace
Subnamespace is a special type of namespace that allows users of a tenant to create an arbitrary amount of nested namespaces. It allows the inheritance of RBAC and network policy configurations.

### Slice
Slice describes the repartitioning of resources. A slice object claims certain types of resources such as memory, CPU or disk, or a node until the expiration.

### Slice Claim
Slice claim is the request of a tenant for described slices. It also contains a node selector where the inapplicable or unwanted nodes can be filtered out before granting the slice to the tenant.

### Tenant Request
Tenant request is sent to any user with the aim of joining the cluster for research purposes. The request should contain the contact information of the tenant as well as the address of the establishment or research institution. 

To create a tenant request in EdgeNet refer to the [tutorial](tenant_registration.md).

### Role Request
Roles are composed of operations or actions that the holder can perform. These roles do not have cluster-wide capabilities instead they are only effective on subnamespaces. Role requests of the user with the associated user email address, are sent to the tenant of the user. If the tenant accepts the action space of the user changes accordingly. 

### Cluster Role Request
Cluster roles are composed of operations or actions that the holder can perform. These roles have cluster-wide capabilities. Cluster role requests of the user with the associated user email address, are sent to the EdgeNet administrators. If EdgeNet admins accept the action space of the user changes accordingly. 

---

## Multi-provider

In a multi-provider environment different from conventional single-provider environments, there are multiple vendors for resources to be used in the cluster. In EdgeNet it is possible to easily add a node to the cluster.

### Node Contribution
EdgeNet is designed in such a way that it allows multiple providers to contribute their own nodes in a single cluster. This information is held in cluster by node contribution objects.

When a new node is added to the cluster which is done by a bootstrap script, it is necessary to configure connection settings. Node Contribution objects are used for setting up the ssh communication channel.

### VPN Peer
EdgeNet nodes are distributed around the world. To connect these nodes in the same network a VPN connection is used. Additionally, VPN is used for overcoming the limitations od NAT.

---

## Location-based node selection

Majority of the use cases of edge computing require low latency. This is why physical location of the nodes that are running the programs are extreemly important. EdgeNet provides this meachanism by allowing location based deployment.

### Selective Deployment
Selective deployment as the name suggests allows deployments to be run in nodes where the geographic information is specified.


