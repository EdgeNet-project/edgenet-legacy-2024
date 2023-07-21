# Cluster role request for a user

This support page describes whether and how you can request a *cluster role* in the cluster.
Your cluster role request is subject to the approval of the central administrators of EdgeNet. 
Upon approval of your request, you will receive permissions corresponding to the requested cluster role.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

In order to obtain your user-specific kubeconfig file, you need to take yourself to the [landing application](https://landing.edge-net.org).

## What you will do

You will authenticate yourself through the landing application to obtain your kubeconfig file from the landing application.
Using this kubeconfig file, you will create a *cluster role request* object that is associated with your e-mail address. 
This will alert the central administrators of EdgeNet, who will if all is in order, approve your request. 
With approval, corresponding permissions are granted through cluster role binding so as to allow your user to hold requested permissions.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Obtain your access credential

An EdgeNet cluster role request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file.

You can fetch your kubeconfig file here: [https://landing.edge-net.org](https://landing.edge-net.org). In what follows, we will assume that it is saved in your working directory on your system as ``./edgenet.cfg``.

Default permissions do not allow any actions beyond the creation of a tenant/role/cluster role request. Once the request goes through, you can start using EdgeNet as an ordinary user holding requested permissions.

### Prepare a description of your user

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the cluster role request object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future permissions:
- the **name** that will be seen by the EdgeNet central administrator; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing name in the namespace
- a **first name** (human readable)
- a **last name** (human readable)
- an **e-mail address**, which should be an institutional e-mail address
- a **rolename** must match the name of the ClusterRole you request to bind to

In what follows, we will assume that this file is saved in your working directory on your system as ``./clusterrolerequest.yaml``.

Example:
```yaml
apiVersion: registration.edgenet.io/v1alpha1
kind: ClusterRoleRequest
metadata:
  name: beratsenel
spec:
  firstname: Berat
  lastname: Senel
  email: berat.senel@lip6.fr
  rolename: edgenet:node-reader
```

### Create your role request

Using ``kubectl``, create a role request object:

```
kubectl create -f ./clusterrolerequest.yaml --kubeconfig ./edgenet.cfg
```

### Wait for approval and receive your corresponding permissions

At this point, the EdgeNet central administrators will, if needed, contact you, and, provided everything is in order, approve your role request. Upon approval, you will receive an email that confirms that your registration is complete and contains your user information.

You can now start using EdgeNet, holding the requested permissions, with your user-specific kubeconfig file.
