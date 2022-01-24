#!/bin
#
# Copyright (c) 2021. Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
#

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
instances=${instances:-4}
ndevices=${ndevices:-0}
nvolumes=${nvolumes:-4}
zone=${zone:-""}
storageClassName=${storageClassName:-unity-nfs}
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
prefix="pmtu"
replicas=1
deploymentType="statefulset"
driverLabel="csi-unity"
podAffinity="false"

if [ "$DEBUG"x != "x" ]; then
  DEBUG="--dry-run --debug"
fi

for param in $*
do
    case $param in
       "--instances")
          shift
          instances=$1
          shift
          ;;
       "--ndevices")
          shift
          ndevices=$1
          shift
          ;;
       "--nvolumes")
          shift
          nvolumes=$1
          shift
          ;;
       "--prefix")
          shift
          prefix=$1
          shift
          ;;
       "--storage-class")
          shift
          storageClassName=$1
          shift
          ;;
       "--replicas")
          shift
          replicas=$1
          shift
          ;;
       "--podAffinity")
          podAffinity="true"
          shift
          ;;
       "--deployment")
          deploymentType="deployment"
          shift
          ;;
       "--label")
          shift
          driverLabel=$1
          shift
          ;;
    esac
done

cd "$SCRIPTDIR"

i=1
while [ $i -le $instances ]; do
	echo $i
	kubectl create namespace ${prefix}$i
  helm install -n "${prefix}$i" "${prefix}$i" "${SCRIPTDIR}"/deploy \
              ${DEBUG} \
              --values deploy/values-unity-nfs.yaml \
              --set podmonTest.namespace="${prefix}$i"  \
              --set podmonTest.storageClassName="$storageClassName" \
              --set podmonTest.ndevices=$ndevices \
              --set podmonTest.nvolumes=$nvolumes \
              --set podmonTest.deploymentType=$deploymentType \
              --set podmonTest.replicas=$replicas \
              --set podmonTest.podAffinity=$podAffinity \
              --set podmonTest.image="$image" \
              --set podmonTest.zone="$zone" \
              --set podmonTest.driverLabel="$driverLabel"
  i=$((i + 1))
done
