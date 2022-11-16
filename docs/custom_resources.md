# EdgeNet Documentationfor Custom Resources
In EdgeNet there are 11 custom resources. Each of them will be explained here.

## Cluster Role Request

## Node Contribution
EdgeNet allows institutions and individuals from every part of the world to contribute nodes to the global cluster. When a new node is added to the cluster which is done by a bootstrap script, it is necessary to configure connection settings. Node Contribution objects are used for setting up the ssh communication channel.

## Role Request

## Selective Deployment
Selective deployment as the name suggests allows deployments to be run in nodes where the geographic information is specified. 

## Slice

## Slice Claim

## Subnamespace
Subnamespace is a special type of namespace that allows users of a Tenant to create arbitrary amount of nested namespaces which allows the inheritance of RBAC and network policy configurations.

## Tenant
Tenant in EdgeNet is a party that occupies certain resources and able to have users attached to it. The users have access to the same subnamespace and inherits the same set of roles with the aid of [subnamespaces](#subnamespace).

## Tenant Request
Tenant request is sent to by any user for the aim of joining the cluster for reseach purposes. The request should contain contact information of the Tenant as well as the address of the establishment or research institution. 

To create a tenant request in EdgeNet refer to the [tutorial](tenant_registration.md).

## Tenant Resource Quota

## VPN Peer