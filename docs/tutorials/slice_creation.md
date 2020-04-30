# Create a slice in EdgeNet

In EdgeNet, a PI or manager can directly create a slice that is a workspace to deploy applications towards the cluster. There are three slice profiles, which are Low, Medium, and High, that directly impacts on the Slice expiration date and resource quota on the namespace. Participants, ie users, may belong to different authorities.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to create a slice.

## How to do?

You will use your EdgeNet kubeconfig file to create a slice.

### Create a slice
This object must include a slice name consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), slice type that can be Classroom, Experiment, Testing, and Development, slice profile that can be Low, Medium, and High, users that include username and authority to which username belongs. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: Slice
metadata:
  name: <your slice name>
spec:
  type: <your slice type>
  profile: <your slice profile>
  users:
    - authority: <authority name>
      username: <username>
    - authority: <authority name>
      username: <username>
```

```
kubectl create -f ./slice.yaml --kubeconfig ./your-kubeconfig.cfg
```

### Notification process

At this point, the PI(s) and manager(s) of the authority on which slice created and the participants of the slice get their invitations by email containing slice information.
