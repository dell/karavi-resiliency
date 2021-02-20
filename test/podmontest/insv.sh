#!/bin
instances="1 2"
storageClassName=vxflexos-notopo
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
port=5000

for i in $instances
do
	echo $i
	kubectl create namespace pmtv$i
        helm install -n "pmtv$i" "pmtv$i" deploy --values deploy/values-vxflex.yaml --set podmonTest.namespace="pmtv$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.image="$image"
done
