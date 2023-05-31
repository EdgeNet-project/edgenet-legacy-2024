EdgeNet Architectural Decision Record ```ADR-002``` <!-- added by Timur; not part of the MADR template -->

# EdgeNet's web presence consists of six elements

* Status: pending approval (revised from prior, approved, version) <!-- optional -->
* Deciders: Berat Senel, Ciro Scognamiglio, Timur Friedman, Rick McGeer <!-- optional -->
* Date: 2020-05-06<!-- optional (last update) -->

Technical Story: [Issue #75](https://github.com/EdgeNet-project/edgenet/issues/75) <!-- optional -->

## Context and Problem Statement

EdgeNet is visible in a number of ways via the web. We aim to inventory these and agree on how to move forward.

## Decision Outcome

Our web presence should consist of:
1. a public website for users of the testbed
   * at https://www.edge-net.org/
   * source at [edgenet/website](../../website) in the project's GitHub repository (see below)
1. the Kubernetes console for users of the testbed
   * at https://dashboard.edge-net.org/
   * the API endpoint for ``kubectl`` commands is the same server, on another port number: https://51.75.127.152:6443
1. a web console for non-Kubernetes users of the testbed
   * (TO BE PUT IN PLACE)
1. the [EdgeNet project](https://github.com/EdgeNet-project/) on GitHub (https://github.com/EdgeNet-project/) containing:
   * resources for users
      * tutorials at [edgenet/docs/tutorials](../tutorials)
   * resources for developers
      * the EdgeNet code, in the [edgenet](../..) repository
      * architectural decision records (ADRs) at [edgenet/docs/adrs](../adrs)
      * the [project backlog](https://github.com/orgs/EdgeNet-project/projects)
      * the [list of open issues](https://github.com/EdgeNet-project/edgenet/issues)
      * the [project milestones](https://github.com/EdgeNet-project/edgenet/milestones)
1. API documentation for developers [on Postman](https://documenter.getpostman.com/view/7656709/SzYT4gRL?version=latest)
1. code documentation for developers
   * (TO BE PUT IN PLACE)
1. a live chat page for both users and developers at [https://tawk.to/edgenet](https://tawk.to/edgenet)
