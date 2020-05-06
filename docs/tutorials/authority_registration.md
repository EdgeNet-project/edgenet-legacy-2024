# Registering an EdgeNet authority

This tutorial describes whether and how you can set up an *authority* on EdgeNet, with yourself as the authority administrator. An authority administrator takes responsibility for the approval of EdgeNet users who they can vouch for.

Authorizations to use EdgeNet are handed out hierarchically, establishing a chain of responsibility. We, as the central administrators of EdgeNet, approve the establishment of authorities and their administrators. An administrator, in turn, approves the creation of individual user accounts. The administrator can give some of those users administrative rights to, in turn, approve other users.

Our job is to ensure that only bona fide researchers can create and administer local authorities. If you wish to create an authority, please be sure to register with your institutional e-mail address, and please direct us to an institutional website or provide other evidence of your role. In general, we approve faculty members at institutions of higher education and senior researchers at research centers, but we will consider other cases as well.

A local authority administrator also approves the creation of *teams*, which group users. And an administrator manages, either directly or through a user to whom they delegate this role, any *nodes* that are contributed on behalf of the authority to the EdgeNet cluster.

If you believe that you may be eligible to act as the local administrator of an authority, the remainder of this tutorial guides you through the registration process.

If you would like to use EdgeNet but do not believe that you can act as a local administrator, we encourage you to identify someone at your institution who is already an administrator, or who would be willing to register as one.

### A note on terminology for PlanetLab users

For those of you familiar with PlanetLab, an authority is similar to a *site* and a local authority administrator is similar to a *PI*.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface, in conjunction with e-mail.

## What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert EdgeNet's central administrators, who will, if all is in order, approve your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as both the local administrator and a user of your authority.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Obtain a temporary access credential

An EdgeNet authority request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a public kubeconfig file that anyone can use for the prupose of creating authority requests.

This public kubeconfig file is available here: [http://edge-net.org/downloads/config/public.cfg](http://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved in your working directory on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation of an authority request and the use of the one-time code to confirm the request. Once the request goes through, you will be provided with another kubeconfig file that is specific to you and that will allow you to carry out adminstrative actions having to do with your authority, as well as to use EdgeNet as an ordinary user.

### Prepare a description of your authority

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the authority request object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future authority:
- the **authority name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet authority names
- the **full name** of the authority, which is a human-readable name
- the **short name** of the authority, which is also human-readable, and can be the same as the full name, or a shorter name, in case the full name is long
- the **URL** of the authority; this should be a web page from your institution that confirms your role as a bona fide researcher
- the **postal address** of the authority
- the **contact person** who is the responsible for this authority; this is the authority's first administrator, who is typically yourself; the information provided for this person consists of:
  - a **username** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/); note that usernames need only be distinct within an authority
  - a **first name** (human readable)
  - a **last name** (human readable)
  - an **e-mail address**, which should be an institutional e-mail address
  - a **phone number**, which should be in qutoation marks, start with the country code using the plus notation, and not contain any spaces or other formatting

In what follows, we will assume that this file is saved in your working directory on your system as ``./authorityrequest.yaml``.

Example:
```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: AuthorityRequest
metadata:
  name: lip6-lab
spec:
  fullname: Laboratoire LIP6-CNRS
  shortname: lip6
  url: https://www.lip6.fr/recherche/team_membres.php?acronyme=NPA
  address: 4 place Jussieu, boite 169, 75005 Paris, France
  contact:
    username: timurfriedman
    firstname: Timur
    lastname: Friedman
    email: timur.friedman@sorbonne-universite.fr
    phone: "+33123456789"
```

### Create your authority request

Using ``kubectl``, create a authority request object:

```
kubectl create -f ./authorityrequest.yaml --kubeconfig ./public.cfg
```

This will cause an e-mail containing a one-time code to be sent to the address that you specified.

### Authenticate your request using a one-time code

The e-mail that you receive will contain a ``kubectl`` command that you can copy and paste onto your command line, editing only the path for the public kubeconfig file on your local system, if needed.

In the example here, the one-time code is ``bsv10kgeyo7pmazwpr``:

```
kubectl patch emailverification bsv10kgeyo7pmazwpr -n registration --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public.cfg
```

After you have done this, the EdgeNet system sends a notification e-mail to EdgeNet's central administrators, informing them of your registration request.

### Wait for approval and receipt of your permanent access credential

At this point, the EdgeNet central administrators will, if needed, contact you, and, provided everything is in order, approve your registration request. Upon approval, you will receive two emails. The first one confirms that your registration is complete, while the second one contains your user information and user-specific kubeconfig file.

You can now start using EdgeNet, as both administrator of your local authority and as a regular user, with your user-specific kubeconfig file.
