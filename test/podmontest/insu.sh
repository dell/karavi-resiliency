#!/bin
instances="1 2 3 4"
ndevices=0
nvolumes=4
storageClassName=unity-virt21048j9rzz-nfs
image="$REGISTRY_HOST:$REGISTRY_PORT/podmontest:v0.0.54"
port=5000

for i in $instances
do
	echo $i
	kubectl create namespace pmtu$i
        helm install -n "pmtu$i" "pmtu$i" deploy --values deploy/values-unity-nfs.yaml --set podmonTest.namespace="pmtu$i"  --set podmonTest.storageClassName="$storageClassName" --set podmonTest.ndevices=$ndevices --set podmonTest.nvolumes=$nvolumes --set podmonTest.image="$image" 
done
