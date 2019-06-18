# EdgeNet Head Node

This repository hosts the code that implements
[EdgeNet Project's](https://github.com/EdgeNet-project) early version of the head node. The head node is where the portal interacts with Kubernetes through and tells to add a new worker to the DNS.



## Components

The EdgeNet project code is split into two parts: A user-visible Web frontend, and
a backend that interfaces with Kubernetes. The frontend is a
[Google App Engine web app](https://github.com/EdgeNet-project/portal/tree/master)
written in Python / [`webapp2`](https://pypi.org/project/webapp2/),
with [Jinja](http://jinja.pocoo.org/) templates. 
It lets the user register an account, agree to the Acceptable Usage Policy (AUP),
and then hands out credentials in the form of a Kubernetes configuration.
With this config, the user can access their personalized Kubernetes
dashboard for further interactions with their resources on the Sundew cluster.

The [backend](https://github.com/EdgeNet-project/headnode)
is a [Golang](https://golang.org/) app that is only accessible by the
frontend. For various routes in its app, it gets the benefit of [client-go](https://github.com/kubernetes/client-go) and [Go packages](https://godoc.org/k8s.io/kubernetes) for talking to the Kubernetes cluster of EdgeNet and makes use of Namecheap API. The [API documentation](https://documenter.getpostman.com/view/7656709/S1ZxapRG?version=latest) which describes main functionallities of the head node is created by [Postman](https://www.getpostman.com/).



## Design Options Deferred For The Prototype

As of the May 2018 prototype,

* There is only a **single Kubernetes head node** for all users in the cluster.
  We will later on think about scaling up, distributing and federating head nodes, etc.
* New **user registrations are handled manually** on a per-case basis, and must go
  through some Google App Engine-specific processing. Later, other identity
  providers and user verification methods should be supported.

## Implementation of The New Architecture

As of May 2019,

* Firstly, we will the port current head node code to Go, to getting the benefit of **client-go** library.
* Then, **CRDs** will substitute the current datastore and cronjobs by taking advantage of **custom controllers**.
