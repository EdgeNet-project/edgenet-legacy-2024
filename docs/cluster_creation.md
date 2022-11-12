# Cluster creation from EdgeNet source code

This documentation describes how to create a working EdgeNet cluster in your own environment. You can refer to this tutorial for creating a sandbox, local or production cluster for your own desires.

## Technologies you will use
You will need a kubernetes cluster to extend it to support EdgeNet. There are many ways to setup a kubernetes cluster all of which requires different knowledge. You can refer [here](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/) for how to setup a kubernetes cluster with kubeadm. Also, to setup a local cluster you can use [minikube](https://minikube.sigs.k8s.io/docs/) or [k3s](https://docs.k3s.io/installation) (be avare EdgeNet is not yet tested with k3s).

## What will you do

Before the tutorial it is assumed you have an access to a working kubernetes cluster with a valid kubeconfig and kubectl command-line tool installed.

There are two stages for installing EdgeNet to a kubernetes cluster. First, it is necessary to install the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to the cluster. This operation will create the objects which are required for EdgeNet to function. Then, the custom resources will be deployed. This allows CRDs to be created deleted and altered, and thus EdgeNet to work.

## Steps

### Install the required CRDs and other objects
*TODO: all-in-one.yaml analysis*

### Option 1: Compile EdgeNet contollers from source
EdgeNet is an open-source which means it can be compiled and deployed. In this case this will be the approach.

 *TODO: go version, kubernetes version... etc.*