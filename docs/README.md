# Table of Contents
- [Table of Contents](#table-of-contents)
- [Prerequisites](#prerequisites)
- [What is EdgeNet?](#what-is-edgenet)
- [Tutorials](#tutorials)
- [Design](#design)
  - [Extending Kubernetes](#extending-kubernetes)
  - [Multitenant Design](#multitenant-design)
  - [Selective Deployments](#selective-deployments)
  - [Federation of Multiple EdgeNet Clusters](#federation-of-multiple-edgenet-clusters)
- [Components](#components)
- [Development and Contributing](#development-and-contributing)
    - [Unit Tests](#unit-tests)
    - [Branches](#branches)

# Prerequisites
For understanding the design and how to use EdgeNet in your Kubernetes cluster, it is required to have a basic understanding of Kubernetes. For more information on how Kubernetes operates you refer to the [Kuebrnetes documentation](https://kubernetes.io/docs/).

# What is EdgeNet?
It is important to distinguish EdgeNet software and EdgeNet Testbed. If you are a user of EdgeNet Testbed please refer to the [Testbed's website](https://edge-net.org). This documentation is intended for users and developers of EdgeNet software (will be referred to as simply EdgeNet).

EdgeNet is an open-source edge cloud orchestration software extension that is built on top of industry-standard cloud software [Kubernetes](https://kubernetes.io/). The EdgeNet's source code is hosted in the official [Github repository](https://github.com/EdgeNet-project/edgenet).

# Tutorials
You can install EdgeNet to your Kubernetes cluster and start using the features. To deploy EdgeNet to your cluster, we recommend following the [Deploying EdgeNet to Kubernetes Tutorial](/docs/tutorials/deploy_edgenet_to_kube.md).

We have also provided some of the tutorials below for using different functionalities offered by EdgeNet.

- [Deploy EdgeNet on a Kubernetes Cluster](/docs/tutorials/deploy_edgenet_to_kube.md)
- [Registering a Tenant](/docs/tutorials/tenant_registration.md)
- [Creating a Node-level Slice for Your Tenant](/docs/tutorials/slice_creation.md)
- [Subnamespace Creation for Your Tenant](/docs/tutorials/subnamespace_creation.md)
- [Configure User Permissions](/docs/tutorials/user_permissions.md)
- [Role Request](/docs/tutorials/role_request.md)
- [Cluster Role Request](/docs/tutorials/cluster_role_request.md)

Old EdgeNet tutorials can be accessed under the `/doc/tutorials/old` folder. See the [old tutorial's Readme](/docs/tutorials/old/README.md).

# Design
## Extending Kubernetes
To extend Kubernetes API, EdgeNet makes use of Kubernetes [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) (custom resource definitions). CRDs allow us to define custom API objects to be manipulated by the Kubernetes API. These components and their properties are explained in [components seciton](#components).

A [controller](https://kubernetes.io/docs/concepts/architecture/controller/) in Kubernetes listens for changes in an object. Then it tries to converge the desired state of the object with the current state. It is also possible to implement custom controllers that may contain business logic for custom objects. EdgeNet employs its custom controllers can be found under the `pkg/controllers` folder and uses the design of Kubernetes' [sample-controller](https://github.com/kubernetes/sample-controller) example. Additionally, custom controllers usually run inside the cluster in a production environment. The images of these custom controllers can be found on [EdgeNet's DockerHub](https://hub.docker.com/u/edgenetio) page.

## Multitenant Design
The architecture of the EdgeNet custom resources and controllers runs in the control plane along with other Kubernetes components. To ensure better isolation of workloads, we advise using [Kata Containers](https://katacontainers.io/) as the container runtime, but this is optional and any container runtime can be configured when creating the Kubernetes cluster. You can refer to official Kubernetes documentation on [how to create Kubernetes clusters](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/) for more information.

EdgeNet uses the [tenant](/docs/custom_resources.md#tenant) custom resource to represent a tenant where each tenant is a customer that contracts for services on behalf of one or more users. EdgeNet provides multitenancy support to Kubernetes clusters it has installed. This means a tenant and its workloads cannot access resources, or objects reserved for other tenants or their workloads. 

In some cases, we want to ensure the resources are reserved for a specific tenant. This requirement is satisfied by a mechanism called [slicing](/docs/custom_resources.md#slice), which is assigned to tenants by creating [slice claims](/docs/custom_resources.md#slice-claim). There are two types to create two types of slices; Node-level slices allow the reservation of whole nodes just for a single tenant. Sub-node-level slices, on the other hand, allow granular resources on a selected node to be reserved.

![Slicing](/docs/architecture/slicing.png)

There is also the [subnamespace](/docs/custom_resources.md#subnamespace) mechanism implemented in EdgeNet to ensure tenants create non-flat namespaces with specific resource quotas. The resource limitations are also propagated when a new subnamespace is added. This can be seen in the figure below. `r` represents the root namespace which has a specified quota of 100 units. Note that, `r` doesn't have a quota directly since it is an abstraction. However, each other namespace exists in the flat namespaces of Kubernetes thus, they also have a quota assigned to them. For example, when the two subnamespaces `aa` and `ab` are added to the subnamespace `a`, the 60-unit resource is divided by 3 to 20, 25, and 15 units. 

![D](/docs/architecture/subnamespaces.png)

Lastly, the tenants can have [tenant resource quotas](/docs/custom_resources.md#tenant-resource-quota). Which puts a limit to a tenant's useable resource quota.

## Selective Deployments
EdgeNet utilizes a custom resource known as [selective deployment](/docs/custom_resources.md#selective-deployment) to facilitate the specification of a deployment's geographic area. This feature enables users to precisely define the geographical boundaries within which their deployment operates. To support this mechanism, it is essential to determine the geographic locations of the nodes involved.

To achieve this, EdgeNet employs a node labeler, which is responsible for assigning geographical labels to each node. The node labeler plays a crucial role in accurately identifying and categorizing the nodes based on their physical locations. By labeling the nodes accordingly, EdgeNet can effectively manage the selective deployment of resources and ensure that workloads are distributed within the specified geographic area.

The combination of the selective deployment custom resource and the node labeler enhances EdgeNet's capabilities in achieving targeted and geographically constrained deployments. This enables users to have greater control over the geographical distribution of their resources and optimize their system's performance based on specific requirements or constraints.

## Federation of Multiple EdgeNet Clusters
The incorporation of federation support within EdgeNet empowers multiple clusters to collaborate and distribute workloads efficiently. This feature aims to facilitate the creation of a flexible system by enabling clusters from different providers. By leveraging the federation capabilities, EdgeNet users can harness the advantages of geographically distributed resources and optimize their workload management.

However, it is important to note that the federation features are currently in the development phase and have not yet reached their full potential in the release-1.0 version of EdgeNet. The development team is actively working to enhance and refine these features to ensure their reliability and effectiveness. 


# Components
We have devised 4 major categories for EdgeNet and assigned the CRDs to their places. However, this classification does not fully differentiate CRDs. For instance, the Selective Deployment CRD takes the role in both federation and location-based node selection categories. 

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
  

- [Locations-Based Node Selection](/docs/custom_resources.md#selective-deployment):
    - [Selective Deployment](custom_resources.md#selective-deployment)


- [Cluster Federation](/docs/custom_resources.md#cluster-federation):
    - [Selective Deployment Anchor](custom_resources.md#selective-deployment-anchor)
    - [Manager Cache](custom_resources.md#manager-cache)
    - [Cluster](custom_resources.md#cluster)

# Development and Contributing

To get a sense of where we are heading, please see our [planned features board](https://github.com/orgs/EdgeNet-project/projects/1). We follow an agile development approach, with two-week sprints, each one leading to a new production version of the code. Our current sprint is one of the milestones, and you can see more near-term issues in our [project backlog](https://github.com/orgs/EdgeNet-project/projects/2). You can pick one of these to work on or suggest your own.

To start work, clone the latest release branch. If you add a new code, please be sure to preface it with the standard copyright notice and license information found elsewhere in the code. When you have something you would like us to look at, please create a pull request for [@bsenel](https://github.com/bsenel) to review.

Please refer to the [contributing guides](/docs/guides/contribution_guides.md) before creating a pull request.

For [architectural design proposal](/docs/adrs/README.md) please create a ADR located under `/doc/adrs/`.

### Unit Tests

To make sure the code works correctly it is important to have high-quality unit tests. You can find the [unit test guides](/docs/guides/unit_test_guides.md) for creating unit tests.

### Branches
* The `master` branch reflects the currently-deployed version of EdgeNet.
* The latest `release` branch is where we prepare the next EdgeNet release. Please use this branch for all of the pull requests.

<!-- ## About the Source Code
If you are familiar with the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) used by other Kubernetes-related projects, you will easily be able to navigate this repository.

EdgeNet extends Kubernetes via [custom controllers](https://kubernetes.io/docs/concepts/architecture/controller/). They check for state changes of custom EdgeNet resources and try to converge the current state with the desired state. We EdgeNet source code contains these controllers' source code. You can find the docker controller images in [EdgeNet's DockerHub](https://hub.docker.com/u/edgenetio).


The architecture of EdgeNet is described in the [architecture document](/docs/architecture/README.md). -->