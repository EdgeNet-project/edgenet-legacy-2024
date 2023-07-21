# Multitenancy
EdgeNet enables the utilization of a shared cluster by multiple tenants who lack trust in each other. Tenants can allocate resource quotas or slices, and they also have the ability to offer their resources to other tenants. This functionality empowers tenants to function both as providers and consumers, operating in both vendor and consumer modes.

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
#           claim:
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
#           # example
#           resourceList:
#               cpu: 100m # Corresponds to 1 cpu core
#               memory: 2GiB
#               storage: 10 GiB
#               ephemeral-storage: 5GiB
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
By accommodating the collaboration of diverse providers, EdgeNet encourages numerous entities to contribute to nodes, thus fostering a rich and expansive ecosystem that thrives on heterogeneity. With the power of multitenancy, contributors with different hardware can easily lend their hardware.

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

# Location-Based Node Selection
While the involvement of multiple providers in EdgeNet extends beyond hardware vending, the possibilities encompass a broader spectrum. Node contributions can originate from individuals across the globe, and by leveraging a selective deployment mechanism, EdgeNet empowers the targeted deployment of resources to specific geographical regions, thereby augmenting localization capabilities and enabling efficient utilization of computing power where it is most needed.

## Selective Deployment

Selective deployment, as the name suggests, enables the execution of deployments on specific nodes based on specified geographic information. Alongside its federation support, EdgeNet allows the outsourcing of workloads declared with selective deployments, further enhancing its capabilities.

The selective deployment feature primarily consists of two key fields. The first field is the "workloads" field, which encompasses workload definitions that adhere to regular Kubernetes standards. Currently, the supported workloads include deployments, daemonsets, statefulsets, jobs, and cronjobs. This enables users to employ a variety of workload types when leveraging selective deployment in EdgeNet.

The second field is the "selector," which offers flexibility in specifying the geographic criteria for deployment. Users can select geographic areas based on various parameters such as city, country, state, continent, or define a custom area using a polygon selector. Additionally, the "operator" and "quantity" fields provide further control, allowing users to specify whether to include or exclude nodes based on their geographic information.

By utilizing the selective deployment feature in EdgeNet, users gain the ability to strategically deploy their workloads to specific geographic locations, optimizing performance, data locality, and resource utilization as per their specific requirements.

```yaml
APIV3Schema:
  type: object
  properties:
    spec:
      type: object
      required:
        - workloads
        - selector
      properties:
        workloads:
          type: object
          properties:
            deployment:
              type: array
              items:
                type: object
                x-kubernetes-embedded-resource: true
                x-kubernetes-preserve-unknown-fields: true
              nullable: true
            daemonset:
              type: array
              items:
                type: object
                x-kubernetes-embedded-resource: true
                x-kubernetes-preserve-unknown-fields: true
              nullable: true
            statefulset:
              type: array
              items:
                type: object
                x-kubernetes-embedded-resource: true
                x-kubernetes-preserve-unknown-fields: true
              nullable: true
            job:
              type: array
              items:
                type: object
                x-kubernetes-embedded-resource: true
                x-kubernetes-preserve-unknown-fields: true
              nullable: true
            cronjob:
              type: array
              items:
                type: object
                x-kubernetes-embedded-resource: true
                x-kubernetes-preserve-unknown-fields: true
              nullable: true
        selector:
          type: array
          items:
            type: object
            properties:
              name:
                type: string
                enum:
                  - City
                  - State
                  - Country
                  - Continent
                  - Polygon
              value:
                type: array
                items:
                  type: string
              operator:
                type: string
                enum:
                  - In
                  - NotIn
              quantity:
                type: integer
                description: The count of nodes that will be picked for this selector.
                minimum: 1
                nullable: true
          minimum: 1
        recovery:
          type: boolean
    status:
      type: object
      properties:
        ready:
          type: string
        state:
          type: string
        message:
          type: array
          items:
            type: stringdo
```

# Cluster Federation
EdgeNet envisions the federation of Kubernetes clusters worldwide, starting from the edge. By granting clusters the ability to assume workload or federation roles, EdgeNet enables the outsourcing of workloads to these clusters, fostering a seamless and globally interconnected network of distributed computing resources.

## Selective Deployment Anchor

When a workload is designated for outsourcing, the originating cluster transmits the relevant information of the selective deployment to the federation cluster. This action triggers the creation of a "federation selective deployment anchor" on the federation cluster. The purpose of this anchor is to serve as a reference point, indicating the specific selective deployment that needs to be scheduled on the new working cluster.

The "federation selective deployment anchor" object includes several important fields to facilitate its functionality within the EdgeNet federation framework.

Firstly, the object contains an "origin reference" field that serves as a pointer to the original selective deployment object. This reference establishes the connection between the federation selective deployment anchor and its corresponding selective deployment, allowing for proper tracking and management of the outsourced workload.

Additionally, the "federation selective deployment anchor" utilizes the "cluster affinity" field, which contains selectors for both the cluster and workload cluster. These selectors define the criteria for identifying eligible clusters within the federation. The cluster selector determines the suitable federation cluster to host the selective deployment, while the workload cluster selector identifies the specific working cluster where the selective deployment should be scheduled.

By leveraging the origin reference and cluster affinity fields, the federation selective deployment anchor maintains the necessary associations and criteria to effectively orchestrate the outsourcing of workloads across the EdgeNet federation.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      properties:
        originRef:
          type: object
          properties:
            uuid:
              type: string
            namespace:
              type: string
            name:
              type: string
        clusterAffinity:
          type: object
          properties:
            matchExpressions:
              type: array
              items:
                type: object
                properties:
                  key:
                    type: string
                  operator:
                    enum:
                      - In
                      - NotIn
                      - Exists
                      - DoesNotExist
                    type: string
                  values:
                    type: array
                    items:
                      type: string
                      pattern: "^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$"
            matchLabels:
              x-kubernetes-preserve-unknown-fields: true
        federationManagerNames:
          type: array
          items:
            type: string
        secretName:
          type: string
    status:
      type: object
      properties:
        state:
          type: string
        message:
          type: string
```

## Manager Cache

The "Manager cache" plays a crucial role in propagating resource information across federation clusters within the EdgeNet system. It serves as a mechanism to exchange and synchronize vital data regarding the available resources in different working clusters.

The manager cache object comprises several key components. Firstly, it contains "hierarchy information" that provides details about the position of the federation cluster within the broader multi-clustered universe. This hierarchy information helps establish the relationships and structure between federation clusters, enabling efficient resource management and workload distribution.

Furthermore, the manager cache holds "available resource information" of other working clusters within the federation. This information gives insights into the resources, such as compute capacity, storage, and networking capabilities, present in each working cluster. By maintaining an up-to-date record of available resources, the federation cluster can make informed decisions when scheduling and distributing workloads.
Regular synchronization is performed between federation clusters to keep the manager cache objects consistent and accurate. As a result, the manager cache includes a "last update time" field indicating when the synchronization occurred, ensuring that the cached resource information remains current.

Lastly, the manager cache features an "enabled" field that can be toggled to enable or disable the scheduling of workloads on the federation cluster. This capability provides flexibility and control, allowing administrators to selectively enable or disable a federation cluster based on specific requirements or maintenance activities.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      properties:
        hierarchy:
          type: object
          properties:
            level: 
              type: integer
              minimum: 0
            parent:
              type: string
        cluster:
          type: array
          items:
            type: object
            properties:
              characteristic:
                type: array
                items:
                  type: string
              resourceAvailability:
                enum:
                  - Abundance
                  - Normal
                  - Limited
                  - Scarcity
                type: string
    status:
      type: object
      properties:
        state:
          type: string
        message:
          type: string
        updatedAt:
          type: string
```

## Cluster

In EdgeNet, a cluster object serves as a representation of a peer, workload, or manager cluster, each associated with its own set of definitions. Additionally, the cluster object incorporates cluster preferences, which determine the permissions granted or denied to specific tenants. The `visibility` and `enabled` fields further enable the modification of visibility and scheduling capabilities.

To establish a connection with the Kubernetes API server of the adjacent cluster, a secret is created, aligning with a corresponding service account in said cluster. The cluster object conveniently stores the name of this secret, facilitating access to the adjacent cluster and its resources.

```yaml
openAPIV3Schema:
  type: object
  properties:
    spec:
      type: object
      properties:
        uuid:
          type: string
        role:
          type: string
          enum:
          - Workload
          - Federation
        server:
          type: string
        preferences:
          type: object
          properties:
            allowlist:
              type: object
              properties:
                matchExpressions:
                  type: array
                  items:
                    type: object
                    properties:
                      key:
                        type: string
                      operator:
                        enum:
                          - In
                          - NotIn
                          - Exists
                          - DoesNotExist
                        type: string
                      values:
                        type: array
                        items:
                          type: string
                          pattern: "^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$"
                matchLabels:
                  x-kubernetes-preserve-unknown-fields: true
            denylist:
              type: object
              properties:
                matchExpressions:
                  type: array
                  items:
                    type: object
                    properties:
                      key:
                        type: string
                      operator:
                        enum:
                          - In
                          - NotIn
                          - Exists
                          - DoesNotExist
                        type: string
                      values:
                        type: array
                        items:
                          type: string
                          pattern: "^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$"
                matchLabels:
                  x-kubernetes-preserve-unknown-fields: true
        visibility:
          type: string
          enum:
          - Private
          # - Protected
          - Public
        secretName:
          type: string
    status:
      type: object
      properties:
        state:
          type: string
        message:
          type: string
```