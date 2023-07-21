# RTT measurement tutorial on EdgeNet

This tutorial describes whether and how you can make RTT measurements on EdgeNet. You need to register with EdgeNet as a first step.

Our job is to ensure that only bona fide researchers can create and run local tenants. If you wish to create a tenant, please be sure to register with your institutional e-mail address, and please direct us to an institutional website or provide other evidence of your role. In general, we approve faculty members at institutions of higher education and senior researchers at research centers, but we will consider other cases as well.

If you believe that you may be eligible to act as the local owner of a tenant, please go through [Registering a tenant](#registering-a-tenant) for the registration.

If you would like to use EdgeNet but do not believe that you can act as a local owner, we encourage you to identify someone at your institution who is already an owner, or who would be willing to register as one. Once the registration of the tenant where you want to join is completed, please follow the instructions at [Registering a user in a tenant](#registering-a-user-in-a-tenant).

1. [Prerequisites](#prerequisites)
1. [Registering a tenant](#registering-a-tenant)
1. [Registering a user in a tenant](#registering-a-user-in-a-tenant)
1. [Deploying containers](#deploying-containers)

## Prerequisites

To follow this tutorial you will need [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface.

- [Install kubectl on Linux](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux)
- [Install kubectl on macOS](https://kubernetes.io/docs/tasks/tools/install-kubectl-macos)
- [Install kubectl on Windows](https://kubernetes.io/docs/tasks/tools/install-kubectl-windows)

## Registering a tenant

In this section we show how you can set up a *tenant* on EdgeNet, with yourself as the tenant owner. A tenant owner takes responsibility for the approval of EdgeNet users who they can vouch for.

Authorizations to use EdgeNet are handed out hierarchically, establishing a chain of responsibility. We, as the central administrators of EdgeNet, approve the establishment of tenants and their owners. An owner, in turn, approves the creation of individual user accounts. The owner can give some of those users administrative rights to, in turn, approve other users.

A local tenant owner also approves the creation of *subsidiary namespaces*, which allows sharing tenant resource quotas with a group of users. And an owner manages, either directly or through a user to whom they delegate this role, any *nodes* that are contributed on behalf of the tenant to the EdgeNet cluster.

#### A note on terminology for PlanetLab users

For those of you familiar with PlanetLab, a tenant is similar to a *site* and a local tenant owner is similar to a *PI*.

### What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert EdgeNet's central administrators, who will, if all is in order, approve your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as both the local owner and a user of your tenant.

### Steps

#### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

#### Obtain a temporary access credential

An EdgeNet tenant request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a public kubeconfig file that anyone can use for the prupose of creating tenant requests.

This public kubeconfig file is available here: [https://edge-net.org/downloads/config/public.cfg](https://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved in your working directory on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation of a tenant request and the use of the one-time code to confirm the request. Once the request goes through, you will be provided with another kubeconfig file that is specific to you and that will allow you to carry out adminstrative actions having to do with your tenant, as well as to use EdgeNet as an ordinary user.

#### Prepare a description of your tenant

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the tenant request object, following the model of the example shown below. Your ``.yaml`` file must specify the following information regarding your future tenant:
- the **tenant name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet tenant names
- the **full name** of the tenant, which is a human-readable name
- the **short name** of the tenant, which is also human-readable, and can be the same as the full name, or a shorter name, in case the full name is long
- the **URL** of the tenant; this should be a web page from your institution that confirms your role as a bona fide researcher
- the **postal address** of the tenant; the information provided for this person consists of:
  - a **street** address
  - a **ZIP** code/postal code
  - a **city** name
  - a **region**, or state name (not mandatory)
  - a **country** name
- the **contact person** who is the responsible for this tenant; this is the tenant's first administrator, who is typically yourself; the information provided for this person consists of:
  - a **username** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/); note that usernames need only be distinct within a tenant
  - a **first name** (human readable)
  - a **last name** (human readable)
  - an **e-mail address**, which should be an institutional e-mail address
  - a **phone number**, which should be in quotation marks, start with the country code using the plus notation, and not contain any spaces or other formatting

In what follows, we will assume that this file is saved in your working directory on your system as ``./tenantrequest.yaml``.

Example:
```yaml
apiVersion: registration.edgenet.io/v1alpha
kind: TenantRequest
metadata:
  name: lip6-lab
spec:
  fullname: Laboratoire LIP6-CNRS
  shortname: lip6
  url: https://www.lip6.fr/recherche/team_membres.php?acronyme=NPA
  address:
    street: 4 place Jussieu, boite 169
    zip: "75005"
    city: Paris
    region: Île-de-France
    country: France
  contact:
    username: timurfriedman
    firstname: Timur
    lastname: Friedman
    email: timur.friedman@sorbonne-universite.fr
    phone: "+33123456789"
```

#### Create your tenant request

Using ``kubectl``, create a tenant request object:

```bash
kubectl create -f ./tenantrequest.yaml --kubeconfig ./public.cfg
# tenantrequest.registration.edgenet.io/lip6-lab created
```

This will cause an e-mail containing a one-time code to be sent to the address that you specified.

#### Authenticate your request using a one-time code

The e-mail that you receive will contain a ``kubectl`` command that you can copy and paste onto your command line, editing only the path for the public kubeconfig file on your local system, if needed.

In the example here, the one-time code is ``bsv10kgeyo7pmazwpr``:

```bash
kubectl patch emailverification bsv10kgeyo7pmazwpr --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public.cfg
# emailverification.registration.edgenet.io/bsv10kgeyo7pmazwpr patched
```

After you have done this, the EdgeNet system sends a notification e-mail to EdgeNet's central administrators, informing them of your registration request.

#### Wait for approval and receipt of your permanent access credential

At this point, the EdgeNet central administrators will, if needed, contact you, and, provided everything is in order, approve your registration request. Upon approval, you will receive two emails. The first one confirms that your registration is complete, while the second one contains your user information and user-specific kubeconfig file.

The second email will contain the instructions to approve the [Acceptable Use Policy](https://www.edge-net.org/pages/usage-policy.html).
Download the attached `edgenet-kubeconfig.cfg` file and run the command provided in the email.
For example, for the `timurfriedman` user:
```bash
kubectl patch aup timurfriedman --type='json' -p='[{"op": "replace", "path": "/spec/accepted", "value": true}]' --kubeconfig ./edgenet-kubeconfig.cfg
# acceptableusepolicy.core.edgenet.io/timurfriedman patched
```

You can now start using EdgeNet, as both administrator of your local tenant and as a regular user, with your user-specific kubeconfig file.

## Registering a user in a tenant

In this section we show how you can register a *user* in a tenant with EdgeNet.
Your registration in a tenant is subject to the approval of that tenant's administrator. However, anyone
who wants to use EdgeNet can make registration request in a tenant only to become a user.

### What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert the tenant administrators, who will, if all is in order, approve your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as a user of the tenant from which you make the request.

### Steps

#### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

#### Obtain a temporary access credential

An EdgeNet user registration request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a public kubeconfig file that anyone can use for the purpose of creating user registration requests.

This public kubeconfig file is available here: [https://edge-net.org/downloads/config/public.cfg](https://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved in your working directory on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation of a user registration request and the use of the one-time code to confirm the request. Once the request goes through, you will be provided with another kubeconfig file that is specific to you and that will allow you to use EdgeNet as an ordinary user.

#### Prepare a description of your user

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the user registration request object, following the model of the example shown below. Your ``.yaml`` file must specify the following information regarding your future tenant:
- the **username** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet tenant names
- the **tenant** name; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/)
- a **first name** (human readable)
- a **last name** (human readable)
- an **e-mail address**, which should be an institutional e-mail address

In what follows, we will assume that this file is saved in your working directory on your system as ``./userregistrationrequest.yaml``.

Example:
```yaml
apiVersion: registration.edgenet.io/v1alpha
kind: UserRequest
metadata:
  name: bsenel
spec:
  tenant: gdr-tutorial
  firstname: Berat
  lastname: Senel
  email: berat.senel@lip6.fr
```

#### Create your user registration request

Using ``kubectl``, create a user registration request object:

```bash
kubectl create -f ./userregistrationrequest.yaml --kubeconfig ./public.cfg
# userrequest.registration.edgenet.io/bsenel created
```

This will cause an e-mail containing a one-time code to be sent to the address that you specified.

#### Authenticate your request using a one-time code

The e-mail that you receive will contain a ``kubectl`` command that you can copy and paste onto your command line, editing only the path for the public kubeconfig file on your local system, if needed.

In the example here, the one-time code is ``bsv10kgeyo7pmazwpr``:

```bash
kubectl patch emailverification bsv10kgeyo7pmazwpr --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public.cfg
# emailverification.registration.edgenet.io/bsv10kgeyo7pmazwpr patched
```

After you have done this, the EdgeNet system sends a notification e-mail to the tenant administrators, informing them of your registration request.

#### Wait for approval and receipt of your permanent access credential

At this point, the tenant administrators will, if needed, contact you, and, provided everything is in order, approve your registration request. Upon approval, you will receive an email that confirms that your registration is complete, and contains your user information and user-specific kubeconfig file.

The email will contain the instructions to approve the [Acceptable Use Policy](https://www.edge-net.org/pages/usage-policy.html).
Download the attached `edgenet-kubeconfig.cfg` file and run the command provided in the email.
For example, for the `bsenel` user:
```bash
kubectl patch aup bsenel --type='json' -p='[{"op": "replace", "path": "/spec/accepted", "value": true}]' --kubeconfig ./edgenet-kubeconfig.cfg
# acceptableusepolicy.core.edgenet.io/bsenel patched
```

You can now start using EdgeNet, as a regular user, with your user-specific kubeconfig file.

## Deploying containers

The novel multi-tenancy model allows users to deploy their pods (containers) across the cluster straightforwardly via the core namespaces. You have been notified about your core namespace via notification email. You can access your core namespace with your tenant's name because it gets the name from your tenant.

### Creating a selective deployment

EdgeNet is shining out with the geodiversity of its cluster. To take advantage of it, **Selective Deployment** is a feature that EdgeNet brings to Kubernetes to allow users to deploy pods onto nodes based on their locations.

In this tutorial, you will use the ping tool to measure RTT between a destination and sources. Therefore, you need to prepare two selective deployments, one for the destination and one for the sources. Here is an example ``selectivedeployment.yaml`` file:

```yaml
# selectivedeployment.yaml
apiVersion: apps.edgenet.io/v1alpha
kind: SelectiveDeployment
metadata:
  name: rtt-experiment-destination-<username>
  namespace: gdr-tutorial
spec:
  workloads:
    deployment:
      - apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: ping-destination-<username>
          namespace: gdr-tutorial
          labels:
            app: ping-destination-<username>
        spec:
          replicas: 1
          selector:
            matchLabels:
              app: ping-destination-<username>
          template:
            metadata:
              labels:
                app: ping-destination-<username>
            spec:
              tolerations:
                - key: node-role.kubernetes.io/master
                  operator: Exists
                  effect: NoSchedule
              containers:
                - name: ping-destination-<username>
                  image: busybox
                  command: ['/bin/sh', '-c', 'sleep infinity']
                  resources:
                    limits:
                      cpu: 50m
                      memory: 50Mi
                    requests:
                      cpu: 50m
                      memory: 50Mi
              terminationGracePeriodSeconds: 0
  selector:
    - value:
        - North_America
      operator: In
      quantity: 1
      name: Continent
---
apiVersion: apps.edgenet.io/v1alpha
kind: SelectiveDeployment
metadata:
  name: rtt-experiment-source-<username>
  namespace: gdr-tutorial
spec:
  workloads:
    daemonset:
      - apiVersion: apps/v1
        kind: DaemonSet
        metadata:
          name: ping-source-<username>
          namespace: gdr-tutorial
          labels:
            app: ping-source-<username>
        spec:
          selector:
            matchLabels:
              app: ping-source-<username>
          template:
            metadata:
              labels:
                app: ping-source-<username>
            spec:
              tolerations:
                - key: node-role.kubernetes.io/master
                  operator: Exists
                  effect: NoSchedule
              containers:
                - name: ping-source-<username>
                  image: busybox
                  command: ['/bin/sh', '-c', 'sleep infinity']
                  resources:
                    limits:
                      cpu: 50m
                      memory: 50Mi
                    requests:
                      cpu: 50m
                      memory: 50Mi
              terminationGracePeriodSeconds: 0
  selector:
    - value:
        - North_America
      operator: In
      quantity: 2
      name: Continent
    - value:
        - Europe
      operator: In
      quantity: 1
      name: Continent
```

When the ``selectivedeployment.yaml`` file is ready, you can create it as below:

```bash
kubectl create -f selectivedeployment.yaml --kubeconfig edgenet-kubeconfig.cfg
# selectivedeployment.apps.edgenet.io/rtt-experiment-destination created
# selectivedeployment.apps.edgenet.io/rtt-experiment-source created
```

To delete it:

```bash
kubectl delete -f selectivedeployment.yaml --kubeconfig edgenet-kubeconfig.cfg
# selectivedeployment.apps.edgenet.io "rtt-experiment-destination" deleted
# selectivedeployment.apps.edgenet.io "rtt-experiment-source" deleted
```

### Monitoring the deployment

At this step, you will verify the deployment status. We omit ``--kubeconfig`` and ``-n`` options for brevity here.

You can check the statuses of selective deployments (sd) as below:

```bash
kubectl get sd
# NAME                         READY   STATUS    AGE
# rtt-experiment-destination   1/1     Running   35s
# rtt-experiment-source        1/1     Running   35s
```

```bash
kubectl describe sd rtt-experiment-destination
# Name:         rtt-experiment-destination
# Namespace:    ...
# Labels:       <none>
# Annotations:  <none>
# API Version:  apps.edgenet.io/v1alpha
# Kind:         SelectiveDeployment
# Metadata: ...
```

View the statuses of deployment and daemonset:

```bash
kubectl describe deployment ping-destination
# Name:                   ping-destination
# Namespace:              ...
# CreationTimestamp:      Thu, 10 Jun 2021 09:36:41 +0200
# Labels:                 app=ping-destination
# Annotations:            deployment.kubernetes.io/revision: 1
# Selector:               app=ping-destination
# Replicas:               1 desired | 1 updated | 1 total | 1 available | 0 unavailable
# StrategyType:           RollingUpdate
# ...
```

```bash
kubectl describe daemonset ping-source
# Name:           ping-source
# Selector:       app=ping-source
# Node-Selector:  <none>
# Labels:         app=ping-source
# Annotations:    deprecated.daemonset.template.generation: 1
# Desired Number of Nodes Scheduled: 3
# Current Number of Nodes Scheduled: 3
# Number of Nodes Scheduled with Up-to-date Pods: 3
# Number of Nodes Scheduled with Available Pods: 2
# Number of Nodes Misscheduled: 0
# Pods Status:  2 Running / 1 Waiting / 0 Succeeded / 0 Failed
# ...
```

It is also possible to list the pods:

```bash
kubectl get pods -o wide
# NAME                                READY   STATUS    RESTARTS   AGE     IP                NODE                             
# ping-destination-67d885476d-bdbhp   1/1     Running   0          2m48s   192.168.190.255   bbn-1.edge-net.io
# ping-source-57t9g                   1/1     Running   0          2m48s   192.168.62.42     aws-eu-west-1a-7fe2.edge-net.io
# ping-source-7hlt2                   1/1     Running   0          2m48s   192.168.255.111   case-1.edge-net.io
# ping-source-xb7cb                   1/1     Running   0          2m48s   192.168.190.203   bbn-1.edge-net.io
```

To get the logs of a pod:

```bash
kubectl logs POD_NAME
```

To get a shell to a running container:

```bash
kubectl exec -it POD_NAME -- sh
```

### Using the ping command

From this point on, you will ping toward the destination from the sources by the destination pod's internal IP address and the node's external IP address.

#### Ping to Internal IP address

Retrieve the destination pod's internal IP address by:

```bash
kubectl get pods -l app=ping-destination -o jsonpath='{.items[0].status.podIP}'
# 192.168.190.204 (example)
```

List the source pod names:

```bash
kubectl get pods -l app=ping-source -o name | cut -d'/' -f2
# ping-source-5fn22 (example)
# ping-source-r6d9j (example)
# ping-source-w6pvp (example)
```

Execute `ping` inside the container:

```bash
kubectl exec -it POD_NAME -- ping DESTINATION_INTERNAL_IP -c 10
# PING 192.168.190.205 (192.168.190.205): 56 data bytes
# 64 bytes from 192.168.190.205: seq=0 ttl=62 time=27.660 ms
# 64 bytes from 192.168.190.205: seq=1 ttl=62 time=27.592 ms
# 64 bytes from 192.168.190.205: seq=2 ttl=62 time=27.430 ms
# ...
```

#### Ping to External IP address

Retrieve the destination pod's internal IP address by:

```bash
kubectl get pods -l app=ping-destination -o jsonpath='{.items[0].status.hostIP}'
# 192.1.242.150 (example)
```

List the source pod names:

```bash
kubectl get pods -l app=ping-source -o name | cut -d'/' -f2
# ping-source-5fn22 (example)
# ping-source-r6d9j (example)
# ping-source-w6pvp (example)
```

Execute `ping` inside the container:

```bash
kubectl exec -it POD_NAME -- ping DESTINATION_EXTERNAL_IP -c 10
# PING 192.1.242.150 (192.1.242.150): 56 data bytes
# 64 bytes from 192.1.242.150: seq=0 ttl=54 time=27.654 ms
# 64 bytes from 192.1.242.150: seq=1 ttl=54 time=27.395 ms
# 64 bytes from 192.1.242.150: seq=2 ttl=54 time=27.463 ms
```
