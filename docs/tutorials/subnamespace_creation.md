# Create a subsidiary namespace in EdgeNet

This tutorial describes how you can create a subsidiary namespace on EdgeNet, with yourself as the tenant owner or one of the tenant admins.

Each tenant has a tenant resource quota on EdgeNet to share the cluster resources equally. Once a tenant registers itself with EdgeNet, a core namespace is allocated to the tenant with an assigned resource quota. This resource quota is equal to the tenant's resource quota. Thereby, tenant users can deploy applications toward the cluster straightforward by core namespaces. Additionally, it is possible to collaborate with users in other tenants at the core namespace.

However, for cases where the collaboration at the core namespace is unfavorable, the subsidiary namespace feature allows tenants to establish a hierarchy among namespaces. The use of the subnamespaces is not only limited by the collaboration with other tenants. It is also useful when a tenant owner wants to allocate part of the tenant resource quota to a group of its users. Or for any other situation where teams needed.

In simple terms, a subnamespace generates a child namespace that inherits role-based access control and network policy settings from its parent and eats from the resource quota of the core namespace. An expiry date can also be defined if desired.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

## What you will do

You will use your user-specific kubeconfig file to create a *subsdiary namespace* object. Object creation generates a child namespace with a resource quota that you defined in the spec.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Be sure the user-specific access credential located well

An EdgeNet subspace is a Kubernetes object, and to manipulate objects on a Kubernetes system, you need a kubeconfig file. EdgeNet delivers the user-specific kubeconfig file via email after successful user registration.

The user-specific file does not allow any actions beyond the roles bound to your user. In case of an authorization issue, please contact your tenant administration.

### Prepare a description of your subsidiary namespace

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the subnamespace object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future subnamespace:
- the **subsidiary namespace name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing subnamepace names in your tenant
- the **resources** of the subnamespace; this field is not mandatory to define. The information provided consists of:
  - a **cpu** allocation
  - a **memory** allocation
- the **inheritance** of the subnamespace; the information provided consists of:
  - a **networkpolicy** inheritance from parent
  - a **rbac** inheritance from parent
- the **expiry** of the subnamespace; this should be the date that you want the subnamespace to expire. This field is not mandatory to define.

In what follows, we will assume that this file is saved in your working directory on your system as ``./subnamespace.yaml``.

Example:
```yaml
apiVersion: core.edgenet.io/v1alpha
kind: SubNamespaces
metadata:
  name: iris
spec:
  resources:
    cpu: 4000m
    memory: 4Gi
  inheritance:
    networkpolicy: true
    rbac: true
  expiry: "24/05/2021 18:00:00"
```

### Create your subsidiary namespace

Using ``kubectl``, create a subnamespace object:

```
kubectl create -f ./subnamespace.yaml --kubeconfig ./edgenet-kubeconfig.cfg
```
