# Deploying EdgeNet to a Kubernetes Cluster

This documentation describes how to create a working EdgeNet cluster in your environment. You can refer to this tutorial for deploying EdgeNet software to a local, sandbox or a production cluster.

## Technologies you will use

To deploy EdgeNet to a Kubernetes cluster you need to have a cluster with [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured. 

There are many alternatives for creating a cluster for test purposes you can use [minikube](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover a cluster creation.

## What will you do

EdgeNet extension for Kubernetes consists of two parts, the custom resource definitions ([CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)) and custom controllers. 

CRDs (discussed [here](custom_resources.md)) are custom objects that are manipulated, created and destroyed by custom controllers.

You will be creating CRDs and deploy custom controllers to the cluster.

### Other yaml files
* `multi-tenancy.yaml` contains the feature pack for enabling multiple tenants. Please refer to [multi-tenancy](custom_resources.md#multitenancy) documentation for more information.
* `notifier.yaml` contains the notification such as mailer & slack notifier.
* `multi-provider.yaml` enables the cluster to have multi-provider functionality. For example, node contribution is contained in this file. Please refer to [multi-provider](custom_resources.md#multi-provider) to have more information.
* `location-based-node-selection.yaml` contains a set of features that allows deployments to be made using the node's geographical information. Please refer to the [location-based node selection](custom_resources.md#location-based-node-selection) section for additional information.

## Install the required CRDs and deploying controllers from `all-in-one.yaml`

A handful of CRDs, controllers and additional objects required are for EdgeNet to function. All of these declarations are organized in `build/yamls/kubernetes/all-in-one.yaml` file.

This file contains all of the CRDs, Deployments, [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) and other auxiliary definitions. Before applying this file it is important to configure the secrets.

Secrets in Kubernetes requires base 64 encoding. To achieve this you can use the following linux command to convert your secrets:

```
    echo "<token-or-secret>" | base64
```

The following secrets needs to be edited for features to work properly:

```yaml

  headnode.yaml: |
    # dns: "<Root domain>"
    # ip: "<IP address of the control plane node>"
  smtp.yaml: |
    # host: "<Hostname of the smtp server>"
    # port: "<Port of the smtp client>"
    # from: "<Mail address of the sender of notifications>"
    # username : "<Username of the account>"
    # password : "<Password of the account>"
    # to: "<Mail address of the administrator>"
  console.yaml: |
    # url: "<URL of the console>"
  namecheap.yaml: |
    # Provide the namecheap credentials for DNS records.
    # app: "<App name>"
    # apiUser : "<API user>"
    # apiToken : "<API Token>"
    # username : "<Username>"
  maxmind-account-id: "<MaxMind GeoIP2 precision API account id>"
  maxmind-license-key: "<MaxMind GeoIP2 precision API license key>"

```

After you edit the file, you can use the following command to apply, the CRDs, and the deployment of the custom controllers.

```
    kubectl -f apply ./build/yamls/kubernetes/all-in-one.yaml
```

This command creates all of the objects in Kubernetes including the deployments. Thus, it takes some time for them to start working.


