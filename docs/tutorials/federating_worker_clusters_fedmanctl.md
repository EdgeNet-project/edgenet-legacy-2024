# Federating a Workload Kubernetes Cluster with fedmanctl

This document describes how to federate a workload Kubernetes cluster using the command line tool fedmanctl designed for automating the federation procedure of workload and manager Kubernetes clusters.

## Technologies you will use

To federate Kubernetes clusters with `fedmanctl` you need to have [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured.

To test the federation capabilities you need to have access to at least 2 Kubernetes clusters. For test purposes, you can use [``minikube``](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover cluster creation. Note that, the clusters would have access to each other otherwise the federation functionality of EdgeNet will not work.

Lastly, the EdgeNet federation framework should be installed on the Kubernetes clusters that will be used. You can refer to the [EdgeNet's deployment tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) for help.

## What will you do

In this tutorial, you will federate a workload cluster. First, you will initialize the workload cluster. Then you will generate the token for the workload cluster. And lastly, you will use the token with the manager cluster to link manager and workload clusters.

## Initializing the Workload Cluster

> Before starting, it is important to have the latest version of the EdgeNet federation framework installed on the manager and the federation cluster. Please refer to the [EdgeNet's deployment tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) for more info.

You will use the fedmanctl's workload subcommand init to perform the initialization. This command creates the service account, role, role binding, and the secret that is required for the federation manager cluster to access from outside. You can easily initialize by the following command. Note that, you can specify the cluster and the user by the flag `--context` just like in `kubectl`.

```bash
fedmanctl workload init --context my-workload-cluster
```

If you receive an errors you may want to use the reset command to remove all the objects and try again. Use the following command to reset.

```bash
fedmanctl workload reset --context my-workload-cluster
```

## Generating the Token

> Note that at the moment the federation frameworks only accept an IP address for the clusters. Additionally, some of the configurations with `minikube` may have proxy addresses in the kubeconfig file. To generate the token of the workload cluster you need to use an accessible IP address by the manager cluster. We advise you to specify the ip and port info of the cluster's api-server by `--ip` and `--port` flags.

> At the moment the geographical labeling of the clusters can be configured while generating the token. You can use `--country` and `--city` tags to label the cluster.

To generate the token, you need to use the subcommand token. To only have the token without any additional message you can use the flag `--silent`. For debugging purposes, you can use the `--debug` flag which prints the unencoded version of the token so that the fields can be seen. Note that the token contains highly sensitive secret data. It is important to dispose of the token after use.
<!-- Maybe we can use one-time tokens? -->

```bash
fedmanctl workload token --city <CITY> --country <COUNTRY> --ip <IPADDRESS> --port <PORT> --context <WORKLOAD> --silent
```

## Linking the Workload Cluster with Manager Cluster

As the last step, it is really easy to finalize the work. You will use the federate subcommand for the manager along with the token. The following command performs the federation. Note that, along with the token, you need to supply the namespace which is where the secrets of the workload cluster will be held, the namespace is not created automatically by the fedmanctl.

```bash
fedmanctl manager federate <TOKEN> <NAMESPACE> --context <MANAGER>
```

You can also unlink the workload cluster by its cluster uid. That uid is the uid of the `kube-system` namespace of the workload cluster. With `fedmanctl` you can also list the linked clusters using the following command.

```bash 
fedmanctl manager list --context <MANAGER>
```

An example output is given;

```
CLUSTER NAME                                 CLUSTER NAMESPACE          VISIBILITY          ENABLED             STATE
cluster-c9236686-4188-40f5-9a38-2869f3a2a70b federation-newyork-cluster Public              true                Ready
cluster-02a0ba7c-c650-4e21-9786-68c59476136f federation-paris-cluster   Public              true                Ready
```

You can also separate a workload cluster using the cluster uid and the namespace with the following command.

```bash 
fedmanctl manager separate <CLUSTERUID> <NAMESPACE> --context <MANAGER>
```