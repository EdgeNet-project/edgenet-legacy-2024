# Federating a Worker Kubernetes Cluster with fedmanctl

This document describes how to federate a worker Kubernetes cluster using the command line tool fedmanctl designed for automating the federation procedure of worker and manager Kubernetes clusters.

## Technologies you will use

To federate Kubernetes clusters with `fedmanctl` you need to have [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured.

To test the federation capabilities you need to have access to at least 2 Kubernetes clusters. For test purposes, you can use [``minikube``](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover cluster creation. Note that, the clusters would have access to each other otherwise the federation functionality of EdgeNet will not work.

Lastly, the EdgeNet federation framework should be installed on the Kubernetes clusters that will be used. You can refer to the [EdgeNet's deployment tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) for help.

## What will you do

In this tutorial, you will federate a worker cluster. First, you will initialize the worker cluster. Then you will generate the token for the worker cluster. And lastly, you will use the token with the manager cluster to link manager and worker clusters.

## Initializing the Worker Cluster

> Before starting, it is important to have the latest version of the EdgeNet federation framework installed on the manager and the federation cluster. Please refer to the [EdgeNet's deployment tutorial](/docs/tutorials/deploy_edgenet_to_kube.md) for more info.

You will use the fedmanctl's worker subcommand init to perform the initialization. This command creates the service account, role, role binding, and the secret that is required for the federation manager cluster to access from outside. You can easily initialize by the following command. Note that, you can specify the cluster and the user by the flag `--context` just like in `kubectl`.

```bash
fedmanctl worker init --context my-worker-cluster
```

If you receive an errors you may want to use the reset command to remove all the objects and try again. Use the following command to reset.

```bash
fedmanctl worker reset --context my-worker-cluster
```

## Generating the Token

> Note that at the moment the federation frameworks only accept an IP address for the clusters. Additionally, some of the configurations with `minikube` may have proxy addresses in the kubeconfig file. To generate the token of the worker cluster you need to use an accessible IP address by the manager cluster. We advise you to specify the ip and port info of the cluster's api-server by `--ip` and `--port` flags.

> At the moment the geographical labeling of the clusters can be configured while generating the token. You can use `--country` and `--city` tags to label the cluster.

To generate the token, you need to use the subcommand token. To only have the token without any additional message you can use the flag `--silent`. For debugging purposes, you can use the `--debug` flag which prints the unencoded version of the token so that the fields can be seen. Note that the token contains highly sensitive secret data. It is important to dispose of the token after use.
<!-- Maybe we can use one-time tokens? -->

```bash
fedmanctl worker token --city Paris --country France --ip IP-address-of-cluster --port port-of-cluster --context my-worker-cluster --silent
```

## Linking the Worker Cluster with Manager Cluster

As the last step, it is really easy to finalize the work. You will use the link subcommand for the manager along with the token. The following command performs the linking.

```bash
fedmanctl manager link generated-worker-token --context my-manager-custer
```

You can also unlink the worker cluster by its cluster uid. That uid is the uid of the `kube-system` namespace of the worker cluster. With `fedmanctl` you can also list the linked clusters using the following command.

```bash 
fedmanctl manager list --context my-manager-custer
```

Then you can unlink using the cluster uid with the following command.

```bash 
fedmanctl manager unlink cluster-uid-of-worker-cluster --context my-manager-custer
```