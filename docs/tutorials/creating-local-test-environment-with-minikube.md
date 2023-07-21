# Creating a Local Testing Environment for Federation with Minikube

This tutorial describes how to setup and install EdgeNet's multi-tenancy and federation features into a minikube cluster and demonstrates how to 

## Technologies you will use

We reccomend using the latest version of [``minikube``](https://minikube.sigs.k8s.io/docs/) with [``docker``](https://www.docker.com/) as it's driver.

To test the federation framework we will use the ``fedmanctl`` CLI tool. You can follow the [fedmanctl's installation tutorial]() to install in your local machine.

## What will you do

In this tutorial we will create 3 minikube clusters named `manager`, `workload-paris`, and `workload-newyork`. These clusters will be in the default `bridge` network for communication.

Then, we will install EdgeNet's `multi-tenancy` and `federation` frameworks.

Then we will create a `tenantrequest` to create a tenant along with a subnamespace.

At last we will use fedmanctl 