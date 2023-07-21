# Table of Contents
- [Table of Contents](#table-of-contents)
- [What is EdgeNet?](#what-is-edgenet)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Tutorials](#tutorials)
- [Components](#components)
    - [Fedmanctl](#fedmanctl)
- [Concepts](#concepts)
  - [Extending Kubernetes](#extending-kubernetes)
  - [Multitenancy](#multitenancy)
  - [Location-Based Node Selection](#location-based-node-selection)
  - [Federation of Multiple EdgeNet Clusters](#federation-of-multiple-edgenet-clusters)
- [Development and Contributing](#development-and-contributing)
    - [Contributing Guides](#contributing-guides)
    - [Architectural Design Proposals (ADRs)](#architectural-design-proposals-adrs)
    - [Unit Tests Guides](#unit-tests-guides)
    - [Branches](#branches)

# What is EdgeNet?
EdgeNet is a free and open-source cloud orchestration software extension that brings industry-standard cloud software [Kubernetes](https://kubernetes.io/) to the network edge.

It is important to distinguish EdgeNet software and EdgeNet Testbed. If you are a user of EdgeNet Testbed please refer to the [Testbed's website](https://edge-net.org). This documentation is intended for users and developers of EdgeNet software (will be referred to as simply EdgeNet).

The EdgeNet's source code is hosted in the official [Github repository](https://github.com/EdgeNet-project/edgenet).

# Prerequisites
To effectively understand how to use EdgeNet in your Kubernetes cluster, it is required to have a basic understanding of Kubernetes. For more information on how Kubernetes operates, you can refer to the [Kubernetes documentation](https://kubernetes.io/docs/).

# Installation
To deploy EdgeNet to your Kubernetes cluster, we recommend following the [advanced installation tutorial](/docs/tutorials/deploy_edgenet_to_kube.md). To use the command line tool `fedmanctl` we recommend following the [fedmanctl installation tutorial](/docs/tutorials/fedmanctl_installation.md).

If you want to remove EdgeNet from your cluster, refer to the [removing EdgeNet from a Kubernetes cluster tutorial](/docs/tutorials/remove_edgenet_from_kube.md).

# Tutorials
We have provided some of the tutorials below for using different functionalities:

- [Registering a Tenant](/docs/tutorials/tenant_registration.md)
- [Creating a Node-level Slice for Your Tenant](/docs/tutorials/slice_creation.md)
- [Subnamespace Creation for Your Tenant](/docs/tutorials/subnamespace_creation.md)
- [Configure User Permissions](/docs/tutorials/user_permissions.md)
- [Role Request](/docs/tutorials/role_request.md)
- [Cluster Role Request](/docs/tutorials/cluster_role_request.md)

We have also added tutorials for `fedmanctl` command line tool. They can be accessed here:
- [Federating Workload Clusters with fedmanctl](/docs/tutorials/federating_worker_clusters_fedmanctl.md)

Old EdgeNet tutorials can be accessed under the `/doc/tutorials/old` folder. See the [old tutorial's Readme](/docs/tutorials/old/README.md).

# Components
We have devised 4 major categories of features that EdgeNet offers and grouped the corresponding CRDs. However, this classification does not fully differentiate CRDs. For instance, the Selective Deployment CRD takes the role in both federation and location-based node selection categories. 

Here are the categories and their associated CRDs:

- [Multitenancy](/docs/custom_resources.md#multitenancy):
    - [Tenant](/docs/custom_resources.md#tenant)
    - [Tenant Request](custom_resources.md#tenant-request)
    - [Tenant Resource Quota](custom_resources.md#tenant-resource-quota)
    - [Subnamespace](custom_resources.md#subnamespace)
    - [Slice](custom_resources.md#slice)
    - [Slice Claim](custom_resources.md#slice-claim)
    - [Role Request](custom_resources.md#role-request)
    - [Cluster Role Request](custom_resources.md#cluster-role-request)


- [Multiprovider](/docs/custom_resources.md#multiprovider):
    - [Node Contribution](custom_resources.md#node-contribution)
    - [VPN Peer](custom_resources.md#vpn-peer)
  

- [Location-Based Node Selection](/docs/custom_resources.md#location-based-node-selection):
    - [Selective Deployment](custom_resources.md#selective-deployment)


- [Federation](/docs/custom_resources.md#cluster-federation):
    - [Selective Deployment Anchor](custom_resources.md#selective-deployment-anchor)
    - [Manager Cache](custom_resources.md#manager-cache)
    - [Cluster](custom_resources.md#cluster)

### Fedmanctl
To facilitate federation capabilities, our EdgeNet project incorporates a command line utility called `fedmanctl`. This utility serves the purpose of enabling the federation of workload and manager Kubernetes clusters. 

Currently, `fedmanctl` is undergoing active development, focusing primarily on implementing the core components related to federation features. While it is a work in progress, essential functionalities have already been incorporated. For the basic use case of federating a workload Kubernetes cluster, please consult the [fedmanctl federation tutorial](/docs/tutorials/federating_worker_clusters_fedmanctl.md).

`fedmanctl` comprises two modules known as workload and manager. The workload module encompasses subcommands that initialize the federation capabilities of EdgeNet and generate a token for the manager cluster. This token contains sensitive information that grants external access to the workload cluster's API server. On the other hand, the manager subcommands are responsible for establishing a link between the workload cluster and the manager cluster using the token.

In the future, additional subcommands may be developed to establish connections between manager Kubernetes clusters.

# Concepts
## Extending Kubernetes
To extend Kubernetes API, EdgeNet makes use of Kubernetes [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (custom resource definitions). CRDs allow us to define custom API objects to be manipulated by the Kubernetes API. These components and their properties are explained in [components seciton](#components).

A [controller](https://kubernetes.io/docs/concepts/architecture/controller/) in Kubernetes listens for changes in an object. Then it tries to converge the desired state of the object with the current state. It is also possible to implement custom controllers that may contain business logic for custom objects. EdgeNet employs its custom controllers can be found under the `pkg/controllers` folder and uses the design of Kubernetes' [sample-controller](https://github.com/kubernetes/sample-controller) example. Additionally, custom controllers usually run inside the cluster in a production environment. The images of these custom controllers can be found on [EdgeNet's DockerHub](https://hub.docker.com/u/edgenetio) page.

## Multitenancy
The architecture of the EdgeNet custom resources and controllers runs in the control plane along with other Kubernetes components. To ensure better isolation of workloads, we advise using [Kata Containers](https://katacontainers.io/) as the container runtime, but this is optional and any container runtime can be configured when creating the Kubernetes cluster. You can refer to official Kubernetes documentation on [how to create Kubernetes clusters](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/) for more information.

EdgeNet uses the [tenant](/docs/custom_resources.md#tenant) custom resource to represent a tenant where each tenant is a customer that contracts for services on behalf of one or more users. EdgeNet provides multitenancy support to Kubernetes clusters it has installed. This means a tenant and its workloads cannot access resources, or objects reserved for other tenants or their workloads. 

In some cases, we want to ensure the resources are reserved for a specific tenant. This requirement is satisfied by a mechanism called [slicing](/docs/custom_resources.md#slice), which is assigned to tenants by creating [slice claims](/docs/custom_resources.md#slice-claim). There are two types to create two types of slices; Node-level slices allow the reservation of whole nodes just for a single tenant. Sub-node-level slices, on the other hand, allow granular resources on a selected node to be reserved.

![Slicing](/docs/architecture/slicing.png)

There is also the [subnamespace](/docs/custom_resources.md#subnamespace) mechanism implemented in EdgeNet to ensure tenants create non-flat namespaces with specific resource quotas. The resource limitations are also propagated when a new subnamespace is added. This can be seen in the figure below. `r` represents the root namespace which has a specified quota of 100 units. Note that, `r` doesn't have a quota directly since it is an abstraction. However, each other namespace exists in the flat namespaces of Kubernetes thus, they also have a quota assigned to them. For example, when the two subnamespaces `aa` and `ab` are added to the subnamespace `a`, the 60-unit resource is divided by 3 to 20, 25, and 15 units. 

![D](/docs/architecture/subnamespaces.png)

Lastly, the tenants can have [tenant resource quotas](/docs/custom_resources.md#tenant-resource-quota). Which puts a limit to a tenant's useable resource quota.

<!-- ## Multiprovider -->

## Location-Based Node Selection
EdgeNet utilizes a custom resource known as [selective deployment](/docs/custom_resources.md#selective-deployment) to facilitate the specification of a deployment's geographic area. This feature enables users to precisely define the geographical boundaries within which their deployment operates. To support this mechanism, it is essential to determine the geographic locations of the nodes involved.

To achieve this, EdgeNet employs a node labeler, which is responsible for assigning geographical labels to each node. The node labeler plays a crucial role in accurately identifying and categorizing the nodes based on their physical locations. By labeling the nodes accordingly, EdgeNet can effectively manage the selective deployment of resources and ensure that workloads are distributed within the specified geographic area.

The combination of the selective deployment custom resource and the node labeler enhances EdgeNet's capabilities in achieving targeted and geographically constrained deployments. This enables users to have greater control over the geographical distribution of their resources and optimize their system's performance based on specific requirements or constraints.

## Federation of Multiple EdgeNet Clusters
The incorporation of federation support within EdgeNet empowers multiple clusters to collaborate and distribute workloads efficiently. This feature aims to facilitate the creation of a flexible system by enabling clusters from different providers. By leveraging the federation capabilities, EdgeNet users can harness the advantages of geographically distributed resources and optimize their workload management.

However, it is important to note that the federation features are currently in the development phase and have not yet reached their full potential in the release-1.0 version of EdgeNet. The development team is actively working to enhance and refine these features to ensure their reliability and effectiveness.

# Development and Contributing
If you are familiar with the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) used by other Kubernetes-related projects, you will easily be able to navigate the EdgeNet repository.

To get a sense of where we are heading, please see our [planned features board](https://github.com/orgs/EdgeNet-project/projects/1). We follow an agile development approach, with two-week sprints, each one leading to a new psroduction version of the code. Our current sprint is one of the milestones, and you can see more near-term issues in our [project backlog](https://github.com/orgs/EdgeNet-project/projects/2).

To start work, clone the latest release branch. If you add a new piece of code, please make sure you have prefaced it with the standard copyright notice and license information found in other places in the code. If you have an idea or an implementation you would like us to look at, please create a pull request for [@bsenel](https://github.com/bsenel) to review.

### Contributing Guides
Please refer to the [contributing guides](/docs/guides/contribution_guides.md) before creating a pull request.

### Architectural Design Proposals (ADRs)
For [architectural design proposal](/docs/adrs/README.md) please create a ADR located under `/doc/adrs/`.

### Unit Tests Guides
We make sure the code works correctly by having high-quality unit tests. You can find the [unit test guides](/docs/guides/unit_test_guides.md) for creating unit tests.


### Branches
* The `master` branch reflects the currently-deployed version of EdgeNet.
* The latest `release` branch is where we prepare the next EdgeNet release. Please use this branch for all of the pull requests.