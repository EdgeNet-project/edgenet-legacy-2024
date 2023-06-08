# Deploying EdgeNet to a Kubernetes Cluster

This documentation describes how to create a working EdgeNet cluster in your environment. You can refer to this tutorial for deploying EdgeNet software to a local, sandbox, or production cluster.

## Technologies you will use

To deploy EdgeNet to a Kubernetes cluster you need to have a cluster with [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured. 

There are many alternatives for creating a cluster for test purposes you can use [minikube](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover a cluster creation.

## What will you do

EdgeNet extension for Kubernetes consists of two parts, the custom resource definitions ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) and custom controllers. 

CRDs (discussed [here](/docs/README.md#extending-kubernetes)) are custom objects that are manipulated, created, and destroyed by custom controllers.

You will be creating CRDs and deploying custom controllers to the cluster.

Addiditonally, you will use [git](https://git-scm.com/) for cloning the official EdgeNet repository. 

### YAML Files
* `multi-tenancy.yaml` contains the feature pack for enabling multiple tenants. Please refer to [multi-tenancy](/docs/custom_resources.md#multitenancy) documentation for more information.
* `multi-provider.yaml` enables the cluster to have multi-provider functionality. For example, node contribution is contained in this file. Please refer to [multi-provider](/docs/custom_resources.md#multiprovider) section to have more information.
* `notifier.yaml` contains the notification such as mailer & slack notifier resources.
* `location-based-node-selection.yaml` contains a set of features that allows deployments to be made using the node's geographical information. Please refer to the [location-based node selection](/docs/custom_resources.md#location-based-node-selection) section for additional information.
* `federation.yaml` contains custom resource definitions and custom controller deployments for federation features. Please refer to [federation](/docs/custom_resources.md#cluster-federation) section to have more information.

---
## Installation
EdgeNet is designed portable thus if you only require certain features, it is possible to install EdgeNet performing a full install. Below you can find different sets of features:
* Install [all of the EdgeNet's features](#install-all-features-of-edgenet)
* Install only the [multi-tenancy features](#install-edgenets-multi-tenancy-features)
* Install only the [multi-provider features](#install-edgenets-multi-provider-features)
* Install only the [location-based-node-selection features](#install-edgenets-location-based-node-selection-features)
* Install only the [federation features](#install-edgenets-federation-features)

---
### Install All Features of EdgeNet
First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to a directory. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects required are for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features are grouped is named `all-in-one.yaml` file.

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
First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to a directory. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects required are for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features of multitenancy are grouped is named `multi-tenancy.yaml` file.

This file contains all of the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), and other auxiliary definitions. 

In the multi-tenancy.yaml package EdgeNet does not use any external APIs, thus there is no need for a configuration. You can directly apply the yaml file with the following command:

```bash
kubectl -f apply ./build/yamls/kubernetes/multi-tenancy.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.

---
### Install EdgeNet's *multi-provider* features
First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to a directory. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects required are for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features are grouped is named `multi-provider.yaml` file.

This file contains all of the [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/), [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/), and other auxiliary definitions. 

If you want to use certain APIs that EdgeNet supports, before installing you need to configure the secrets. Do not forget to encode the secrets to base64 by using the `base64` command.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `multi-provider.yaml` file can be configured.

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

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.

---
### Install EdgeNet's *location-based-node-selection* features
First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to a directory. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects required are for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The file in which all of the features of location based node selection are grouped is named `location-based-node-selection.yaml` file.

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
First, you need to clone the official EdgeNet repository to your local file. Use the `cd` command to go to a directory. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

A handful of CRDs, controllers, and additional objects required are for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. 

> The federation features are actively worked on and are experimental. The federation features are built on top of multitenancy, thuse before installing make sure you installed the [multitenancy features](#install-edgenets-multi-tenancy-features) to your Kubernetes cluster.

In the `federation.yaml` file, EdgeNet does not use any external APIs, thus there is no need for a configuration. You can directly apply the yaml file with the following command:

```bash
kubectl -f apply ./build/yamls/kubernetes/federation.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for all of the features to start working.