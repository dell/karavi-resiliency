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

instances=${instances:-"1 2 3 4"}
ndevices=${ndevices:-0}
nvolumes=${nvolumes:-4}
zone=${zone:-""}
storageClassName=vxflexos-retain
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"

if [ "$DEBUG"x != "x" ]; then
  DEBUG="--dry-run --debug"
fi

for i in $instances; do
  echo $i
  kubectl create namespace pmtv$i
  helm install -n "pmtv$i" "pmtv$i" deploy \
              ${DEBUG} \
              --values deploy/values-vxflex.yaml \
              --set podmonTest.namespace="pmtv$i" \
              --set podmonTest.storageClassName="$storageClassName" \
              --set podmonTest.ndevices=$ndevices \
              --set podmonTest.nvolumes=$nvolumes \
              --set podmonTest.image="$image" \
              --set podmonTest.zone="$zone"
done
