# Cluster creation from EdgeNet source code

This documentation describes how to create a working EdgeNet cluster in your own environment. You can refer to this tutorial for creating a sandbox, local or production cluster for your own desires.

## Technologies you will use
You will need a kubernetes cluster to extend it to support EdgeNet. There are many ways to setup a kubernetes cluster all of which requires different knowledge. You can refer [here](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/) for how to setup a kubernetes cluster with kubeadm. Also, to setup a local cluster you can use [minikube](https://minikube.sigs.k8s.io/docs/) or [k3s](https://docs.k3s.io/installation) (be avare EdgeNet is not yet tested with k3s).

If you want to compile EdgeNet from source you need to install [*kompose*](https://kompose.io/#:~:text=Kompose%20is%20a%20conversion%20tool,as%20Kubernetes%20(or%20OpenShift).) command-line tool.

## What will you do

Before the tutorial it is assumed you have an access to a working kubernetes cluster with a valid kubeconfig and kubectl command-line tool installed.

There are two stages for installing EdgeNet to a kubernetes cluster. First, it is necessary to install the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) to the cluster. This operation will create the objects which are required for EdgeNet to function. Then, the custom resources will be deployed. This allows CRDs to be created deleted and altered, and thus EdgeNet to work.

## Steps

### Install the required CRDs and other objects
There are a handful of custom resources required for EdgeNet to function. Additionally, there are others used for configuration. The basic objects that is required for all of the EdgeNet clusters are organized in *all-in-one.yaml*. 

Before applying this yaml it is required to edit the file for adding the secret information. Note that [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) in kubernetes uses base64 encoding.

<!-- *TODO: all-in-one.yaml analysis* -->

You can use the following command to create these auxiliary objects and the CRDs.

```sh
    kubectl -f apply ./build/yamls/kubernetes/all-in-one.yaml
```

<!-- Now we have created the objects necessary for EdgeNet to work. As discussed kubernetes uses controllers to control the state of the objects and thus clusters. Custom controllers allows EdgeNet to implement it's logic for custom objects. For example, a basic mechanism in EdgeNet is about creating Tenants. To create a tenant user should create a Tenant Request which when created signals the Tenant Request Controller and then processed. If the request is valid, then a mail is sent to the cluster administrators.

To enable this functionality there are two options:

* The pre-compiled binaries can be deployed to the cluster. 
* The source can be compiled from source (still testing).

### Option 1: Download EdgeNet contoller images from Docker Hub
[Docker Hub](https://www.docker.com/products/docker-hub/#:~:text=Docker%20Hub%20is%20a%20hosted,push%20them%20to%20Docker%20Hub) is a marketplace for container images. It allows to create and share custom container images. The official Docker Hub name of EdgeNet is [edgenetio](https://hub.docker.com/u/edgenetio).  -->

<!-- ### Option 2: Compile EdgeNet contollers from source
EdgeNet is an open-source which means it can be compiled and deployed. With downloading the soruce code from the [official github repository](https://github.com/EdgeNet-Project/edgenet/). Using the docker-compose file and kompose command-line tool, the controllers can be created.

First, create the yaml files with kompose by the following command:

```sh
    
```

Note that to use a stable versions of EgeNet please build from the *release-X.X* branches such as *release-1.0*.

```sh
    docker-compose up -d
``` -->