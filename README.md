# sundew-one

[![Build Status](https://travis-ci.org/aaaaalbert/sundew-one.svg?branch=master)](https://travis-ci.org/aaaaalbert/sundew-one)
[![Coverage Status](https://coveralls.io/repos/aaaaalbert/sundew-one/badge.png)](https://coveralls.io/r/aaaaalbert/sundew-one)
[![Python 3](https://pyup.io/repos/github/aaaaalbert/sundew-one/python-3-shield.svg)](https://pyup.io/repos/github/aaaaalbert/sundew-one/)
[![pyup](https://pyup.io/repos/github/aaaaalbert/sundew-one/shield.svg)](https://pyup.io/repos/github/aaaaalbert/sundew-one/)
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2Faaaaalbert%2Fsundew-one.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2Faaaaalbert%2Fsundew-one?ref=badge_shield)
[![CII](https://bestpractices.coreinfrastructure.org/projects/----TODO----/badge)](https://bestpractices.coreinfrastructure.org/projects/----TODO----)


This repository hosts the code that implements
[Sundew Project's](https://sundew-project.github.io/) prototype portal
webpage. The portal is where you register for using Sundew, and receive
the required configuration files for later accessing your personalized
Kubernetes (or "K8s") dashboard to administrate your Sundew resources.



## Components

The portal code is split into two parts: A user-visible Web frontend, and
a backend that interfaces with Kubernetes. The frontend is a
[Google App Engine web app](https://github.com/aaaaalbert/sundew-one/blob/master/test-portal/test1.py)
written in Python / [`webapp2`](https://pypi.org/project/webapp2/),
with [Jinja](http://jinja.pocoo.org/) templates. 
It lets the user register an account, agree to the Acceptable Usage Policy (AUP),
and then hands out credentials in the form of a Kubernetes configuration.
With this config, the user can access their personalized Kubernetes
dashboard for further interactions with their resources on the Sundew cluster.

The [backend](https://github.com/aaaaalbert/sundew-one/blob/master/daemons/user-daemon.py)
is a [Flask](http://flask.pocoo.org/) app that is only accessible by the
frontend. For various routes in its app, it calls down into shell scripts
which interface with `kubectl`, the Kubernetes control tool, to create
[users](https://github.com/aaaaalbert/sundew-one/blob/master/user_files/scripts/make-user.sh),
[list the available nodes and namespaces](https://github.com/aaaaalbert/sundew-one/tree/master/daemons),
and [generates the actual per-user config file](https://github.com/aaaaalbert/sundew-one/blob/master/user_files/scripts/make-config.sh)
(including certificates) that the frontend serves.



## Design Options Deferred For The Prototype

As of the May 2018 prototype,

* There is only a **single Kubernetes head node** for all users in the cluster.
  We will later on think about scaling up, distributing and federating head nodes, etc.
* New **user registrations are handled manually** on a per-case basis, and must go
  through some Google App Engine-specific processing. Later, other identity
  providers and user verification methods should be supported.
