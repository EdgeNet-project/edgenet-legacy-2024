#!/bin/bash
# kubectl exec multilevel-mda-lite-paristraceroute-6fxbd -- bash -c "python3 multilevel-mda-lite/MDALite.py 142.104.197.120"
# ./paris-traceroute.sh multilevel-mda-lite-paristraceroute-6fxbd 142.104.197.120
# -paristraceroute-6fxbd -- bash -c "python3 multilevel-mda-lite/MDALite.py 142.104.197.120"
kubectl exec multilevel-mda-lite-paristraceroute-$1 -- bash -c "python3 multilevel-mda-lite/MDALite.py $2"
