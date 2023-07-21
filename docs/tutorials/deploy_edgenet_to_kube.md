# Deploying EdgeNet to a Kubernetes Cluster

This documentation describes how to create a working EdgeNet cluster in your environment. You can refer to this tutorial for deploying EdgeNet software to a local, sandbox, or production cluster.

## Technologies you will use

To deploy EdgeNet to a Kubernetes cluster you need to have a cluster with [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured. 

There are many alternatives for creating a cluster for test purposes you can use [minikube](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover a cluster creation.

## What will you do

You will install cert-manager and clone the EdgeNet repository. Then you will install desired features.

## 1. Before the EdgeNet's deployment install cert-manager
EdgeNet requires a `cert-manager` to work. Please deploy [cert-manager](https://cert-manager.io/docs/installation/).

## 2. Clone EdgeNet repository
You need to clone the official EdgeNet repository to your local filesystem. Use the `cd` command to go to an empty directory you can use. Then use the following command to clone and go inside the EdgeNet repository.

```bash
git clone https://github.com/EdgeNet-project/edgenet.git && cd ./edgenet
```

After the cloning, you may want to switch to the latest release branch. You can find EdgeNet's releases [here](https://github.com/EdgeNet-project/edgenet/releases). To switch to a branch of a release you can use the command below.

```bash
git checkout release-1.0
```

A handful of CRDs, controllers, and additional objects are required for EdgeNet to function. All of these declarations are organized in yaml files under `build/yamls/kubernetes/`. The files represent different feature-pack of edgeNet. They are briefly explained below:

* `multi-tenancy.yaml` contains the CRDs, controllers, etc. for enabling single instance native multi-tenancy. Please refer to [multi-tenancy](/docs/custom_resources.md#multitenancy) documentation for more information.

* `multi-provider.yaml` contains the CRDs, controllers, etc. to cluster to have multi-provider functionality. Please refer to the [multi-provider](/docs/custom_resources.md#multiprovider) section to have more information.
  
* `notifier.yaml` contains the notification manager. It is used for sending notifications about events in the cluster such as tenant requests, rolerequests, clusterrolerequests, etc. These notifications are sent via mail and/or Slack.
  
* `location-based-node-selection.yaml` contains a set of features that allows deployments to be made using the geographical information of nodes. Please refer to the [location-based node selection](/docs/custom_resources.md#location-based-node-selection) section for additional information.
  
* `federation-manager.yaml` contains the CRDs, controllers, etc. for federation features intended for manager clusters. Note that, manager clusters should track caches, etc. which workload clusters don't have to thus it is redundant for workload clusters to have these definitions. Please refer to [federation](/docs/custom_resources.md#cluster-federation) section to have more information.
  
* `federation-workload.yaml` contains the CRDs, controllers, etc. for federation features intended for workload clusters.

* `all-in-one.yaml` contains the combined definitions for `multi-tenancy`, `multi-provider.yaml`, `location-based-node-selection`. Federation and notifier definitions are not included. We recommend installing features separately.

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

Wait until the creation of custom controllers and it is done. You can test the multi-tenancy by first [registering a tenant](/docs/tutorials/tenant_registration.md). 

<!-- Additionally, if you are in a test environment, you may want to remove the admission validation hook for testing multi-tenancy. However, **do not do this in a production environment**. -->

### 3.2 Install only Multi-provider
The yaml file for multi-provider is located in `build/yamls/kubernetes/multi-provider.yaml`

Unlike multi-tenancy, multi-provider features need some configuration in order to work. You can edit the yaml file. Note that the API-keys, tokens, etc. of the external services that EdgeNet uses need to be encoded in `base64`. You can find the command to encode the secrets.

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

Wait until the creation of custom controllers and it is done. 
<!-- More info is required on how to use multi-provider features, maybe a tutorial -->

### 3.3 Install only Location-based-node-selection
The yaml file for location-based-node-selection is located in `build/yamls/kubernetes/location-based-node-selection.yaml`

Unlike multi-tenancy, location-based-node-selection features need some configuration in order to work. You can edit the yaml file. Note that the API-keys, tokens, etc. of the external services that EdgeNet uses need to be encoded in `base64`. You can find the command to encode the secrets.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `location-based-node-selection.yaml` file can be configured:

```yaml
  # Used for node-labeler, if empty node-labeler cannot label nodes by their geoIPs
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"
```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```bash
kubectl -f apply ./build/yamls/kubernetes/location-based-node-selection.yaml
```

Wait until the creation of custom controllers and it is done. 

### 3.4 Install only Notifier
The yaml file for the notifier is located in `build/yamls/kubernetes/notifier.yaml`

Unlike multi-tenancy, notifier features need some configuration in order to work. You can edit the yaml file. Note that the API-keys, tokens, etc. of the external services that EdgeNet uses need to be encoded in `base64`. You can find the command to encode the secrets.

Notifier can handle email, Slack, and console notifications. You need to create a Slack bot for Slack notifications and an email client for emails. Additionally, [EdgeNet Console](https://github.com/EdgeNet-project/console) is used with the EdgeNet testbed. You can leave the fields empty if you don't plan to use those features.

```bash
echo "<token-or-secret>" | base64
```

The following fields in the `notifier.yaml` file can be configured:

```yaml
  headnode.yaml: |
    # DNS should contain the root domain consisting of the domain name and top-level domain.
    # dns: "<Root domain>"
    # ip: "<IP address of the control plane node>"
  smtp.yaml: |
    # SMTP settings for mailer service. The 'to' field indicates the email address to receive the emails
    # that concerns the cluster administration.
    # host: ""
    # port: ""
    # from: ""
    # username : ""
    # password : ""
    # to: ""
  console.yaml: |
    # URL to the console if you deploy on your cluster. For example, https://console.edge-net.org.
    # url: "<URL of the console>"

# Below there is another secret object for slack
data:
  token: auth token
  channelid: channel ID
```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```bash
kubectl -f apply ./build/yamls/kubernetes/notifier.yaml
```

Wait until the creation of custom controllers and it is done. 

### 3.5 Install Federation
> The federation features are actively worked on and are experimental. The federation features are built on top of multitenancy, thuse before installing make sure you installed the [multitenancy features](#31-install-only-multi-tenancy) to your Kubernetes cluster.

You can have 2 types of clusters `manager cluster` and `workload cluster`. `Manager clusters` federate multiple `workload clusters`. They can send and receive workloads in the form of [selective deployments](/docs/custom_resources.md#selective-deployment).

The federation framework can be installed without any configurations. However, the `manager cluster` should have the `federation-manager.yaml` installed. As such the `workload cluster` should have the `federation-workload.yaml`.

You can use the below command to deploy the CRDs, custom controllers, etc. to the clusters.

```bash
kubectl -f apply ./build/yamls/kubernetes/federation-manager.yaml --context <MANAGER>
kubectl -f apply ./build/yamls/kubernetes/federation-workload.yaml --context <WORKLOAD>
```

Wait until the creation of custom controllers and it is done. 

> After installing the federation extensions to your manager and workload clusters, we recommend [installing fedmanctl](/docs/tutorials/fedmanctl_installation.md) for automated federation. Additionally, you can refer to [federation tutorial](/docs/tutorials/federating_worker_clusters_fedmanctl.md).
