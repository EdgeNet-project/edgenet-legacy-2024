EdgeNet Architectural Decision Record ```ADR-003``` <!-- added by Timur; not part of the MADR template -->

# EdgeNet's multi-tenant design

* Status: pending approval (revised from prior, approved, version) <!-- optional -->
* Deciders: Berat Senel, Ciro Scognamiglio, Maxime Mouchet, Olivier Fourmaux, Timur Friedman, Rick McGeer <!-- optional -->
* Date: 2021-04-13<!-- optional (last update) -->

Technical Story: [Brief Spec Doc](https://docs.google.com/document/d/1lAF6PS6BnV541dyGBlzbTRjkAtHY4xFoPksFOVgnY_s/edit?usp=sharing) <!-- optional -->

## Context and Problem Statement

EdgeNet was positioning itself to replace PlanetLab Europe (PLE) with using novel technologies. Thus, the concepts and design of multi-tenancy are inherited from PLE and Fed4FIRE+ in order to be integrated into Kubernetes. However, the Kubernetes community is not familiar with the terms, such as Authority and Slice. And this is one of the reasons that make it harder to contribute back to the community. Furthermore, there are drawbacks in the current design, such as the additional steps required to deploy an application. They also prevent us from achieving cleaner code.

## Decision Drivers <!-- optional -->

* Reach out to a broader community with the terms
* Be integrable with Kubernetes Working Group solutions
* Straightforward to deploy an application
* Less and lean code

## Considered Options

1. naming pattern at tenant namespace creation
  * \<tenant name>
  * tenant-\<tenant name>
2. naming pattern at subsidiary namespace creation
  * \<subnamespace name>
  * tenant-\<tenant name>-\<subnamespace name>
  * tenant-\<tenant name>-sub-\<subnamespace name>
  * \<tenant name>-\<subnamespace name>
  * \<tenant name>-sub-\<subnamespace name>
  * tenant-\<tenant name>-\<subnamespace name>-\<subnamespace name>-...
  * tenant-\<tenant name>-sub-\<subnamespace name>-sub-\<subnamespace name>-...
  * \<tenant name>-\<subnamespace name>-\<subnamespace name>-...
  * \<tenant name>-sub-\<subnamespace name>-sub-\<subnamespace name>-...

## Decision Outcome

Our multi-tenancy design should:
1. use Tenant term instead of Authority
  * naming pattern at tenant namespace creation is <tenant name> (Opt 1.1), there is no prefix or suffix included (TBD)
  * tenant user list exists at tenant spec
  * user registration request list exists at tenant status
  * node contribution list exists at tenant status
2. replace Slice and Team with SubNamespace by inheriting the necessary functionalities
  * naming pattern at subsidiary namespace creation is <tenant name>-<subnamespace name> (Opt 2.4), tenant name is included as prefix (TBD)
3. remove User custom resource
4. make User Registration Request, Acceptable Use Policy, Node Contribution, Email Verification cluster-scoped
  * RBAC for authorization
