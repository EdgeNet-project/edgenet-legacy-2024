# Registering an EdgeNet authority

Authorizations to use EdgeNet are handed out hierarchically, so that local administrators approve local users who they know.

The central administrators of EdgeNet approve the establishment of *authorities*, each authority having its own local authority administrator called a *PI* (principal investigator). A PI, in turn, approves the creation of individual user accounts. PIs also approve the creation of *teams*, which group users. And the PIs may be responsible for *nodes* that are contributed to the EdgeNet cluster.

This tutorial guides you, as a future PI, through the registration of your authority.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface, in conjunction with e-mail, to register a authority with EdgeNet.

## What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert the EdgeNet administrators, who will either approve or deny your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as both a PI and a user of your authority.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Obtain a temporary access credential

An EdgeNet authority request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a kubeconfig file that anyone can use for this.

This public kubeconfig file is available here: [https://edge-net.org/downloads/config/public.cfg](https://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation and, together with a one-time code, the modification of your own authority request. Upon successful termination of the authority creation process, you will be provided with another kubeconfig file that is specific to you and that will allow you to carry out PI and user actions having to do with your authority.

### Prepare a description of your authority

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the authority request object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future authority:
- the **authority name** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet authority names
- the **full name** of the authority, which is a human-readable name
- the **short name** of the authority, which is also human-readable, and can be the same as the full name, or a shorter name, in case the full name is long
- the **URL** of the authority
- the **postal address** of the authority
- the **contact person** who is the responsible for this authority; this is the authority's first PI, who is typically yourself; the information provided for this person consists of:
  - a **username** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet usernames, **across all authorities**
  - a **first name** (human readable)
  - a **last name** (human readable)
  - an **e-mail address**
  - a **phone number**, which should start with the country code using the plus notation, and without spaces or other formatting

In what follows, we will assume that this file is saved on your system as ``./authorityrequest.yaml``.

Example:
```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: AuthorityRequest
metadata:
  name: sorbonne-universite
spec:
  fullname: Sorbonne Université
  shortname: SU
  url: http://www.sorbonne-universite.fr/
  address: 21 rue de l’École de médecine, 75006 Paris, France
  contact:
    username: timurfriedman
    firstname: Timur
    lastname: Friedman
    email: timur.friedman@sorbonne-universite.fr
    phone: +33123456789
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

After you have done this, the EdgeNet system sends a notification e-mail to the EdgeNet administrators, informing them of your registration request.

### Wait for approval and receipt of your permanent access credential

At this point, your request will be approved or denied by an admin. Assuming that your request has been approved, you will receive two emails. The first one confirms that your registration is complete, while the second one contains your user information and user-specific kubeconfig file.

You can now start using EdgeNet with your user-specific kubeconfig file.
