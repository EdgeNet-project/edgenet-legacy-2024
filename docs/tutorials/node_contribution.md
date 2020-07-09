# Contributing a Node to the EdgeNet cluster

EdgeNet thrives thanks to contributions from its users and from others who volunteer resources in support of the system. This tutorial describes the very simple process of adding a node to EdgeNet. If you are able to set up a virtual machine (VM) on a server that you administer, then you can contribute, which will be a great benefit to our growing infrastructure and to all EdgeNet users.

Each node falls under what we call an *authority* in EdgeNet, which is a group, or possibly just one person, that takes responsibility for users and/or resources. To contribute a node, please be sure that you are an EdgeNet authority administrator, or that an administrator of your authority has authorized you to make node contributions.

## Preliminaries

To contribute a node, we assume that you already know how to set up a VM with its own public IP address and ensure that certain of its port numbers are accessible from the internet.

### Set up a VM

EdgeNet is currently accepting nodes that run recent *Ubuntu* or *CentOS* distributions. We plan to broaden our range of supported operating systems over time.

### Open your firewall

EdgeNet is most useful to researchers if its nodes are entirely open to the internet, without a firewall blocking incoming traffic in any way. This is easiest if your server is in a perimeter network, sometimes called a *DMZ* or *science DMZ*. If you are contributing resources from elsewhere, please do your best, within the limits of what your institution allows, or, if the server is in your home, what you yourself are prepared to offer.

At a minimum, you may only contribute a node if the Kubernetes [required ports](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#check-required-ports) are accessible from the internet.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

You will most likely have used ``kubectl`` to create your EdgeNet user account to begin with. If you need to install it again, please see the releavant [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/).

You will also have received a user-specific kubeconfig file when you created your EdgeNet user account. You will authenticate with this kubeconfig file when you make a node contribution. In what follows, we will assume that it is saved in your working directory on your system as ``./edgenet-kubeconfig.cfg``. If it is elsewhere, please be sure to modify the commands accordingly.


## What you will do

You will set up EdgeNet access via SSH to your VM, and then invoke ```kubectl create``` to create a *node contribution* object, which causes the VM to be integrated as a node into EdgeNet's Kubernetes cluster.


## Steps

### Set up EdgeNet's SSH access to your VM

Enable an SSH server on your VM, preferably on a port number other than the default port number of 22. You could use port 25020, for instance.

Create an EdgeNet user (the username does not matter) as a **sudoer**.

Copy & paste the contents of [the EdgeNet public key](https://github.com/EdgeNet-project/edgenet/blob/master/config/id_rsa.pub) into the SSH authorized keys file for the EdgeNet user.

**P.S.** The user should be allowed to run commands by using *sudo* without being prompted for the password.

### Prepare a description of your node contribution

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the node contribution object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your node contribution:
- the node **name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing node names in your authority
- the **namespace** of the authority
- the **host** IP address of the node contribution, in human-readable (ASCII) IPv4 or IPv6 format
- the **port** number of the SSH server
- whether scheduling of the nodes is **enabled**, which is a boolean, with ```true``` allowing the node to participate in the cluster
- the SSH **user**, which is the username of the sudoer that you set up on the VM
- the **password** of the SSH user; provide this only if for some reason you are not able to enable SSH access via the EdgeNet public key

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

#### Node naming pattern

The node name pattern in use is `<authority-name>.<node-contribution-name>.edge-net.io` to provide a node list grouping the authorities. According to the example above, the node name would appear as **lip6-lab.ple-1.edge-net.io**.

### Make your node contribution

Using ``kubectl``, create a node contribution object:

```
kubectl create -f ./nodecontribution.yaml --kubeconfig ./edgenet-kubeconfig.cfg
```

This will launch the automated installation of Kubernetes on the VM and the VM's integration as a node into EdgeNet's Kubernetes cluster. The EdgeNet headnode connects via SSH to the user account that you created on the VM. As a sudoer, that user installs the necessary packages and runs the ```kubeadm join``` command.


### Check the installation status

Follow the status messages as each installation step is completed.

You can at any time check on the status of your node contribution by invoking the ```kubectl describe``` command. In this example, the node name is **ple-1** and the authority namespace is **authority-lip6-lab**:

```
kubectl describe nodecontribution ple-1 -n authority-lip6-lab --kubeconfig ./edgenet-kubeconfig.cfg
```

#### In case of failure

If you encounter the state of **Failure** on the status of your node contribution, please make sure that you follow the instructions correctly. When you make sure you correctly follow the instructions, you can delete the node contribution object and recreate it as below:

```
kubectl delete -f ./nodecontribution.yaml --kubeconfig ./edgenet-kubeconfig.cfg
```

```
kubectl create -f ./nodecontribution.yaml --kubeconfig ./edgenet-kubeconfig.cfg
```

In case of any other issue, please contact our support team by [opening our tawk.to window](https://tawk.to/edgenet).

## Legal rights

The use of EdgeNet is subject to the user's acceptance of the Acceptable Use Policy. Please [click here](https://edge-net.org/aup.html) to reach the AUP.

In addition to this, the containers in EdgeNet doesn't hold root privileges by default.

Furthermore, we will put the EdgeFlow solution inherited from PlanetFlow in place to track all activity in the next releases.
