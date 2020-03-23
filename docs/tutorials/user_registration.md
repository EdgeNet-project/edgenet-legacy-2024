# Make a user registration request in EdgeNet

In EdgeNet, a user can have a variety of roles as PI, Manager, and User, and Tech role will be enabled in the future to manage node operations. However, anyone who wants to use EdgeNet can make registration request to a authority only to become a user.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to sign your user up for a authority in EdgeNet.

## How to do?

You will use an EdgeNet public kubeconfig file to make your registration request.

### Create a request
In the first place, you need to create a user registration object according to your
information. This object must include username consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), the namespace of the authority, which is a combination of **"authority"** prefix and authority nickname, you want yourself to register in, firstname, lastname, email, password, and roles. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: UserRegistrationRequest
metadata:
  name: <your username>
  namespace: <your authority name as a nickname with a authority prefix, e.g. authority-sorbonne-university>
spec:
  firstname: <your firstname>
  lastname: <your lastname>
  email: <your email address>
  password: <your password at least base64 encoded>
  roles: [User]
```

```
kubectl create -f ./userregistrationrequest.yaml --kubeconfig ./public-user.cfg
```

### Email verification

When you create a user registration request, EdgeNet automatically sends you an email that includes a kubectl command providing unique identifier to verify your email address. You can find the example below for verification.

```
kubectl patch emailverification bsv10kgeyo7pmazwpr -n <your authority name as a nickname with a authority prefix> --type='json' -p='[{"op": "replace", "path": "/spec/verified", "value": true}]' --kubeconfig ./public-user.cfg
```

The system sends notification emails to the PI(s) and manager(s) about your registration request when the verification is done.

### Approval process

At this point, your request will be approved or denied by the PI(s) or manager(s) of the authority. However, we assume that your request has been approved, in this case, you will receive two emails. The first one says your registration completed while the second one contains your user information and user-specific kubeconfig file. Then you can start using EdgeNet with that kubeconfig file.
