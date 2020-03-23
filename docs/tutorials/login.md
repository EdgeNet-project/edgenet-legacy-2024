# Get web token to login in EdgeNet

In EdgeNet, a user create a login object to get a web token to log in to the portal or dashboard. When an object is created, the controller checks the credentials and, if there is a match, creates a temporary service account dedicated to serving that user. As a final step, the controller generates a kubeconfig file based on that temporary service account and send it to the user via email. Unless the expiration date is extended, a login object expires after an hour.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to create a login object.

## How to do?

You will use your EdgeNet kubeconfig file to create a login object.

### Get a web token
This object must include a username consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), the namespace of the authority, which is a combination of **"authority"** prefix and authority nickname, you belong to, and your password. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: Login
metadata:
  name: <your username>
  namespace: <your authority name as a nickname with a authority prefix, e.g. authority-sorbonne-university>
spec:
  password: <your password at least base64 encoded>
```

```
kubectl create -f ./login.yaml --kubeconfig ./your-kubeconfig.cfg
```

### Notification process

At this point, the user receives an email with the web token information.
