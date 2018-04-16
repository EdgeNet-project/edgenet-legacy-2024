#!/bin/bash
kubectl get namespaces | grep -Ev 'default|kube-*|NAME' | awk '{ print $1 }'
