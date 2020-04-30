# Create a user in EdgeNet

In EdgeNet, a user can have a variety of roles as PI, Manager, and User, and Tech role will be enabled in the future to manage node operations. You can create a user in a authority if you are a PI of that authority or a cluster admin.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to register a user in a authority.

## How to do?

You will use your EdgeNet admin or PI kubeconfig file to create a user object.

### Create a user
In the first place, you need to create a user object according to your
information. This object must include username consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), the namespace of the authority, which is a combination of **"authority"** prefix and authority nickname, you want yourself to register in, firstname, lastname, email, password, and roles. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: User
metadata:
  name: <your username>
  namespace: <your authority name as a nickname with a authority prefix, e.g. authority-sorbonne-university>
spec:
  firstname: <your firstname>
  lastname: <your lastname>
  email: <your email address>
  password: <your password at least base64 encoded>
  roles: [PI, Manager, User]
```

```
kubectl create -f ./user.yaml --kubeconfig ./pi-user.cfg
```

### Notification process

When you create a user in EdgeNet, the system automatically sends a notification email that includes a user-specific kubeconfig file. The user can start using EdgeNet after receiving this kubeconfig file.
