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
storageClassName=unity-virt21048j9rzz-nfs
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"

if [ "$DEBUG"x != "x" ]; then
  DEBUG="--dry-run --debug"
fi

for i in $instances; do
	echo $i
	kubectl create namespace pmtu$i
  helm install -n "pmtu$i" "pmtu$i" deploy \
              ${DEBUG} \
              --values deploy/values-unity-nfs.yaml \
              --set podmonTest.namespace="pmtu$i"  \
              --set podmonTest.storageClassName="$storageClassName" \
              --set podmonTest.ndevices=$ndevices \
              --set podmonTest.nvolumes=$nvolumes \
              --set podmonTest.image="$image" \
              --set podmonTest.zone="$zone"
done
