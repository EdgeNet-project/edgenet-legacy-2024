#!/bin/bash
DELIMETER='{{"\n"}}'
FIELD='Hostname'
TEMPLATE="{{range.items}}{{range.status.addresses}}{{if eq .type \"$FIELD\"}}{'name': '{{.address}}'},{{end}}{{end}}{{if and .status .status.conditions}}{{(.status.conditions 4).type}}{{end}}$DELIMETER{{end}}"
kubectl get nodes -o template --template="${TEMPLATE}"
