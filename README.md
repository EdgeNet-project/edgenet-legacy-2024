# EdgeNet Head Node

This repository hosts the code that implements an early version of the
[EdgeNet Project's](https://github.com/EdgeNet-project) head node.
The head node is where the portal interacts with Kubernetes
and makes requests for new workers to be added to the DNS.



## Components

The EdgeNet project code is split into two parts: A user-visible Web front end, and
a back end that interfaces with Kubernetes.

The front end is a
[Google App Engine web app](https://github.com/EdgeNet-project/portal/tree/master)
written in Python / [`webapp2`](https://pypi.org/project/webapp2/),
with [Jinja](http://jinja.pocoo.org/) templates. 
It lets the user register an account and agree to the Acceptable Usage Policy (AUP),
and then hands out credentials in the form of a Kubernetes configuration.
With this config, the user can access their personalized Kubernetes
dashboard for further interactions with their resources on the Sundew cluster.

The [back end](https://github.com/EdgeNet-project/headnode)
is an app written in [Go](https://golang.org/) that is only accessible via the
front end. For various routes in its app, it gets the benefit of [client-go](https://github.com/kubernetes/client-go) and [Go packages](https://godoc.org/k8s.io/kubernetes) for talking to the Kubernetes cluster of EdgeNet and makes use of the Namecheap API. The [API documentation](https://documenter.getpostman.com/view/7656709/S1ZxapRG?version=latest) which describes the main functionallities of the head node is created by [Postman](https://www.getpostman.com/).



## Design Options Deferred For The Prototype

As of the May 2018 prototype,

* There is only a **single Kubernetes head node** for all users in the cluster.
  We will later on think about scaling up, distributing, and federating head nodes, etc.
* New **user registrations are handled manually** on a case-by-case basis, and must go
  through some Google App Engine-specific processing. Later, other identity
  providers and user verification methods should be supported.

## Implementation of The New Architecture

As of May 2019,

* Firstly, we port the existing head node code to Go, so as to benefit from the **client-go** library.
* Then, **CRDs** will be used to substitute for the current datastore and cronjobs, thereby taking advantage of **custom controllers**.
