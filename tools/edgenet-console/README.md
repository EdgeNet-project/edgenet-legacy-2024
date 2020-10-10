<img src="https://raw.githubusercontent.com/EdgeNet-project/edgenet/master/assets/logos/edgenet_logos_2020_05_03/edgenet_logo_2020_05_03_w_text.svg" width="400">

## Setup Notes

### Dashboard webhook authentication
To setup the webhook auth api

- Create the file authn-config.yaml like so. 
"server" should point to the authentication endpoint configured in the routes
```
apiVersion: v1
kind: Config
clusters:
  - name: authn
    cluster:
      server: https://edgenet-test.planet-lab.eu/kubernetes/authenticate
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
$ kubectl describe pod kube-apiserver-XX -n kube-system

```

- Modify the Dashboard API server configuraton by adding the
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
    - --authentication-token-webhook-config-file=/etc/kubernetes/pki/authn-config.yaml
    - ...
[...]
```

## NOTES

### Create kubernetes config file
```
# https://stackoverflow.com/questions/47770676/how-to-create-a-kubectl-config-file-for-serviceaccount
# your server name goes here
server=https://localhost:6443
# the name of the secret containing the service account token goes here
name=default-token-zrp26

ca=$(kubectl get secret/$name -o jsonpath='{.data.ca\.crt}')
token=$(kubectl get secret/$name -o jsonpath='{.data.token}' | base64 --decode)
namespace=$(kubectl get secret/$name -o jsonpath='{.data.namespace}' | base64 --decode)

echo "
apiVersion: v1
kind: Config
clusters:
- name: default-cluster
  cluster:
    certificate-authority-data: ${ca}
    server: ${server}
contexts:
- name: default-context
  context:
    cluster: default-cluster
    namespace: default
    user: default-user
current-context: default-context
users:
- name: default-user
  user:
    token: ${token}
```

## OLD Console User Admin

Here is the full example with creating admin user and getting token:

Creating a admin / service account user called console

kubectl create serviceaccount console -n kube-system

Give the user admin privileges

kubectl create clusterrolebinding console --clusterrole=cluster-admin --serviceaccount=kube-system:console

Get the token

kubectl -n kube-system describe secret $(kubectl -n kube-system get secret | (grep console || echo "$_") | awk '{print $1}') | grep token: | awk '{print $2}'

