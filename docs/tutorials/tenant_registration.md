# Registering an EdgeNet tenant

This tutorial describes whether and how you can set up an *tenant* on EdgeNet, with yourself as the tenant owner. A tenant owner takes responsibility for the approval of EdgeNet users who they can vouch for.

Authorizations to use EdgeNet are handed out hierarchically, establishing a chain of responsibility. We, as the central administrators of EdgeNet, approve the establishment of tenants and their owners. An owner, in turn, approves the creation of individual user accounts. The owner can give some of those users administrative rights to, in turn, approve other users.

Our job is to ensure that only bona fide researchers can create and run local tenants. If you wish to create a tenant, please be sure to register with your institutional e-mail address, and please direct us to an institutional website or provide other evidence of your role. In general, we approve faculty members at institutions of higher education and senior researchers at research centers, but we will consider other cases as well.

A local tenant owner also approves the creation of *subsidiary namespaces*, which allows to share tenant resource quota with a group of users. And an owner manages, either directly or through a user to whom they delegate this role, any *nodes* that are contributed on behalf of the tenant to the EdgeNet cluster.

If you believe that you may be eligible to act as the local owner of a tenant, the remainder of this tutorial guides you through the registration process.

If you would like to use EdgeNet but do not believe that you can act as a local owner, we encourage you to identify someone at your institution who is already an owner, or who would be willing to register as one.

### A note on terminology for PlanetLab users

For those of you familiar with PlanetLab, a tenant is similar to a *site* and a local tenant owner is similar to a *PI*.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface, in conjunction with e-mail.

Or, you can register via [the console](https://console.edge-net.org/signup) with an attractive user interface design to facilitate the process. If you take yourself to the console, you no longer need to follow these instructions as it provides you a classical registration procedure.

## What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert EdgeNet's central administrators, who will, if all is in order, approve your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as both the local owner and a user of your tenant.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Obtain a temporary access credential

An EdgeNet tenant request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a public kubeconfig file that anyone can use for the prupose of creating tenant requests.

This public kubeconfig file is available here: [https://edge-net.org/downloads/config/public.cfg](https://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved in your working directory on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation of a tenant request and the use of the one-time code to confirm the request. Once the request goes through, you will be provided with another kubeconfig file that is specific to you and that will allow you to carry out adminstrative actions having to do with your tenant, as well as to use EdgeNet as an ordinary user.

### Prepare a description of your tenant

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the tenant request object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future tenant:
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

### Create your tenant request

Using ``kubectl``, create a tenant request object:

```
kubectl create -f ./tenantrequest.yaml --kubeconfig ./public.cfg
```

This will cause an e-mail containing a one-time code to be sent to the address that you specified.

### Authenticate your request using a one-time code

The e-mail that you receive will contain a ``kubectl`` command that you can copy and paste onto your command line, editing only the path for the public kubeconfig file on your local system, if needed.

In the example here, the one-time code is ``bsv10kgeyo7pmazwpr``:

```
kubectl patch emailverification bsv10kgeyo7pmazwpr --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public.cfg
```

After you have done this, the EdgeNet system sends a notification e-mail to EdgeNet's central administrators, informing them of your registration request.

### Wait for approval and receipt of your permanent access credential

At this point, the EdgeNet central administrators will, if needed, contact you, and, provided everything is in order, approve your registration request. Upon approval, you will receive two emails. The first one confirms that your registration is complete, while the second one contains your user information and user-specific kubeconfig file.

You can now start using EdgeNet, as both administrator of your local tenant and as a regular user, with your user-specific kubeconfig file.
