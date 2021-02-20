#!/bin/sh
# This is a basic test that cycles through a set of listed nodes and kills them, making sure all the labeled apps continue to run after time for recovery.
NODELIST="lglw2213 lglw2215"
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
