# Multitenancy

Some Kubernetes users need it to support multitenancy. A good example is a large organization with multiple teams, each of which is going to deploy services to a shared cluster. The organization doesn’t want one team’s work to interfere with another’s. EdgeNet presents a similar challenge, except that the teams that share a single EdgeNet cluster come from entirely separate organizations, so there can be no assumption of everyone being bound by common policies, aside from those that they explicitly agree to when joining and using EdgeNet. Because EdgeNet is also multiprovider-based, the need for tenant accountability is particularly acute: individuals and institutions will only provide nodes if they can trust the diverse actors that come from a multitenant environment.

EdgeNet’s multitenancy extensions are built on top of the Kubernetes notions of [namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) and [resource quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/). Namespaces provide isolation: users who are deploying services in one namespace do not see and cannot touch services that are deployed by other users in other namespaces. Resource quotas are a means to ensure that the overall resources of the cluster are shared out amongst the namespaces, so that, ideally, users in each namespace have sufficient resources in which to conduct their work.

Using namespaces and resource quotas is a perfectly classic approach to handling Kubernetes multitenancy, and so is EdgeNet’s adoption of a hierarchical naming convention for the namespaces. Currently, the hierarchy is not limited and a tenant may choose to create any number of subnamespaces they deserve.

With this description, the elements of EdgeNet that make possible multitenancy will be presented.

## Tenant

Multitenancy is a standard feature of the three well-known cloud service models; SaaS (Software as a Service), PaaS (Platform as a Service), and IaaS (Infrastructure as a Service). Hence, a tenant is a customer of a multi-tenant cluster where there is no trust in between. In the EdgeNet context, a tenant can operate in two modes, vendor and consumer. In vendor mode, the tenant is allowed to resell its resources to other tenants. 

To create a tenant in EdgeNet it s required to create a tenant request 

Below a tenant's open API schema is presented.

```yaml
openAPIV3Schema:
    type: object
    properties:
    spec:
        type: object
        required:
        - fullname
        - shortname
        - url
        - address
        - contact
        - enabled
        properties:
        fullname:
            type: string
        shortname:
            type: string
        url:
            type: string
        address:
            type: object
            required:
            - street
            - zip
            - city
            - country
            properties:
            street:
                type: string
            zip:
                type: string
            city:
                type: string
            region:
                type: string
                description: region or state
            country:
                type: string
        contact:
            type: object
            required:
            - firstname
            - lastname
            - email
            - phone
            properties:
            firstname:
                type: string
            lastname:
                type: string
            email:
                type: string
            phone:
                type: string
        clusternetworkpolicy:
            type: boolean
            default: false
        enabled:
            type: boolean
    status:
        type: object
        properties:
        state:
            type: string
        message:
            type: string
```

## Tenant Request
To create a tenant in EdgeNet it is required to first create a tenant request. The request should contain the tenant and admin of the tenant's contact information. To create a tenant we recommend to follow the tenant registration tutorial. The open API scheme is given below as a yaml file.

Note that the admission control mechanism prevents `tenant requests` to be created with the field `approved: true`.

```yaml
openAPIV3Schema:
    type: object
    properties:
    spec:
        type: object
        required:
        - fullname
        - shortname
        - url
        - address
        - contact
        properties:
        fullname:
            type: string
        shortname:
            type: string
        url:
            type: string
        address:
            type: object
            required:
            - street
            - zip
            - city
            - country
            properties:
            street:
                type: string
            zip:
                type: string
            city:
                type: string
            region:
                type: string
                description: region or state
            country:
                type: string
        contact:
            type: object
            required:
            - firstname
            - lastname
            - email
            - phone
            properties:
            firstname:
                type: string
            lastname:
                type: string
            email:
                type: string
            phone:
                type: string
        clusternetworkpolicy:
            type: boolean
            default: true
        resourceallocation:
            type: object
            x-kubernetes-preserve-unknown-fields: true
        approved:
            type: boolean
    status:
        type: object
        properties:
        expiry:
            type: string
            format: dateTime
            nullable: true
        state:
            type: string
        message:
            type: string
        notified:
            type: boolean
            default: false
```

## Tenant Resource Quota

To prevent starvation or excessive use it is beneficial to put resource quotas on tenants. These resources are standard Kubernetes resources. The resource quota contains two fields for either claiming a resource or dropping it off. The expiration dates can also be defined. The open API scheme is given below as a yaml file.

```yaml
openAPIV3Schema:
    type: object
    properties:
    spec:
        type: object
        required:
        - claim
        properties:
        claim:
            type: object
            x-kubernetes-preserve-unknown-fields: true 
#           # example
#           my-claim:
#               resourceList:
#                   cpu: 100m # Corresponds to 1 cpu core
#                   memory: 2GiB
#                   storage: 10 GiB
#                   ephemeral-storage: 5GiB
#               expiry: "2023-06-01 11:20:49.431444" # No expiration if not specified 
        drop:
            type: object
            x-kubernetes-preserve-unknown-fields: true
    status:
        type: object
        properties:
        state:
            type: string
        message:
            type: string
```

## Subnamespace

The subnamespace object in Kubernetes serves as a mechanism to emulate hierarchical namespaces within the flat namespace structure. Upon approval of a tenant request, a subnamespace is dynamically generated in tandem with the tenant. This subnamespace, referred to as the core namespace, bears the same name as the tenant.

In addition to the core namespace creation, tenants are empowered to define the resources that should be propagated to the subnamespace. This includes a range of Kubernetes objects such as `network policies`, `rbacs`, `limit ranges`, `secrets`, `config maps`, and `service accounts`. By setting the corresponding property value to true, tenants can selectively share these Kubernetes objects with the subnamespace, enabling seamless resource access and utilization within the tenant's environment.

When the scope of a subnamespace definition is set to "federation" instead of the default value "local," EdgeNet provides support for selective deployments to be deployed from other clusters within the same tenant's environment. This means that EdgeNet can accept targeted deployments originating from other clusters associated with the tenant.

The sync field within the subnamespace definition allows for the synchronization of the subnamespace with its child subnamespaces. By enabling this synchronization, changes, and updates made to the subnamespace are propagated to its children, ensuring consistency and coherence across the hierarchical structure.

The owner field in the subnamespace contains relevant information about the owner of the subnamespace, providing ownership details.

Moreover, a subtenant is assigned to the subnamespace, with comprehensive information including contact details, resource allocation, and slice specifications. This facilitates effective management and collaboration within the subnamespace environment.

Lastly, an expiration date can be specified for the subnamespace. If this date is not null, upon reaching the expiration date, the subnamespace undergoes a cleanup process, where all associated resources are deallocated and returned to the parent subnamespace.


```yaml
openAPIV3Schema:
    type: object
    properties:
    spec:
        type: object
        properties:
        workspace:
            type: object
            properties:
            resourceallocation:
                type: object
                x-kubernetes-preserve-unknown-fields: true
#           # example
#           resourceList:
#               cpu: 100m # Corresponds to 1 cpu core
#               memory: 2GiB
#               storage: 10 GiB
#               ephemeral-storage: 5GiB
            inheritance:
                type: object
                properties:
                rbac: 
                    type: boolean
                    default: true
                networkpolicy:
                    type: boolean
                    default: true
                limitrange:
                    type: boolean
                    default: true
                secret:
                    type: boolean
                    default: false
                configmap:
                    type: boolean
                    default: false
                serviceaccount:
                    type: boolean
                    default: false
            scope:
                type: string
                default: "local"
            sync:
                type: boolean
                default: true
            owner:
                type: object
                required:
                - email
                nullable: true
                properties:
                firstname:
                    type: string
                lastname:
                    type: string
                email:
                    type: string
                phone:
                    type: string
            sliceclaim:
                type: string
                nullable: true
        subtenant:
            type: object
            properties:
            resourceallocation:
                type: object
                x-kubernetes-preserve-unknown-fields: true
#               # example
#               resourceList:
#                   cpu: 100m # Corresponds to 1 cpu core
#                   memory: 2GiB
#                   storage: 10 GiB
#                   ephemeral-storage: 5GiB
            owner:
                type: object
                required:
                - firstname
                - lastname
                - email
                - phone
                properties:
                firstname:
                    type: string
                lastname:
                    type: string
                email:
                    type: string
                phone:
                    type: string
            sliceclaim:
                type: string
                nullable: true
        expiry:
            type: string
            format: dateTime
            nullable: true 
    status:
        type: object
        properties:
        state:
            type: string
        message:
            type: string
```

## Slice

Slice describes the repartitioning of resources. A slice object claims certain types of resources such as memory, CPU or disk, or a node until the expiration.

## Slice Claim

Slice Claim represents a request for certain resources that a tenant describes. It also contains a node selector where the inapplicable or unwanted nodes can be filtered out before granting the slice to the tenant.

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

## Manager Cache

## Cluster