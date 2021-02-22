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

instances="1 2 3 4"
storageClassName=vxflexos-notopo
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
port=5000

for i in $instances
do
	echo $i
	kubectl create namespace pmt$i
        helm install -n "pmt$i" "pmt$i" deploy --values deploy/values-vxflex.yaml --set podmonTest.namespace="pmt$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.image="$image"
done
