# Make a site registration request in EdgeNet

In EdgeNet, a site holds the users, projects and slices, and **maybe** the nodes added into the EdgeNet cluster by that site. Therefore,
if you want to use EdgeNet as a user group with a hierarchy of authority responsibility,
you need to register your site for EdgeNet.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to sign your site up for EdgeNet.

## How to do?

You will use an EdgeNet public kubeconfig file to make your registration request.

### Create a request
In the first place, you need to create a site registration object according to your
information. This object must include site name consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), fullname, shortname, url, address, and contact who is the responsible for this site. It is required to provide username with allowed chars, firstname, lastname, email, and phone in the contact info. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: SiteRegistrationRequest
metadata:
  name: <your site name as a nickname, e.g. sorbonne-university>
spec:
  fullname: <your site name, e.g. Sorbonne University>
  shortname: <your site shortname, e.g. SU>
  url: <your site website, e.g. http://www.sorbonne-universite.fr/>
  address: <your site address, e.g. 21 rue de l’École de médecine 75006 Paris>
  contact:
    username: <your username>
    firstname: <your firstname>
    lastname: <your lastname>
    email: <your email address>
    phone: <your phone number, in format of "+XXXXXXXXXX">
```

```
kubectl create -f ./siteregistrationrequest.yaml --kubeconfig ./public-user.cfg
```

### Email verification

When you create a site registration request, EdgeNet automatically sends you an email that includes a kubectl command providing unique identifier to verify your email address. You can find the example below for verification.

```
kubectl patch emailverification bsv10kgeyo7pmazwpr -n site-edgenet --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public-user.cfg
```

The system sends notification emails to the EdgeNet admins about your registration request when the verification is done.

### Approval process

At this point, your request will be approved or denied by an admin. However, we assume that your request has been approved, in this case, you will receive two emails. The first one says your registration completed while the second one contains your user information and user-specific kubeconfig file. Then you can start using EdgeNet with that kubeconfig file.
