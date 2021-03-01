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

instances="1 2"
ndevices=0
nvolumes=4
zone=""
storageClassName=unity-virt21048j9rzz-nfs
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
port=5000

for i in $instances
do
	echo $i
	kubectl create namespace pmtu$i
        helm install -n "pmtu$i" "pmtu$i" deploy --values deploy/values-unity-nfs.yaml --set podmonTest.namespace="pmtu$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.ndevices=$ndevices --set podmonTest.nvolumes=$nvolumes --set podmonTest.zone="$zone" --set podmonTest.image="$image" 
done
