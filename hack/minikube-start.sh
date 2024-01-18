#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

if command -v fedmanctl &>/dev/null; then
    echo "fedmanctl is installed. Creating clusters."

    # cluster_names=("cluster-worker-paris" "cluster-worker-munich" "cluster-worker-milan" "cluster-federator-eu")
    # cluster_nodes=(2 1 2 1)
    # cluster_federators=(0 0 0 1)

    cluster_names=("cluster-worker-paris")
    cluster_nodes=(1)
    cluster_federators=(0)

    # Create 3 worker 1 federator cluster, this also updates the KUBECONFIG file and adds different contexts.

    for ((i=0; i<${#cluster_names[@]}; i++)); do
        echo "> Creating ${cluster_names[i]} with ${cluster_nodes[i]} node(s)"
        minikube start --memory 2g --cpus 2 -n ${cluster_nodes[i]} --network bridge -p ${cluster_names[i]} &>/dev/null
    done

    # # Setup cert-manager and edgenet 
    echo "Clusters are created, deploying cert-manager and edgenet"
    for ((i=0; i<${#cluster_names[@]}; i++)); do
        echo "> Deploying to ${cluster_names[i]}"
        kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml --context ${cluster_names[i]}
    done

    for ((i=0; i<${#cluster_names[@]}; i++)); do
        # wait for cert-manager's webhook to run, there can be more sophisticated waiting mechanism
        sleep 25
        kubectl apply -f build/yamls/kubernetes/multi-tenancy.yaml --context ${cluster_names[i]}

        # For now delete this.
        kubectl delete validatingwebhookconfigurations.admissionregistration.k8s.io edgenet-admission-control --context ${cluster_names[i]}

        # If the cluster is a federator
        if [ "${cluster_federators[i]}" = "1" ]; then
            kubectl apply -f build/yamls/kubernetes/federation-manager.yaml --context ${cluster_names[i]}
        else
            kubectl apply -f build/yamls/kubernetes/federation-workload.yaml --context ${cluster_names[i]}
        fi
    done

    echo "DONE!"
else
    echo "fedmanctl is not installed or not in the system's PATH. Exitting..."
fi



