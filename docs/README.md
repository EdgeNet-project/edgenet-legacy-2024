# EdgeNet Documentation
## What is EdgeNet?
EdgeNet is an open-source edge cloud orchestration software extension that is built on top of industry-standard cloud software [Kubernetes](https://kubernetes.io/) and utilizes different container runtimes such as [Containerd](https://containerd.io/), or [KataContainers](https://katacontainers.io/). The EdgeNet's source code is hosted in the official [Github repository](https://github.com/EdgeNet-project/edgenet).

The EdgeNet software can be installed on any Kubernetes cluster. The documentation of EdgeNet software is distinct from the [EdgeNet Testbed](https://edge-net.org) which is a globally distributed edge cloud for Internet researchers. We encourage everybody to contribute a node to the testbed. To participate in the testbed and learn more please visit the [edgenet website](https://edge-net.org).

## Getting Started
EdgeNet is an open-source extension for Kubernetes clusters. You can install EdgeNet to your Kubernetes cluster and start using the functionalities. To deploy EdgeNet to your cluster, we reccomend following the [Getting Started](/docs/installation/README.md).

## Tutorials
EdgeNet is a complex system with many different capabilities. We highly advise to see the [Tutorials Section](/docs/tutorials/README.md).

## EdgeNet's Components
To flexibly add functionalities to the Kubernetes API server without the burden of updating the codebase, EdgeNet introduces it's own [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/). We have devised 4 major categories for EdgeNet and assigned the CRDs to their places. However, this classification does not fully differentiate CRDs. For instance, the Selective Deployment CRD takes role in both federation and location-based node selection categories. 

Here are the categories and their associated CRDs:

* Multitenancy:
    * [Tenant](/docs/custom_resources.md#tenant)
    * [Tenant Resource Quota](custom_resources.md#tenant-resource-quota)
    * [Subnamespace](custom_resources.md#subnamespace)
    * [Slice](custom_resources.md#slice)
    * [Slice Claim](custom_resources.md#slice-claim)
    * [Tenant Request](custom_resources.md#tenant-request)
    * [Role Request](custom_resources.md#role-request)
    * [Cluster Role Request](custom_resources.md#cluster-role-request)


* Multiprovider:
    * [Node Contribution](custom_resources.md#node-contribution)
    * [VPN Peer](custom_resources.md#vpn-peer)
  

* Locations-Based Node Selection:
    * [Selective Deployment](custom_resources.md#selective-deployment)

* Cluster Federation:
    * [Selective Deployment Anchor](custom_resources.md#selective-deployment-anchor)

EdgeNet also provides custom [controllers](https://kubernetes.io/docs/concepts/architecture/controller/) for these resources. These controllers check the states of the CRDs in a loop and try to make them closer to their specs. In the Kubernetes world, status represents the current state of the objects and specs represent the desired states.

These controllers usually run inside the cluster and communicate with the kube-api server to fulfill certain functionalities. To see how the system is designed see the [architecture document](/docs/architecture/README.md).

<!-- FOR THE DOCUMENTORS! We can add more specific documentation such as the ones below as time progresses. -->
<!-- ## Scheduling and Selective Deployment -->
<!-- ## Federating Clusters -->

## Development and Contributing

To get a sense of where we are heading, please see our [planned features board](https://github.com/orgs/EdgeNet-project/projects/1). We follow an agile development approach, with two-week sprints, each one leading to a new production version of the code. Our current sprint is one of the milestones, and you can see more near-term issues in our [project backlog](https://github.com/orgs/EdgeNet-project/projects/2). You can pick one of these to work on or suggest your own.

To start work, clone the latest release branch. If you add a new code, please be sure to preface it with the standard copyright notice and license information found elsewhere in the code. When you have something you would like us to look at, please create a pull request for [@bsenel](https://github.com/bsenel) to review.

Please refer to the [contributing guides](/docs/guides/contribution_guides.md) before creating a pull request.

### Unit Tests

To make sure the code works correctly it is important to have high-quality unit tests. You can find the [unit test guides](/docs/guides/unit_test_guides.md) for creating unit tests.

### Branches
* The `master` branch reflects the currently-deployed version of EdgeNet.
* The latest `release` branch is where we prepare the next EdgeNet release. Please use this branch for all of the pull requests.