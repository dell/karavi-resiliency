#!/bin/sh
alias k=kubectl
while true;
do
	date
	k get nodes -o wide
	k get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
	k get leases $vn
	k get pods $vn -o wide
	k get pods -l podmon.dellemc.com/driver=csi-vxflexos -A -o wide
	sleep 5
done
