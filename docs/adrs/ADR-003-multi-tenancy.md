EdgeNet Architectural Decision Record ```ADR-003``` <!-- added by Timur; not part of the MADR template -->

# EdgeNet's multi-tenant nomenclature

* Status: pending approval (revised from prior, approved, version) <!-- optional -->
* Deciders: Berat Senel, Ciro Scognamiglio, Maxime Mouchet, Olivier Fourmaux, Timur Friedman, Rick McGeer <!-- optional -->
* Date: 2021-04-13<!-- optional (last update) -->

Technical Story: [Brief Spec Doc](https://docs.google.com/document/d/1lAF6PS6BnV541dyGBlzbTRjkAtHY4xFoPksFOVgnY_s/edit?usp=sharing) <!-- optional -->

## Context and Problem Statement

EdgeNet has been positioning itself to replace PlanetLab Europe (PLE)'s Linux Containers (LXC) based design with a contemporary Docker and Kubernetes based design. In keeping with this approach, the concepts and the design of EdgeNet multitenancy were inherited from PlanetLab and from the Horizon 2020 Fed4FIRE+ European testbed federation project in which PLE participates. However, the Kubernetes community is not familiar with the terms from the testbed community, such as Authority and Slice, or PI (for principal investigator), and continuing with such nomenclature is expected to hinder the ability of EdgeNet to contribute back to Kubernetes. We therefore seek to bring EdgeNet's mutitenancy nomenclature into line with current practice in the Kubernetes community. We would also like to take advantage of the nomenclature update to adjust certain aspects of EdgeNet's current multitenancy design that require steps that ought not to be strictly necessary (creation of a Slice) before a service can be deployed. Finally, this is an opportunity for us to produce cleaner code.

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
