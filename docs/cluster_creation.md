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
A handful of CRDs, controllers and additional objects  required are for EdgeNet to function. All of these declerations are orgainzed in *all-in-one.yaml* file.

The CRDs are special objects of EdgeNet in edgenet namespace that defines EdgeNet specific objects. The objects are discussed in [here](custom_resources.md).

The controllers are used by kubernetes to controll the state of the objects and thus the state of the cluster. Custom controllers allows EdgeNet to implement it's logic to maintain custom objects. For example, to create a tenant in EdgeNet, user should create a Tenant Request which when created signals the Tenant Request Controller and then processed. If the request is valid, then a mail is sent to the cluster administrators.

Other auxiliary objects in includes [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) that require configuration for elements in EdgeNet to work. Note that before applying this yaml it is required to edit the file for adding the secret information which the it requires a base64 enconding. To convert a token the following code can be used.

```
    echo "<token-or-secret>" | base64
```

You can use the following command to create these auxiliary objects, the CRDs and the deployment of the custom controllers.

```
    kubectl -f apply ./build/yamls/kubernetes/all-in-one.yaml
```

This command creates all of the objects in kubernetes including the deployments. Thus, it takes some time for them to start working.

<!-- *TODO: all-in-one.yaml analysis* -->

<!-- *TODO: How to compile EdgeNet from source?* -->