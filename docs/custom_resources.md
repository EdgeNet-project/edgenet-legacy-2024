# Multitenancy

Some Kubernetes users need it to support multitenancy. A good example is a large organization with multiple teams, each of which is going to deploy services to a shared cluster. The organization doesn’t want one team’s work to interfere with another’s. EdgeNet presents a similar challenge, except that the teams that share a single EdgeNet cluster come from entirely separate organizations, so there can be no assumption of everyone being bound by common policies, aside from those that they explicitly agree to when joining and using EdgeNet. Because EdgeNet is also multiprovider-based, the need for tenant accountability is particularly acute: individuals and institutions will only provide nodes if they can trust the diverse actors that come from a multitenant environment.

EdgeNet’s multitenancy extensions are built on top of the Kubernetes notions of [namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) and [resource quotas](https://kubernetes.io/docs/concepts/policy/resource-quotas/). Namespaces provide isolation: users who are deploying services in one namespace do not see and cannot touch services that are deployed by other users in other namespaces. Resource quotas are a means to ensure that the overall resources of the cluster are shared out amongst the namespaces, so that, ideally, users in each namespace have sufficient resources in which to conduct their work.

Using namespaces and resource quotas is a perfectly classic approach to handling Kubernetes multitenancy, and so is EdgeNet’s adoption of a hierarchical naming convention for the namespaces. Currently, the hierarchy is not limited and a tenant may choose to create any number of subnamespaces they deserve.

With this description, the elements of EdgeNet that make possible multitenancy will be presented.

## Tenant

Multitenancy is a standard feature of the three well-known cloud service models; SaaS (Software as a Service), PaaS (Platform as a Service), and IaaS (Infrastructure as a Service). Hence, a tenant is a customer of a multi-tenant cluster where there is no trust in between. In the EdgeNet context, a tenant can operate in two modes, vendor and consumer. In vendor mode, the tenant is allowed to resell its resources to other tenants. 

To create a tenant in EdgeNet it s required to create a tenant request 

Below a tenant's OpenAPI schema is presented.

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
To create a tenant in EdgeNet it is required to first create a tenant request. The request should contain the tenant and admin of the tenant's contact information. To create a tenant we recommend to follow the tenant registration tutorial. The OpenAPI scheme is given below as a yaml file.

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

To prevent starvation or excessive use it is beneficial to put resource quotas on tenants. These resources are standard Kubernetes resources. The resource quota contains two fields for either claiming a resource or dropping it off. The expiration dates can also be defined. The OpenAPI scheme is given below as a yaml file.

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

A slice in EdgeNet defines the distribution and allocation of resources. There are two types of slices available: node-level slices and resource slices.

Node-level slices reserve one or more nodes based on a node selector criteria and establish a subcluster dedicated to a specific tenant. This allows the tenant to have exclusive access to the reserved nodes within the subcluster.

On the other hand, resource slices allocate the specified resources to the tenant. These slices ensure that the tenant receives the designated amount of resources according to their requirements.

When a slice reaches its expiration, a one-minute grace period is provided to any workloads utilizing that particular slice. During this grace period, the workloads are given the opportunity to terminate gracefully and wrap up any ongoing operations.

After the grace period, if any workloads are still active, they are terminated in a controlled manner to ensure a smooth transition and proper resource cleanup. This ensures efficient resource management and allows for the timely release of resources associated with the expired slice.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - nodeselector
      properties:
        sliceclassname:
          type: string
          default: "Node"
          enum:
            - Node
            - Resource
        claimref:
          type: object
          x-kubernetes-embedded-resource: true
          x-kubernetes-preserve-unknown-fields: true
        nodeselector:
          type: object
          required:
            - selector
            - nodecount
            - resources
          properties:
            selector:
              type: object
              x-kubernetes-preserve-unknown-fields: true
            nodecount:
              type: integer
              minimum: 1
            resources:
              type: object
              x-kubernetes-preserve-unknown-fields: true
    status:
      type: object
      properties:
        state:
          type: string
        message:
          type: string
        expiry:
          type: string
          format: dateTime
          nullable: true
```

## Slice Claim

To create a slice in EdgeNet, the initial step involves submitting a slice request. This request is encapsulated within a slice claim, which contains all the necessary information for the creation of the desired slice. Below is a yaml file that outlines the OpenAPI specification for describing the slice claim.

```yaml
hema:
  type: object
  properties:
    spec:
      type: object
      required:
        - slicename
        - nodeselector
      properties:
        sliceclassname:
          type: string
          default: "Node"
          enum:
            - Node
            - Resource
        slicename:
          type: string
        nodeselector:
          type: object
          required:
            - selector
            - nodecount
            - resources
          properties:
            selector:
              type: object
              x-kubernetes-preserve-unknown-fields: true
            nodecount:
              type: integer
              minimum: 1
            resources:
              type: object
              x-kubernetes-preserve-unknown-fields: true
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

## Role Request

In the cluster, there exist two types of roles: cluster roles, which encompass cluster-wide roles, and normal roles, which pertain to roles specific to namespaces. These roles facilitate the assignment of user permissions and determine their accessibility to various resources within the cluster. For further information on role-based access control in Kubernetes, you can refer to the [role-based access documentation](https://kubernetes.io/docs/reference/access-authn-authz/rbac/).

EdgeNet introduces a request mechanism to create predefined roles, enhancing the role management capabilities. Below, you will find the OpenAPI specification of a role request object.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - email
        - roleref
      properties:
        firstname:
          type: string
        lastname:
          type: string
        email:
          type: string
          format: email
        roleref:
          type: object
          required:
          - kind
          - name
          properties:
            kind:
              type: string
              enum:
                - Role
                - ClusterRole
            name:
              type: string
              pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
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

## Cluster Role Request

As discussed above, cluster roles are roles that have cluster-wide access. Here an OpenAPI specification of the cluster role request is given.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - email
        - rolename
      properties:
        firstname:
          type: string
        lastname:
          type: string
        email:
          type: string
          format: email
        rolename:
          type: string
          pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
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

# Multiprovider

EdgeNet nodes are not all furnished out of a single data center, or even a small number of data centers, but rather from large numbers of individuals and institutions, who may be users of the system, or simply those wishing to contribute to the system. The “node contribution” extensions enable this.

## Node Contribution

When a new node is added to the cluster using a [bootstrap script](https://github.com/EdgeNet-project/node/blob/main/bootstrap.sh), it triggers the activation of a node contribution object. This object encompasses vital information regarding the node provider, SSH details, limitations, and user-related data. Below is the OpenAPI specification for the node contribution object in yaml format.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - host
        - port
        - enabled
      properties:
        tenant:
          type: string
          nullable: true
          pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
        host:
          type: string
        port:
          type: integer
          minimum: 1
        user:
          type: string
          default: edgenet
        enabled:
          type: boolean
        limitations:
          type: array
          nullable: true
          items:
            type: object
            properties:
              kind:
                type: string
                enum:
                  - Tenant
                  - Namespace
              identifier:
                type: string
                pattern: '[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*'
    status:
      type: object
      properties:
        state:
          type: string
        message:
          type: string
```

## VPN Peer

To facilitate the connectivity between EdgeNet nodes distributed worldwide, a Virtual Private Network (VPN) is employed. This VPN enables seamless communication and access among the nodes, thereby forming a connected network.

The VPN serves a dual purpose: it not only establishes connectivity between the nodes but also overcomes the limitations imposed by Network Address Translation (NAT). NAT can hinder direct communication between nodes with private IP addresses, but the VPN mitigates this issue by providing a secure and private communication channel.

In the process of setting up the nodes, a VPN peer is created using the [bootstrap script](https://github.com/EdgeNet-project/node/blob/main/bootstrap.sh). This script handles the configuration and establishment of the VPN peer, ensuring that each node can effectively communicate and interoperate within the EdgeNet cluster.

```yaml
penAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - addressV4
        - addressV6
        - publicKey
      properties:
        addressV4:
          type: string
          pattern: '^[0-9.]+$'
          description: The IPv4 address assigned to the node's VPN interface.
        addressV6:
          type: string
          pattern: '^[a-f0-9:]+$'
          description: The IPv6 address assigned to the node's VPN interface.
        endpointAddress:
          type: string
          pattern: '^[a-f0-9.:]+$'
          nullable: true
          description: The public IPv4/v6 address of the node. Required for NAT-NAT communications.
        endpointPort:
          type: integer
          minimum: 1
          nullable: true
          description: The port on which WireGuard is listening. Required for NAT-NAT communications.
        publicKey:
          type: string
          descripti
```

# Locations-Based Node Selection



## Selective Deployment

<!-- Selective deployment as the name suggests allows deployments to be run in nodes where the geographic information is specified. -->

# Cluster Federation

<!-- EdgeNet allows the federation of multiple clusters to share and outsource workloads. Currently, this feature is in alpha and hasn't yet been fully implemented. -->

## Selective Deployment Anchor

<!-- When a workload is scheduled to be outsourced, the cluster sends the `selective deployment` information to the federation cluster. This creates a `federation selective deployment anchor` on the federation cluster to indicate the `selective deployment` to be scheduled on the new working cluster. -->

## Manager Cache

## Cluster