#!/bin/sh
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

# This is a basic test that cycles through a set of listed nodes and kills them, making sure all the labeled apps continue to run after time for recovery.
NODELIST="node1 node2 node3"
APP_NAMESPACES="pmt1 pmt2 pmt3 pmt4"
PODMON_LABEL=podmon.dellemc.com/driver=csi-vxflexos
APP_RECOVERY_TIME=300
NODE_DELAY_TIME=90
REBOOT=off
COUNT=0


check_running() {
	kubectl get pods -l podmon.dellemc.com/driver=csi-vxflexos -A -o wide
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver=csi-vxflexos -A -o wide | grep -v Running | wc -l)
	# account for header
	if [ $non_running_pods -gt 1 ]; then
		echo "some applications not running- terminating test"
		exit 2
	fi
        return 0
}
check_running

while true
do
	for node in $NODELIST
	do
		# kill node
		date
		COUNT=$( expr $COUNT + 1)
		echo bouncing $node $COUNT
		ssh $node nohup sh /root/bounce.ip &
		if [ $REBOOT = "on" ]
		then
			sleep 60
			ssh $node nohup sh reboot.node
		fi
		sleep $APP_RECOVERY_TIME
		date
		check_running
		sleep $NODE_DELAY_TIME
	done
done
