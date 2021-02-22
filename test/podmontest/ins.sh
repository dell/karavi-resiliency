#!/bin
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
