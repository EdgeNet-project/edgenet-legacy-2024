# Create a team in EdgeNet

In EdgeNet, a PI or manager can directly create a team that empowers participants to create slices. Participants, ie users, may belong to different authorities.

## Technologies you will use
The technology that you will use is [Kubernetes](https://kubernetes.io/), to create
and manipulate objects in EdgeNet. Furthermore, you will use [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/), which is the Kubernetes command-line interface
tool, to create a team.

## How to do?

You will use your EdgeNet kubeconfig file to create a team.

### Create a team
This object must include a team name consisting of [allowed characters](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/), users that include username and authority to which username belongs, and team description. Here is an example:

```yaml
apiVersion: apps.edgenet.io/v1alpha
kind: Team
metadata:
  name: <your team name>
spec:
  users:
    - authority: <authority name>
      username: <username>
  description: <team description>
```

```
kubectl create -f ./team.yaml --kubeconfig ./your-kubeconfig.cfg
```

### Notification process

At this point, the PI(s) and manager(s) of the authority on which team created and the participants of the team get their invitations by email containing team information.
