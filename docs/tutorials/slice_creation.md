# Create a slice at the node granularity in EdgeNet

This tutorial describes how you can create a node-level slice on EdgeNet, which forms a subcluster, with yourself as the tenant owner or one of the tenant or subnamespace admins.

The EdgeNet tenants, on a best-effort basis, share the cluster. It allows multiple tenants to take advantage of existing infrastructure. However, certain deployments require complete isolation and flexible configuration. Some experiments also require reproducibility.

In order to meet these requirements, EdgeNet provides you with Slice and a Slice Claim custom resources that allow slicing among nodes by defining a selector. These nodes, forming a subcluster, are completely isolated from multitenant workloads and are dedicated to the slice owner. An expiry date can also be defined if desired.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

## What you will do

You will use your user-specific kubeconfig file to create a *slice claim* object and then a *subnamespace* object. Creating these objects in sequence reserves the nodes that satisfy the selector and cleans them from multitenant workloads.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Be sure the user-specific access credential is located well

An EdgeNet slice claim is a Kubernetes object, and to manipulate objects on a Kubernetes system, you need a kubeconfig file.
You can fetch your kubeconfig file here: [https://landing.edge-net.org](https://landing.edge-net.org). In what follows, we will assume that it is saved in your working directory on your system as ``./edgenet.cfg``.

The user-specific file does not allow any actions beyond the roles bound to your user. In case of an authorization problem, please contact your tenant administration.

### Prepare a description of your slice claim at the node granularity

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the slice claim object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future node-level slice:

- the **slice claim** name that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing slice claim names in the namespace
- the **namespace** name in which you want to create a slice claim; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)
- the **slice** name is the name of the slice that will be dynamically created; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)
- the **node selector** is to choose the nodes by using the labels; the selector must be compatible with [Kubernetes Node Selector](https://pkg.go.dev/k8s.io/api@v0.21.2/core/v1?utm_source=gopls#NodeSelector)
- the **expiry** of the slice; this should be in the dateTime format of [RFC3339](https://xml2rfc.tools.ietf.org/public/rfc/html/rfc3339.html#anchor14). This field is not mandatory to define.

In what follows, we will assume that this file is saved in your working directory on your system as ``./sliceclaim.yaml``.

Example:
```yaml
apiVersion: core.edgenet.io/v1alpha1
kind: SliceClaim
metadata:
  name: netmet
  namespace: lip6-lab
spec:
  slicename: lip6-lab-netmet
  nodeselector:
    selector:
      nodeSelectorTerms:
      - matchExpressions:
          - key: edge-net.io/continent
            operator: In
            values:
            - Europe
    nodecount: 2
    resources:
      requests:
        cpu: 4
        memory: "4000Mi"
      limits:
        cpu: 8
        memory: "20000Mi"
  expiry: "2023-09-01T09:00:00Z"
```

### Create your slice claim

Using ``kubectl``, create a slice claim object:

```
kubectl create -f ./sliceclaim.yaml --kubeconfig ./edgenet.cfg
```
### Bind your slice to a subnamespace

Please refer to the [subnamespace creation](subnamespace_creation.md) document to bind your slice to a subnamespace.