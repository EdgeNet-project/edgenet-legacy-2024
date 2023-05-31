EdgeNet Architectural Decision Record ```ADR-001```

# Adopt the Standard Go Project Layout

* Status: accepted <!-- optional -->
* Deciders: Berat Senel, Ciro Scognamiglio, Timur Friedman, Rick McGeer <!-- optional -->
* Date: 2020-04-30 <!-- optional -->

## Context and Problem Statement

How should we structure the EdgeNet project on GitHub? We want the project to be legible to ourselves and the outside world.

## Considered Options

* several repositories, structured around our sense of what is important
* one repository, following the [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
* a hybrid, with one repository and links visible at the top level to key subdirectories

## Decision Outcome

Chosen option: "one repository", because this comes out best (see below).

### Positive Consequences <!-- optional -->

* Legibility in the Kubernetes community, for whom this is the standard project layout

### Negative Consequences <!-- optional -->

* A bit harder for us to read, at least initially

## Links <!-- optional -->

* Standard Go Project Layout: https://github.com/golang-standards/project-layout
