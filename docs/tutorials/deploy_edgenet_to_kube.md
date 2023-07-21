# Deploying EdgeNet to a Kubernetes Cluster

This documentation describes how to create a working EdgeNet cluster in your environment. You can refer to this tutorial for deploying EdgeNet software to a local, sandbox, or production cluster.

## Technologies you will use

To deploy EdgeNet to a Kubernetes cluster you need to have a cluster with [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured. 

There are many alternatives for creating a cluster for test purposes you can use [minikube](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover a cluster creation.

## What will you do

You will install cert-manager and clone the EdgeNet repository. Then you will install desired features.

<!-- EdgeNet extension for Kubernetes consists of two parts, the custom resource definitions ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) and custom controllers. 

CRDs (discussed [here](/docs/README.md#extending-kubernetes)) are custom objects that are manipulated, created, and destroyed by custom controllers.

You will be creating CRDs and deploying custom controllers to the cluster.

Addiditonally, you will use [git](https://git-scm.com/) for cloning the official EdgeNet repository.  -->

## 1. Before the EdgeNet's deployment install cert-manager
EdgeNet requires `cert-manager` to work. Please deploy [cert-manager](https://cert-manager.io/docs/installation/).

## 2. Clone EdgeNet repository
You need to clone the official EdgeNet repository to your local filesystem. Use the `cd` command to go to an empty directory you can use. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

<!-- After the cloning you may want to switch to the latest release. You can find the EdgeNet's releases [here](https://github.com/EdgeNet-project/edgenet/releases). To switch to a tag of a release you can use the command below.

```bash
git checkout tags/<tag-name>
``` -->

After the cloning you may want to switch to the latest release branch. You can find the EdgeNet's releases [here](https://github.com/EdgeNet-project/edgenet/releases). To switch to a branch of a release you can use the command below.

```bash
git checkout release-1.0
```

A handful of CRDs, controllers, and additional objects are required for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The files represents different feauture-pack of edgeNet. They are briefly explained below:

* `multi-tenancy.yaml` contains the CRDs, controllers, etc. for enabling single instance native multi-tenancy. Please refer to [multi-tenancy](/docs/custom_resources.md#multitenancy) documentation for more information.

* `multi-provider.yaml` contains the CRDs, controllers, etc. to cluster to have multi-provider functionality. Please refer to [multi-provider](/docs/custom_resources.md#multiprovider) section to have more information.
  
* `notifier.yaml` contains the notification manager. It is used for sending notifications about events in the cluster such as tenantrequests, rolerequests, clusterrolerequests, etc. These notifications are sent vie mail and/or slack.
  
* `location-based-node-selection.yaml` contains a set of features that allows deployments to be made using the geographical information of nodes. Please refer to the [location-based node selection](/docs/custom_resources.md#location-based-node-selection) section for additional information.
  
* `federation-manager.yaml` contains the CRDs, controllers, etc. for federation features intended for manager clusters. Note that, manager clusters should track caches, etc. which workload clusters doesn't have to thus it is redundant for workload clusters to have these definitions. Please refer to [federation](/docs/custom_resources.md#cluster-federation) section to have more information.
  
* `federation-workload.yaml` contains the CRDs, controllers, etc. for federation features intended for workload clusters.

* `all-in-one.yaml` contains the combined definitions for `multi-tenancy`, `multi-provider.yaml`, `location-based-node-selection`. Federation and notifier definitions are not included. We reccomend to install features seperately.

## 3. Installation
EdgeNet is designed portable thus if you only require certain features, it is possible to install EdgeNet without performing a full install. Below you can find different sets of features:
* Install only the [multi-tenancy features](#31-install-only-multi-tenancy)
* Install only the [multi-provider features](#32-install-only-multi-provider)
* Install only the [location-based-node-selection features](#33-install-only-location-based-node-selection)
* Install only the [notifier](#34-install-only-notifier)
* Install the [federation features](#35-install-federation)

### 3.1 Install only Multi-tenancy
The yaml file for multi-tenancy is located in `build/yamls/kubernetes/multi-tenancy.yaml`

Since it does not contain any configuration, you can directly apply and start using it's features. Run the following kubectl command to apply the yaml file.

```bash
kubectl apply -f build/yamls/kubernetes/multi-tenancy.yaml
```

Wait until the creation of custom containers and it is done. You can test the multi-tenancy by first [registering a tenant](/docs/tutorials/tenant_registration.md). 

Additionally, if you are on a test environment, you may want to remove the admission validation hook for testing multi-tenancy. However, **do not do this in a production environment**.

### 3.2 Install only Multi-provider
The yaml file for multi-provider is located in `build/yamls/kubernetes/multi-provider.yaml`

Unlike multi-tenancy, multi-provider features needs some configuration in order to work. You can edit the yaml file. Note that the api-keys, tokens, etc. of the external services that EdgeNet uses needs to be encoded in `base64`. You can find the command to encode the secrets.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `multi-provider.yaml` file can be configured:

```yaml
  # Used for DNS service not strictly required for EdgeNet to work
  namecheap.yaml: | 
    # Provide the namecheap credentials for DNS records.
    # app: "<App name>"
    # apiUser : "<API user>"
    # apiToken : "<API Token>"
    # username : "<Username>"

  # Used for node-labeler, if empty node-labeler cannot label nodes by their geoIPs
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"
```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```bash
kubectl -f apply ./build/yamls/kubernetes/multi-provider.yaml
```

Wait until the creation of custom containers and it is done. 
<!-- More info is required on how to use multi-provider features, maybe a tutorial -->

### 3.3 Install only Location-based-node-selection

### 3.4 Install only Notifier

### 3.5 Install Federation
> The federation features are actively worked on and are experimental. The federation features are built on top of multitenancy, thuse before installing make sure you installed the [multitenancy features](#31-install-only-multi-tenancy) to your Kubernetes cluster.





<!-- apiVersion: v1
kind: Secret
metadata:
  name: slack
  namespace: edgenet
type: Opaque
data:
  # token: auth token
  # channelid: channel ID 


### Install All Features of EdgeNet
Before your installation of EdgeNet, install [cert-manager](https://cert-manager.io/docs/installation/). EdgeNet uses `cert-manager` to manage certificates in your Kubernetes cluster.

> Note that since federation features needs to be installed to either managers or workload clusters, they are not included in the `all-in-one.yaml`



ou can change the current branch to install the desired version.

This file contains all of the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/), and other auxiliary definitions. 

If you want to use certain APIs that EdgeNet supports, before installing you need to configure the secrets. Do not forget to encode the secrets to base64 by using the `base64` command.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `all-in-one.yaml` file can be configured.

```yaml
  # Used for mailing service, if empty mailing service will not work
  smtp.yaml: | 
    # host: "<Hostname of the smtp server>"
    # port: "<Port of the smtp client>"
    # from: "<Mail address of the sender of notifications>"
    # username : "<Username of the account>"
    # password : "<Password of the account>"
    # to: "<Mail address of the administrator>"

  # Used for DNS service not strictly required for EdgeNet to work
  namecheap.yaml: | 
    # Provide the namecheap credentials for DNS records.
    # app: "<App name>"
    # apiUser : "<API user>"
    # apiToken : "<API Token>"
    # username : "<Username>"

  # Used for node-labeler, if empty node-labeler cannot label nodes by their geoIPs
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"
```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```bash
kubectl -f apply ./build/yamls/kubernetes/all-in-one.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.

---
### Install EdgeNet's *multi-tenancy* features
Before your installation of EdgeNet, install [cert-manager](https://cert-manager.io/docs/installation/). EdgeNet uses `cert-manager` to manage certificates in your Kubernetes cluster.

First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to an empty directory you can use. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects are required for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features of multitenancy are grouped is named `multi-tenancy.yaml` file.

ou can change the current branch to install the desired version.

This file contains all of the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), and other auxiliary definitions. 

In the multi-tenancy.yaml package EdgeNet does not use any external APIs, thus there is no need for a configuration. You can directly apply the yaml file with the following command:

```bash
kubectl -f apply ./build/yamls/kubernetes/multi-tenancy.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.

---
### Install EdgeNet's *multi-provider* features


---
### Install EdgeNet's *location-based-node-selection* features
Before your installation of EdgeNet, install [cert-manager](https://cert-manager.io/docs/installation/). EdgeNet uses `cert-manager` to manage certificates in your Kubernetes cluster.

First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to an empty directory you can use. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects are required for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features of location based node selection are grouped is named `location-based-node-selection.yaml` file.

ou can change the current branch to install the desired version.

This file contains all of the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/), and other auxiliary definitions. 

If you want to use certain APIs that EdgeNet supports, before installing you need to configure the secrets. Do not forget to encode the secrets to base64 by using the `base64` command.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `location-based-node-selection.yaml` file can be configured.

```yaml
  # Used for node-labeler, if empty node-labeler cannot label nodes by their geoIPs
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"
```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```bash
kubectl -f apply ./build/yamls/kubernetes/location-based-node-selection.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.

---
### Install EdgeNet's *federation* features
Before your installation of EdgeNet, install [cert-manager](https://cert-manager.io/docs/installation/). EdgeNet uses `cert-manager` to manage certificates in your Kubernetes cluster.

First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to an empty directory you can use. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

You can change the current branch to install the desired version.

A handful of CRDs, controllers, and additional objects are required for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. 

> The federation features are actively worked on and are experimental. The federation features are built on top of multitenancy, thuse before installing make sure you installed the [multitenancy features](#install-edgenets-multi-tenancy-features) to your Kubernetes cluster.

In the `federation.yaml` file, EdgeNet does not use any external APIs, thus there is no need for a configuration. You can directly apply the yaml file with the following command:

```bash
kubectl -f apply ./build/yamls/kubernetes/federation.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working. -->
