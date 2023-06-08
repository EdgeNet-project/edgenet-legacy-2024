# Removing EdgeNet from a Kubernetes Cluster

This documentation describes how to remove a working EdgeNet cluster in your environment. You can refer to this tutorial for deploying EdgeNet software to a local, sandbox, or production cluster.

## Technologies you will use

To remove EdgeNet to a Kubernetes cluster you need to have [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured. 

## What will you do

EdgeNet extension for Kubernetes consists of two parts, the custom resource definitions ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) and custom controllers. 

CRDs (discussed [here](/docs/README.md#extending-kubernetes)) are custom objects that are manipulated, created, and destroyed by custom controllers.

You will be removing CRDs and custom controllers from the cluster.


## Removal
EdgeNet is designed portable thus if you only require certain features, it is possible to install specific features of EdgeNet. Please refer to the [advanced installation tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) for the installation of specific features. 

For removal, it is important to know which EdgeNet version is installed. If you have installed EdgeNet by cloning the git repository, you can see [removing EdgeNet by cloned git repository](#remove-edgenet-by-cloned-repository). 

If you installed EdgeNet directly by the main readme's [creating and EdgeNet cluster](/README.md#create-an-edgenet-cluster) section, you can [remove EdgeNet directly](#remove-edgenet-directly).

---
### Remove EdgeNet by Cloned Repository
If you followed the [advanced installation tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) and cloned a repository to install EdgeNet, we recommend you uninstall using the yaml files.

To remove go to the directory of the cloned EdgeNet repository. Then run the following command to remove all of the objects created with the installment of EdgeNet.

```bash
kubectl -f delete ./build/yamls/kubernetes/all-in-one.yaml
```

---

### Remove EdgeNet Directly
If you followed the ['Create an EdgeNet Cluster' tutorial in the main readme](/README.md#create-an-edgenet-cluster) and installed EdgeNet using the URL, we recommend you uninstall using the same way

It is important to know which branch you installed EdgeNet from. By default, the branch is `main`.

To remove go to the directory of the cloned EdgeNet repository. Then run the following command to remove all of the objects created with the installment of EdgeNet.

```bash
RELEASE=main

kubectl delete -f https://raw.githubusercontent.com/EdgeNet-project/edgenet/$RELEASE/build/yamls/kubernetes/all-in-one.yaml
```