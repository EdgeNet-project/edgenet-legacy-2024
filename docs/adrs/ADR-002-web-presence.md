EdgeNet Architectural Decision Record ```ADR-002``` <!-- added by Timur; not part of the MADR template -->

# EdgeNet's web presence consists of six elements

* Status: accepted <!-- optional -->
* Deciders: Berat Senel, Ciro Scognamiglio, Timur Friedman, Rick McGeer <!-- optional -->
* Date: 2020-04-30 <!-- optional (last update) -->

Technical Story: [Issue #75](https://github.com/EdgeNet-project/edgenet/issues/75) <!-- optional -->

## Context and Problem Statement

EdgeNet is visible in a number of ways via the web. We aim to inventory these and agree on how to move forward.

## Decision Outcome

Our web presence should consist of:
1. public website for users
   * at http://www.edge-net.org/
   * source at [edgenet/website](../../website)
1. support section for users on the GitHub
   * to be put in place
1. web console for non-Kubernetes users
   * to be put in place
1. tutorial section on the GitHub, containing specific tutorials for different events
   * at [edgenet/docs/tutorials](../tutorials)
1. development documents on GitHub, outlining the architecture
   * in this directory and other [edgenet/docs](..) subdirectories
1. source code on the GitHub
   * in various files under the [edgenet](https://github.com/EdgeNet-project/edgenet/tree/master) repository
