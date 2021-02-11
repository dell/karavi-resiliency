#!/bin/sh
# Fails a list of nodes passed in as arguments.
# Assumes that bounce.ip is installed an appropriately configured on each node.

nodelist=$*
echo failing nodes: $nodelist
for node in $nodelist
do
	echo bouncing $node $COUNT
	ssh $node nohup sh /root/bounce.ip &
done
