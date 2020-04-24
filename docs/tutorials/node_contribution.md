# Contributing a Node into the EdgeNet cluster

This tutorial describes how a node can join EdgeNet. An authority administrator manages, perhaps by delegating the responsibilities, the nodes contributed by the authority itself or maybe authority users.

Besides the nodes provided by EdgeNet partners, each authority in EdgeNet can become a contributor to enlarge the cluster. An authority is able to allow their contributions to be *schedulable* or *unschedulable* for deployments.

As a contributor to the infrastructure, you will set up a VM and make it available to EdgeNet. Step by step, you will enable SSH server, create a new SSH user as a sudoer, and put our SSH public key into authorized keys or set a password that is not recommended. EdgeNet automates the installation and recovery processes of nodes.

The more you and other authorities contribute, the more of the global infrastructure all EdgeNet users can use. You can disable or remove the nodes that you contribute if and when you desire.

### A note on the operating systems we currently support

We strongly recommend that you choose *Ubuntu* or *CentOS* as the operating system on the server you want to add to the EdgeNet cluster. We tend to increase the supported operating systems.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

## What you will do

In the first place, please be sure your site, if exists, doesn't block the [required ports](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#check-required-ports). You will enable SSH server on your server, then create an EdgeNet user as a **sudoer**, and copy & paste the contents of our [SSH public key](https://github.com/EdgeNet-project/headnode/blob/release-1.0/config/id_rsa.pub) into authorized keys. Once you have done so, you will use the user-specific kubeconfig file provided by EdgeNet to create a *node contribution* object. Object creation generates an SSH client allowing the headnode to connect to your server running an SSH server. You don't need to do anything since the node contribution controller starts a pre-defined procedure including the installation of necessary packages and running the kubeadm join command.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Save your access credential

The user-specific kubeconfig file provided by EdgeNet when you registered in allows you to do node contributions. In what follows, we will assume that it is saved in your working directory on your system as ``./edgenet-kubeconfig.cfg``.

### Prepare a description of your node contribution

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the node contribution object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future node contribution:
- the **node name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing node names in your authority
- the **namespace** of the authority
- the **host** of the node contribution, which is a human-readable IP address in IPv4 or IPv6 format
- the **port** of the SSH server, which is 22 by default and recommended to allocate another port such as port 25020
- the **enabled** of the scheduling; which is a boolean, and this should be true if you want to open the node you contribute
- the **user** of the SSH client, which is the user you created on your server as a sudoer
- the **password** of the SSH user; you don't need to define that if you put our public SSH key exists in authorized keys

In what follows, we will assume that this file is saved in your working directory on your system as ``./nodecontribution.yaml``.

Example:
```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: NodeContribution
metadata:
  name: ple-1
  namespace: authority-lip6-lab
spec:
  host: 132.227.123.46
  port: 25010
  enabled: true
  user: edgenet
```

### Do your node contribution

Using ``kubectl``, create a node contribution object:

```
kubectl create -f ./nodecontribution.yaml --kubeconfig ./edgenet-kubeconfig.cfg
```

This will start a procedure including the automated installation processes.

### Check the installation status of the node contribution

At this point, EdgeNet lists status messages to inform you about installation steps. You can have status details by invoking the kubectl describe command with a node name and your authority namespace that defined as **ple-1** and **authority-lip6-lab** in this tutorial.

Using ``kubectl``, get status details of a node contribution object:

```
kubectl describe nodecontribution ple-1 -n authority-lip6-lab --kubeconfig ./edgenet-kubeconfig.cfg
```
