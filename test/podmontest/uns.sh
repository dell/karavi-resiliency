#!/bin/sh
instances="v1"
for i in $instances
do
	helm delete -n pmt$i pmt$i
done
