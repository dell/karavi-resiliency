#!/bin/sh
# Copyright (c) 2021-2022 Dell Inc., or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This is a basic test that cycles through a set of listed nodes and kills them, making sure all the labeled apps continue to run after time for recovery.
NODELIST=""
APP_RECOVERY_TIME=540
NODE_DELAY_TIME=30
REBOOT=off
COUNT=0


check_running() {
	kubectl get pods -l podmon.dellemc.com/driver -A -o wide
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v Running | wc -l)
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
