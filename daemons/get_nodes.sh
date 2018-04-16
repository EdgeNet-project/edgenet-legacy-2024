#!/bin/bash
DELIMETER='{{"\n"}}'
FIELD='Hostname'
TEMPLATE="{{range.items}}{{range.status.addresses}}{{if eq .type \"$FIELD\"}}{{.address}}{{end}}{{end}}$DELIMETER{{end}}"
kubectl get nodes -o template --template="${TEMPLATE}"
