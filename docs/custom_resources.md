# EdgeNet Documentation for Custom Resources
In EdgeNet there are 11 custom resources. Each of them will be explained here.

## Cluster Role Request
Cluster roles are composed of operations or actions that the holder can perform. These roles have cluster-wide capabilities. Cluster role requests of the user with the associated user email address, are sent to the EdgeNet administrators. If EdgeNet admins accept the action space of the user changes accordingly. 

## Node Contribution
EdgeNet allows institutions and individuals from every part of the world to contribute nodes to the global cluster. When a new node is added to the cluster which is done by a bootstrap script, it is necessary to configure connection settings. Node Contribution objects are used for setting up the ssh communication channel.

## Role Request
Roles are composed of operations or actions that the holder can perform. These roles do not have cluster-wide capabilities instead they are only effective on subnamespaces. Role requests of the user with the associated user email address, are sent to the tenant of the user. If the tenant accepts the action space of the user changes accordingly. 

## Selective Deployment
Selective deployment as the name suggests allows deployments to be run in nodes where the geographic information is specified. 

## Slice
Slice describes the ownership of a tenant's resources. Each slice references resources such as memory, CPU or disk, or a node. Slices also have expiration dates.

## Slice Claim
Slice claim is the request of a tenant for described slices. It also contains a node selector where the inapplicable or unwanted nodes can be filtered out before granting the slice to the tenant.

## Subnamespace
Subnamespace is a special type of namespace that allows users of a tenant to create an arbitrary amount of nested namespaces which allows the inheritance of RBAC and network policy configurations.

## Tenant
A tenant in EdgeNet is a party that occupies certain resources and has users who have access to these resources. Tenants give roles to users that enable them to restrict or allow access to certain operations. The users may access different subnamespaces depending on their roles.

A tenant also holds the institutions' addresses and the administrator's contact data.

## Tenant Request
Tenant request is sent to any user with the aim of joining the cluster for research purposes. The request should contain the contact information of the tenant as well as the address of the establishment or research institution. 

To create a tenant request in EdgeNet refer to the [tutorial](tenant_registration.md).

## Tenant Resource Quota
In EdgeNet each tenant owns a set of resources. This means tenants shares resources in a multi-tenant cluster. each tenant resource quota describes a set of resources to be added and removed from the access of the tenant.

## VPN Peer
EdgeNet is a distributed edge cluster that has nodes all around the world. To connect these nodes in the same network a VPN is used. VPN peer objects contain information about different nodes IP addresses, endpoints, and public keys.