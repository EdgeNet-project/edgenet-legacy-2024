# Registering a user in an authority

This support page describes whether and how you can register a *user* in an authority with EdgeNet.
Your registration in an authority is subject to the approval of that authority's admin. However, anyone
who wants to use EdgeNet can make registration request in an authority only to become a user.

## Technologies you will use

You will use [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/), the [Kubernetes](https://kubernetes.io/) command-line interface, in conjunction with e-mail.

## What you will do

You will use a public kubeconfig file provided by EdgeNet to create a *registration request* object that is associated with your e-mail address. Object creation generates an e-mail to you, containing a one-time code. You will authenticate yourself by using that code to patch the object. This will alert the authority administrators, who will, if all is in order, approve your request. With approval, you receive via e-mail a kubeconfig file that is specific to you and that allows you to act as a user of the authority from which you make the request.

## Steps

### Make sure you have the Kubernetes command-line tool

If you do not already have ``kubectl``, you will need to install it on your system. Follow the [Kubernetes documentation](https://kubernetes.io/docs/tasks/tools/install-kubectl/) for this.

### Obtain a temporary access credential

An EdgeNet user registration request is a Kubernetes object, and to manipulate objects on a Kubernetes system you need a kubeconfig file. EdgeNet provides a public kubeconfig file that anyone can use for the purpose of creating user registration requests.

This public kubeconfig file is available here: [https://edge-net.org/downloads/config/public.cfg](https://edge-net.org/downloads/config/public.cfg). In what follows, we will assume that it is saved in your working directory on your system as ``./public.cfg``.

The public file does not allow any actions beyond the creation of a user registration request and the use of the one-time code to confirm the request. Once the request goes through, you will be provided with another kubeconfig file that is specific to you and that will allow you to use EdgeNet as an ordinary user.

### Prepare a description of your user

The [``.yaml`` format](https://kubernetes.io/docs/concepts/overview/working-with-objects/kubernetes-objects/) is used to describe Kubernetes objects. Create one for the user registration request object, following the model of the example shown below. Your ``.yaml``file must specify the following information regarding your future authority:
- the **username** that will be used by the EdgeNet system; it must follow [Kubernetes' rules for names](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/) and must be different from any existing EdgeNet authority names
- a **first name** (human readable)
- a **last name** (human readable)
- an **e-mail address**, which should be an institutional e-mail address

In what follows, we will assume that this file is saved in your working directory on your system as ``./userregistrationrequest.yaml``.

Example:
```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: UserRegistrationRequest
metadata:
  name: bsenel
  namespace: authority-edgenet
spec:
  firstname: Berat
  lastname: Senel
  email: berat.senel@lip6.fr
```

### Create your user registration request

Using ``kubectl``, create a user registration request object:

```
kubectl create -f ./userregistrationrequest.yaml --kubeconfig ./public.cfg
```

This will cause an e-mail containing a one-time code to be sent to the address that you specified.

### Authenticate your request using a one-time code

The e-mail that you receive will contain a ``kubectl`` command that you can copy and paste onto your command line, editing only the path for the public kubeconfig file on your local system, if needed.

In the example here, the one-time code is ``bsv10kgeyo7pmazwpr``:

```
kubectl patch emailverification bsv10kgeyo7pmazwpr -n registration --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public.cfg
```

After you have done this, the EdgeNet system sends a notification e-mail to the authority administrators, informing them of your registration request.

### Wait for approval and receipt of your permanent access credential

At this point, the authority administrators will, if needed, contact you, and, provided everything is in order, approve your registration request. Upon approval, you will receive an email that confirms that your registration is complete, and contains your user information and user-specific kubeconfig file.

You can now start using EdgeNet, as a regular user, with your user-specific kubeconfig file.
