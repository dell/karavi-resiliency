#!/bin
instances="1 2 3 4"
for i in $instances
do
	echo $i
	kubectl create namespace pmt$i
	helm install -n pmt$i pmt$i deploy --set namespace=pmt$i
done
