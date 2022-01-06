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
# The values below are default values. Many can be over-ridden by parameters.
REBOOT=off
COUNT=0
# NODELIST1... NODELIST10 are space separated lists of nodes that will be failed as a unit.
NODELIST1=""
NODELIST2=""
NODELIST3=""
NODELIST4=""
NODELIST5=""
NODELIST6=""
NODELIST7=""
NODELIST8=""
NODELIST9=""
NODELIST10=""

POLLTIME=5			# Poll time to print results
BOUNCEIPSECONDS=240             # Bounce IP time in seconds for interface down
TIMEOUT=600			# Maximum time in seconds to wait for a failure cycle (needs to be higher than EVACUATE_TIMEOUT)
MAXITERATIONS=3			# Maximum number of failover iterations
DRIVERNS="vxflexos"		# Driver namespace

rm -f stop			# Remove the stop file

for param in $*; do
  case $param in
  "--ns")
    shift
    DRIVERNS=$1
    shift
    ;;
  "--bounceipseconds")
    shift
    BOUNCEIPSECONDS=$1
    shift
    ;;
  "--maxiterations")
    shift
    MAXITERATIONS=$1
    shift
    ;;
  "--timeoutseconds")
    shift
    TIMEOUT=$1
    shift
    ;;
  "--help")
    shift
    echo "parameters: --ns driver-namespace [--bounceipseconds value] [--maxiterations value] [--timeoutseconds value]"
    exit
    ;;
  esac
done

[ "$DRIVERNS" = "" ] && echo "Required argument: --ns driver_namespace" && exit 2
echo "Collecting logs driver namespace $DRIVERNS podmon label $podmon_label timeout $TIMEOUT"
EVACUATE_TIMEOUT=$TIMEOUT       # Doesn't really matter if most of the time is spent in evacuation

# check_timeout takes an argument $1 for how many seconds we've been running
# and aborts if we exceed TIMEOUT
check_timeout() {
	if [ $1 -gt $TIMEOUT ]; then
		echo "******************* timed out: " $1 "seconds ********************"
		collect_logs.sh --ns $DRIVERNS
		exit 2
	fi
}


print_status() {
	notready=$(kubectl get nodes | awk '/NotReady/ { gsub("\\..*", "", $1); print $1; }';)
	taints=$(kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints |  awk '/.*podmon.*/ { gsub("\\..*", "", $1); print $1}')
	echo "NotReady: " $notready "taints: " $taints
	kubectl get pods -l podmon.dellemc.com/driver -A -o wide | awk  '
	BEGIN   {
		totalpods=0; totalrunning=0; totalcreating=0; totalerrors=0; totalcrashloopbackoff=0;
		nonode = "unscheduled"
	}
	/^NAMESPACE/ { next; }
	{
		ns=$1; podstate=$4; time=$6; node=$8; gsub("\\..*","",node); \
		nodes[node] = 1
		if (podstate !~ "Running") { print ns, podstate, time, node; }
		if (podstate == "Running") {
			runcounts[node]=runcounts[node]+1;
			totalrunning++;
		}
		if (podstate == "ContainerCreating") {
			creatingcounts[node]=creatingcounts[node]+1;
			totalcreating++;
		}
		if (podstate == "Error") {
			errorcounts[node]=errorcounts[node]+1;
			totalerrors++;
		}
		if (podstate == "CrashLoopBackOff") {
			crashloopbackoffcounts[node]=crashloopbackoffcounts[node]+1;
			totalcrashloopbackoff++;
		}
		if (podstate == "Pending") {
			pendingcounts[nonode]=pendingcounts[nonode]+1;
		}
		totalpods = totalpods+1;
	}
	END {
		pending = pendingcounts[nonode]
		if (pending == "") { pending=0 }
		print "Total Pods:", totalpods, "Running:", totalrunning, "Creating:", totalcreating, "Errors:", totalerrors, "CrashLoopBackoff:", totalcrashloopbackoff, "Pending (unscheduled):", pending
		for (node in nodes) {
			runners=runcounts[node];
			if (runners == "") { runners=0 }
			creators=creatingcounts[node];
			if (creators == "") { creators=0 }
			errors = errorcounts[node];
			if (errors == "") { errors=0 }
			crashloopbackoffs = crashloopbackoffcounts[node];
			if (crashloopbackoffs == "") { crashloopbackoffs=0 }
			printf "node %s running %s creating %d errors %d\n", node, runners, creators, errors;
		}
	}
	' | sort
}

# checks that all labeled pods are running, exits if not
check_running() {
	kubectl get pods -l podmon.dellemc.com/driver -A -o wide | sort -k 8
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v Running | wc -l)
	# account for header
	if [ $non_running_pods -gt 1 ]; then
		echo "some applications not running- terminating test"
		exit 2
	fi
        return 0
}

# Returns the number of labeled running pods
get_running_pods() {
	running=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep Running | wc -l)
	echo $running
}

# Returns number of labeled pods running on a list of nodes in $*
get_pods_on_nodes() {
	nodelist=$*
	totalcount=0
	for node in $nodelist
	do
		# Make sure to differentiate between node-1 and node-10 by checking for a blank after the node
		count=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep "$node " | wc -l)
		totalcount=$( expr $count + $totalcount)
	done
	echo $totalcount
}

failovercount=0
# Fails a node give a list of nodes
failnodes() {
	nodelist=$*
	# Fail all the nodes
	for node in $nodelist
	do
		# kill node
		echo bouncing $node $COUNT
		ssh $node nohup sh /root/bounce.ip --seconds $BOUNCEIPSECONDS &
	done
	failovercount=$(expr $failovercount + 1)

	if [ $REBOOT != "on" ]; then return; fi

	sleep 60
	for node in $nodelist
	do
		echo rebooting node $node
		ssh $node nohup sh reboot.node
	done
}

# Returns the number of tainted nodes
gettaints() {
	taints=$(kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints | grep 'storage.dell' | wc -l)
	echo $taints
}

# Given a nodelist, checks which nodes actually have pods running, and returns the nodes with pods only.
get_nodes_to_kill() {
	outputlist=""
	nodelist=$*
	for node in $nodelist
	do
		podCount=$(get_pods_on_nodes $node)
		if [ $podCount -gt 0 ]; then
			outputlist="$outputlist $node"
		fi
		mastCount=$(kubectl get nodes | grep $node | grep control-plane,mast | wc -l)
		if [ $mastCount -gt 0 ]; then
			outputlist="$outputlist $node"
		fi
	done
	echo $outputlist
}

# Does a failure of the nodes in $* and does an analysis of the times to recover
process_nodes() {
	NODELIST=$(get_nodes_to_kill $*)
	[ "$NODELIST" = "" ] && return
	initialRunningPods=$(get_running_pods)
	initialPodsToMove=$(get_pods_on_nodes $NODELIST)
	podsToMove=$initialPodsToMove
	if [ $podsToMove -gt 0 ]; then
		failnodes $NODELIST
		timesec=0
		# 
		while [ $podsToMove -gt 0 -a $timesec -lt $EVACUATE_TIMEOUT ];
		do
			echo $(date) $timesec $failovercount "podsToMove:" $podsToMove
			print_status
			sleep $POLLTIME
			timesec=$(expr $timesec + $POLLTIME)
			podsToMove=$(get_pods_on_nodes $NODELIST)
		done
		if [ $podsToMove -gt 0 ]; then
			echo "Evacuation phase timeout... collecting logs"
			collect_logs.sh --ns $DRIVERNS
		fi
		echo $(date) $timesec $failovercount "podsToMove:" $podsToMove
		movedPods=$(expr $initialPodsToMove - $podsToMove)
		echo "movedPods:" $movedPods " evacuation time seconds: " $timesec
		print_status

		runningPods=$(get_running_pods)
		while [ $runningPods -lt $initialRunningPods ]; 
		do
			echo $(date) $timesec $failovercount "runningPods: " $runningPods
			print_status
			sleep $POLLTIME
			timesec=$(expr $timesec + $POLLTIME)
			check_timeout $timesec
			runningPods=$(get_running_pods)
		done
		echo $(date) $timesec $failovercount "runningPods: " $runningPods
		echo "moving pods: " $initialPodsToMove "time for pod recovery seconds: " $timesec

		taints=$(gettaints)
		while [ $taints -gt 0 ];
		do
			echo $(date) $timesec $failovercount "tainted nodes:" $taints
			#kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints | grep "storage.dell"
			sleep $POLLTIME
			timesec=$(expr $timesec + $POLLTIME)
			check_timeout $timesec
			taints=$(gettaints)
		done
		kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints | grep "storage.dell"
		echo $(date) $timesec $failovercount "tainted nodes:" $taints
		echo "nodes cleanup time:" $(expr $timesec - $BOUNCEIPSECONDS)
	fi
	if [ $failovercount -ge $MAXITERATIONS ]; then
		exit 0
	fi
	sleep 60
	check_running
	
}

while true
do
	process_nodes $NODELIST1

	process_nodes $NODELIST2

	process_nodes $NODELIST3

	process_nodes $NODELIST4

	process_nodes $NODELIST5

	process_nodes $NODELIST6

	process_nodes $NODELIST7

	process_nodes $NODELIST8

	process_nodes $NODELIST9

	process_nodes $NODELIST10

	if [ -e stop ]; then echo "exiting due to stop file"; exit 0; fi
	
done
