# Installing Fedmanctl Command Line Tool

This document describes how to install and use the command line tool fedmanctl designed for automating the federation procedure of workload and manager Kubernetes clusters.

## Technologies you will use

To federate Kubernetes clusters with `fedmanctl` you need to have [``kubectl``](https://kubernetes.io/docs/reference/kubectl/overview/) installed and configured.

Additionally, the `fedmanctl` installation uses the [``go language CLI tool``](https://go.dev/doc/install). You need to configure the `GOAPTH` environment variable to successfully finish the installation.

To test the federation capabilities you need to have access to at least 2 Kubernetes clusters. For test purposes, you can use [``minikube``](https://minikube.sigs.k8s.io/docs/) however this documentation will not cover cluster creation. Note that, the clusters would have access to each other otherwise the federation functionality of EdgeNet will not work.

## What will you do

In this tutorial, you will install `fedmanctl` CLI tool. The installation procedure is tested with Linux and MacOS operating systems only. For Windows users, the procedure might differ. 

## Installation

First, we will clone the repository to a local space. To have the repository you can use the `git clone` command.

```bash
git clone https://github.com/Edgenet-project/edgenet && cd edgenet
```

We will use the `go` CLI tool's functionalities for compiling and installing the `fedmanctl`. You can use the following command to compile and install the `fedmanctl` executable to the Go Bin path.

```bash
go install cmd/fedmanctl/fedmanctl.go 
```

After this, the `fedmanctl` is now installed under Go's binary path. If you haven't added this path to your `PATH` variable we recommend you do so. If you don't want to change the `PATH` variable you can use the following command to copy the executable into the `/usr/bin/` directory. Note that you need privileges to perform this operation. The following command moves the executable.

```bash
mv ${GOPATH}bin/fedmanctl /usr/bin 
```