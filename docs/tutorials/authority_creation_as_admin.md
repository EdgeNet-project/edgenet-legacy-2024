# Create a authority in EdgeNet

In EdgeNet, a authority holds the users, teams and slices, and **maybe** the nodes added into the EdgeNet cluster by that authority. Therefore,
if you want a user group with a hierarchy of authority responsibility to use EdgeNet, you need to create a authority.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to create a authority in EdgeNet.

## How to do?

You will use your EdgeNet admin kubeconfig file to create a authority object.

### Create a authority
In the first place, you need to create a authority object according to your
information. This object must include authority name consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), fullname, shortname, url, address, and contact who is the responsible for this authority. It is required to provide username with allowed chars, firstname, lastname, email, and phone in the contact info. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: Authority
metadata:
  name: <your authority name as a nickname, e.g. sorbonne-university>
spec:
  fullname: <your authority name, e.g. Sorbonne University>
  shortname: <your authority shortname, e.g. SU>
  url: <your authority webauthority, e.g. http://www.sorbonne-universite.fr/>
  address: <your authority address, e.g. 21 rue de l’École de médecine 75006 Paris>
  contact:
    username: <your username>
    firstname: <your firstname>
    lastname: <your lastname>
    email: <your email address>
    phone: <your phone number, in format of "+XXXXXXXXXX">
```

```
kubectl create -f ./authority.yaml --kubeconfig ./admin-user.cfg
```

### Notification process

When you create a authority in EdgeNet, the system automatically sends a notification email that says the authority creation completed to the authority contact defined. During this period, it also creates a user-specific kubeconfig file to be sent by email. The user can start using EdgeNet after receiving this kubeconfig file.
