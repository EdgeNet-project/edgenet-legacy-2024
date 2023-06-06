# Create a subsidiary namespace in EdgeNet

This tutorial describes how you can create a subsidiary namespace on EdgeNet, with yourself as the tenant owner or one of the tenant admins.

Each tenant has a tenant resource quota on EdgeNet to share the cluster resources fairly. Once a tenant registers itself with EdgeNet, a core namespace is allocated to the tenant with an assigned overall resource quota. This resource quota is equal to the tenant's resource quota and applies to its whole namespace tree regarding hierarchical namespaces. Thereby, tenant users can deploy applications toward the cluster straightforwardly by core namespaces. Additionally, it is possible to collaborate with users in other tenants at the core namespace.

However, for cases where the collaboration at the core namespace is unfavorable, the subsidiary namespace feature allows tenants to establish a hierarchy among namespaces. The use of the subnamespaces is not only limited by the collaboration with other tenants. It is also useful when a tenant owner wants to allocate part of the tenant resource quota to a group of its users. Or for any other situation where teams are needed.

In simple terms, a subnamespace generates a child namespace that inherits role-based access control and network policy settings from its parent by default and eats from the resource quota of the core namespace. An expiry date can also be defined if desired.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

## What you will do

You will use your user-specific kubeconfig file to create a *subsidiary namespace* object. Object creation generates a child namespace with a resource quota that you defined in the spec.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Be sure the user-specific access credential is located well

An EdgeNet subnamespace is a Kubernetes object, and to manipulate objects on a Kubernetes system, you need a kubeconfig file.
You can fetch your kubeconfig file here: [https://landing.edge-net.org](https://landing.edge-net.org). In what follows, we will assume that it is saved in your working directory on your system as ``./edgenet.cfg``.

The user-specific file does not allow any actions beyond the roles bound to your user. In case of an authorization problem, please contact your tenant administration.

### Decide what kind of subnamespace you need

The subnamespaces feature offers two types of tenancy; consumer and vendor. 
The consumer one provides workspaces for members of the tenant such as teams and departments, whereas the vendor one shapes out a subtenant to ensure data privacy.
You must decide by which type you are creating a subnamespace because changing the type is not allowed after creation.

### Prepare a description of your subsidiary namespace

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the subnamespace object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future subnamespace:

#### Workspace

- the **subsidiary namespace** name that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing subnamepace names in the namespace
- the **parent namespace** name in which you want to create a subnamespace; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)
- the **workspace** is the type of tenancy mentioned above
  - the **resource allocation** that will be used to assign a quota; resources here must be compatible with [Kubernetes resource types](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-types)
  - the **inheritance** that will be used to inherent resources from the parent; the information you need to provide consists of:
    - a **networkpolicy** inheritance from parent
    - a **rbac** inheritance from parent
    - a **limitrange** inheritance from parent
    - a **configmap** inheritance from parent
  - the **sync** is to make continuous reconciliation between parent and child, which is a boolean
  - the **slice claim** that will be used to bind node-level slice, a subcluster, to the subnamespace
- the **expiry** of the subnamespace; this should be in the dateTime format of [RFC3339](https://xml2rfc.tools.ietf.org/public/rfc/html/rfc3339.html#anchor14). This field is not mandatory to define.

In what follows, we will assume that this file is saved in your working directory on your system as ``./subnamespace.yaml``.

Example:
```yaml
apiVersion: core.edgenet.io/v1alpha1
kind: SubNamespace
metadata:
  name: netmet
  namespace: lip6-lab
spec:
  workspace:
    resourceallocation:
      cpu: "4000m"
      memory: "4Gi"
    inheritance:
      rbac: true
      networkpolicy: false
      limitrange: true
      configmap: true      
    sync: false
    sliceclaim: lab-exercises
  expiry: "2023-09-01T09:00:00Z"
```

#### Subtenant

- the **subsidiary namespace** name that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing subnamepace names in the namespace
- the **parent namespace** name in which you want to create a subnamespace; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)
- the **subtenant** is the type of tenancy mentioned above
  - the **resource allocation** that will be used to assign a quota; resources here must be compatible with [Kubernetes resource types](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-types)
  - the **owner** is a person who is both administrator and a regular user of this subtenant 
  - the **slice claim** that will be used to bind node-level slice, a subcluster, to the subnamespace
- the **expiry** of the subnamespace; this should be in the dateTime format of [RFC3339](https://xml2rfc.tools.ietf.org/public/rfc/html/rfc3339.html#anchor14). This field is not mandatory to define.

In what follows, we will assume that this file is saved in your working directory on your system as ``./subnamespace.yaml``.

Example:
```yaml
apiVersion: core.edgenet.io/v1alpha1
kind: SubNamespace
metadata:
  name: netmet
  namespace: lip6-lab
spec:
  subtenant:
    resourceallocation:
      cpu: "4000m"
      memory: "4Gi"
    owner:
      firstname: Berat
      lastname: Senel
      email: berat.senel@lip6.fr
      phone: "+33123456789"
    sliceclaim: lab-exercises
  expiry: "2023-09-01T09:00:00Z"
```

### Create your subsidiary namespace

Using ``kubectl``, create a subnamespace object:

```
kubectl create -f ./subnamespace.yaml --kubeconfig ./edgenet.cfg
```
