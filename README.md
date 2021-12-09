## <img src="/assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text_300dpi_10pct.png" alt="Welcome to EdgeNet" width="400">

[![Go Report Card](https://goreportcard.com/badge/github.com/EdgeNet-project/edgenet)](https://goreportcard.com/report/github.com/EdgeNet-project/edgenet)
[![Build Status](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml/badge.svg?branch=main)](https://github.com/EdgeNet-project/edgenet/actions/workflows/test_and_publish.yaml)
[![Coverage Status](https://coveralls.io/repos/github/EdgeNet-project/edgenet/badge.svg?branch=main)](https://coveralls.io/github/EdgeNet-project/edgenet?branch=main)
[![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/EdgeNet-project/edgenet)](https://github.com/EdgeNet-project/edgenet/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/EdgeNet-project/edgenet.svg)](https://pkg.go.dev/github.com/EdgeNet-project/edgenet)

EdgeNet is Kubernetes adapted for the network edge. It allows you to deploy applications to hundreds of nodes
that are scattered across the internet, rather than to just one or a small number of centralized datacenters.

The code that we are developing here is in production powering [the EdgeNet testbed](https://www.edge-net.org/),
on which researchers worldwide conduct experiments in distributed systems and internet measurements.

## Support

To chat with a member of the EdgeNet team live, please [open our tawk.to window](https://tawk.to/edgenet).

## Architecture and code layout

EdgeNet extends Kubernetes in the areas of multitenancy, user management, multiprovider support, and the ability to make selective deployments. This is described in the [architecture document](https://github.com/EdgeNet-project/edgenet/tree/release-1.0-documentation/docs/architecture).

If you are familiar with the [Standard Go Project Layout](https://github.com/golang-standards/project-layout) used
by other Kubernetes-related projects, you will easily be able to navigate this repository.

## Contributing

The EdgeNet software is free and open source, licensed under the [Apache 2.0 license](https://www.apache.org/licenses/LICENSE-2.0); we invite you to contribute.

A good way to start familiarizing yourself with the code is to help us write unit tests. Take a look at the [architecture document](https://github.com/EdgeNet-project/edgenet/tree/release-1.0-documentation/docs/architecture) and follow the links to individual code files, and dig in!

To get a sense of where we are heading, please see our 
[planned features board](https://github.com/orgs/EdgeNet-project/projects/1).
We follow an agile development approach, with two week sprints, each one leading to a new production version of the 
code. Our current sprint is one of the milestones, and you can see more near-term issues in our 
[project backlog](https://github.com/orgs/EdgeNet-project/projects/2).
You can pick one of these to work on, or suggest your own.

To start work, clone the latest release branch.
If you add new code, please be sure to preface it with the standard copyright notice and license information found elsewhere in the code.
When you have something you would like us to look at, please create a pull request for @bsenel to review.
