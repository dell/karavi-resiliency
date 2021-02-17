#!/bin/sh
instances="1 2 3 4"
for i in $instances
do
	helm delete -n pmt$i pmt$i
done
