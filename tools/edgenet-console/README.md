<img src="https://raw.githubusercontent.com/EdgeNet-project/edgenet/master/assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text.svg" width="400">

## Setup Notes

### Kubernetes webhook authentication
To setup the webhook auth api

- Create the file authn-config.yaml like so. 
"server" should point to the authentication endpoint configured in the routes
```
apiVersion: v1
kind: Config
clusters:
  - name: authn
    cluster:
      server: https://edgenet-test.planet-lab.eu/k8s/authentication
      insecure-skip-tls-verify: true
users:
  - name: kube-apiserver
contexts:
- context:
    cluster: authn
    user: kube-apiserver
  name: webhook
current-context: webhook
```

- Copy this file in a dir mounted by the K8s API container

```
# you can retreive info on the api pod
$ kubectl describe pod  kube-apiserver-XX -n kube-system

```

- Modify the Kubernetes API server configuraton by adding the
--authentication-token-webhook-config-file command option

```
# vi /etc/kubernetes/manifests/kube-apiserver.yaml

apiVersion: v1
kind: Pod
metadata:
  annotations:
    kubeadm.kubernetes.io/kube-apiserver.advertise-address.endpoint: 192.168.10.8:6443
  creationTimestamp: null
  labels:
    component: kube-apiserver
    tier: control-plane
  name: kube-apiserver
  namespace: kube-system
spec:
  containers:
  - command:
    - kube-apiserver
    - ...
    - --authentication-token-webhook-config-file=/etc/authn-config.yaml
    - ...
[...]
```



## Console User Admin

Here is the full example with creating admin user and getting token:

Creating a admin / service account user called k8sadmin

sudo kubectl create serviceaccount console -n kube-system

Give the user admin privileges

sudo kubectl create clusterrolebinding console --clusterrole=cluster-admin --serviceaccount=kube-system:k8sadmin

Get the token

sudo kubectl -n kube-system describe secret $(sudo kubectl -n kube-system get secret | (grep console || echo "$_") | awk '{print $1}') | grep token: | awk '{print $2}'

