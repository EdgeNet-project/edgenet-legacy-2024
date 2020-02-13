# Create a project in EdgeNet

In EdgeNet, a PI or manager can directly create a project that empowers participants to create slices. Participants, ie users, may belong to different sites.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to create a project.

## How to do?

You will use your EdgeNet kubeconfig file to create a project.

### Create a project
This object must include a project name consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), users that include username and site to which username belongs, and project description. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: Project
metadata:
  name: <your project name>
spec:
  users:
    - site: <site name>
      username: <username>
  description: <project description>
```

```
kubectl create -f ./project.yaml --kubeconfig ./your-kubeconfig.cfg
```

### Notification process

At this point, the PI(s) and manager(s) of the site on which project created and the participants of the project get their invitations by email containing project information.
