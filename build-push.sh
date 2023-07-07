#!/bin/bash

TAG="latest"
HUBUSER="ubombar"

for name in admissioncontrol fedlet nodelabeler selectivedeploymentanchor tenant cluster fedscheduler notifier slice tenantrequest clusterlabeler managercache rolerequest sliceclaim tenantresourcequota clusterrolerequest nodecontribution selectivedeployment subnamespace vpnpeer
do 
    docker buildx build . -f build/images/selectivedeployment/Dockerfile --tag $HUBUSER/$name:$TAG
    docker push $HUBUSER/$name:$TAG
done 

