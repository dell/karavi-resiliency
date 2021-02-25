#!/bin
instances="1 2 3 4 5 6 7 8"
ndevices=0
nvolumes=4
storageClassName=vxflexos-notopo
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
port=5000

for i in $instances
do
	echo $i
	kubectl create namespace pmtv$k
	# following line for debug
        #helm install --dry-run --debug -n "pmtv$i" "pmtv$i" deploy --values deploy/values-vxflex.yaml --set podmonTest.namespace="pmtv$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.image="$image" 
        helm install -n "pmtv$i" "pmtv$i" deploy --values deploy/values-vxflex.yaml --set podmonTest.namespace="pmtv$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.ndevices=$ndevices --set podmonTest.nvolumes=$nvolumes --set podmonTest.image="$image" 
done
