#!/bin/bash

# Copyright © 2020-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#      http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

#!/bin/sh
# Copyright (c) 2021-2025 Dell Inc., or its subsidiaries. All Rights Reserved.
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
BOUNCEKUBELETSECONDS=0          # Bounce the kubelet instead if BOUNCEIPSECONDS > 0
TIMEOUT=600			# Maximum time in seconds to wait for a failure cycle (needs to be higher than EVACUATE_TIMEOUT)
MAXITERATIONS=9999		# Maximum number of failover iterations
DRIVERNS="vxflexos"		# Driver namespace
REBALANCE=0			# Do rebalance if needed for pods with affinity
WORKLOADTYPE=${WORKLOADTYPE:-"pod"}
NODE_USER=${NODE_USER:-"root"}
PASSWORD=${PASSWORD:-""}

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
  "--bouncekubeletseconds")
    shift
    BOUNCEKUBELETSECONDS=$1
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
  "--rebalance")
    REBALANCE=1
    shift
    ;;
  "--workload-type")
    shift
    WORKLOADTYPE=$1
    shift
    ;;
  "--node-user")
    shift
    NODE_USER=$1
    shift
    ;;
  "--password")
    shift
    PASSWORD=$1
    shift
    ;;
  "--help")
    shift
    echo "parameters: --ns driver-namespace [--bounceipseconds value] [--bouncekubeletseconds value] [--maxiterations value] [--timeoutseconds value]"
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
		if [ $REBALANCE -gt 0 ]; then
			rebalance
		else
			echo "******************* timed out: " $1 "seconds ********************"
			../../tools/collect_logs.sh --ns $DRIVERNS
			python3 plot_scale_test.py || { echo "Python script failed"; exit 1; }
			echo "Plot saved as recovery_graph.png"
			exit 2
		fi
	fi
}

# getinitialpods queries each podmontest pod to determine the initial pod id that initialized the volume
# if this changes over time, it might indicate the volume was over-written and reinitialized
# the test stores the initial pod ids at the beginning in initial_pods.orig
# each iteration it gets the ids again and stores them in initial_pods.now and compares
getinitialpods() {
	pmts=$(kubectl get namespace | awk '/pmt/ { print $1; }')
	for i in $pmts; do echo -n "$i "; kubectl logs -n $i podmontest-0 | grep initial-pod | head -1; done
}


# ====================================================================================================================================================
# This part of the code rebalances pods across nodes for pod affinity.
maxPods=90

# nodelist returns a list of nodes(
nodelist() {
	kubectl get nodes -A | grep -v mast.r  | grep -v NAME | awk '{ print $1 }'
}

# get the number of pods on a node $1
podsOnNode() {
	# Add an extra space on match string to differentiate worker-1 from worker-10
	kubectl get pods -A -o wide | grep "$1 " | wc -l
}

# get namespaces of the pending pods 
getNSOfPendingPods() {
	 kubectl get pods -A -o wide | grep Pending | grep -v default | awk '{ print $1}'
}

# cordon a k8s node $1=node id
cordon() {
	echo "cordoning node $1"
	kubectl cordon $1
}

# cordon a k8s node $1=node id
uncordon() {
	echo "uncordoning node $1"
	kubectl uncordon $1
}

# delete pod names in namespace $1=namespace
deletePodsInNS() {
	pods=$(kubectl get pods -n $1 | grep -v NAME | awk '{print $1}')
	echo pods "$pods to be deleted"
	for pod in $pods; do
		echo "kubctl delete pod -n $1 $pod"
		kubectl delete pod --grace-period 0 -n $1 $pod
	done
}


rebalance() {
        echo $(date) "Rebalancing pods to nodes..."
	cordonedNodes=""
	nodes=$(nodelist)
	echo nodes: $nodes
	for n in $nodes; do
		pods=$(podsOnNode $n)
		echo node $n has $pods pods
		if [ $pods -gt $maxPods ]; then
			cordon $n
			cordonedNodes="$cordonedNodes $n"
		fi
	done
	echo cordonedNodes: $cordonedNodes
	namespaces=$(getNSOfPendingPods)
	for ns in $namespaces; do
		echo "deleting pods in namespace $ns"
		deletePodsInNS $ns
	done
	echo "waiting for pods to get moved"
	for i in 1 2 3 4 5 6 7 8 9 10; do
		kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAME | grep -v Running
		non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v Running | wc -l)
		# Sleep if non_running_pods gt 1 (accounting for HEADER)
		if [ $non_running_pods -gt 1 ]; then
			sleep 60
		fi
	done
	for n in $cordonedNodes; do
		uncordon $n
	done
	echo $(date) "Rebalancing complete"
}
# ====================================================================================================================================================


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

copyOverTestScriptsToNode() {
    local address="$1"
    local scriptsDir="../../test/sh"
    local remoteScriptDir

    if [ "$NODE_USER" == "root" ]; then
        remoteScriptDir="/root/karavi-resiliency-tests"
    elif [ "$NODE_USER" == "core" ]; then
        remoteScriptDir="/usr/tmp/karavi-resiliency-tests"
    else
        echo "Unsupported NODE_USER: $NODE_USER"
        return 1
    fi

    echo "Attempting to scp scripts from $scriptsDir to $address:$remoteScriptDir"

    if [ "$NODE_USER" == "core" ]; then
        # Create remote directory without password
        ssh -o StrictHostKeyChecking=no "$NODE_USER@$address" "date; rm -rf $remoteScriptDir; mkdir $remoteScriptDir"
        if [ $? -ne 0 ]; then
            echo "Failed to create remote directory"
            return 1
        fi

        # Copy specific files to remote directory without password
        for file in "bounce.ip" "reboot.node" "bounce.kubelet"; do
            scp -o StrictHostKeyChecking=no "$scriptsDir/$file" "$NODE_USER@$address:$remoteScriptDir/"
            if [ $? -ne 0 ]; then
                echo "Failed to copy $file"
                return 1
            fi
        done

        # Set execute permissions and list directory without password
        ssh -o StrictHostKeyChecking=no "$NODE_USER@$address" "chmod +x $remoteScriptDir/* ; ls -ltr $remoteScriptDir"
        if [ $? -ne 0 ]; then
            echo "Failed to set permissions or list directory"
            return 1
        fi
    else
        # Create remote directory with password
        sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$NODE_USER@$address" "date; rm -rf $remoteScriptDir; mkdir $remoteScriptDir"
        if [ $? -ne 0 ]; then
            echo "Failed to create remote directory"
            return 1
        fi

        # Copy specific files to remote directory with password
        for file in "bounce.ip" "reboot.node" "bounce.kubelet"; do
            sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no "$scriptsDir/$file" "$NODE_USER@$address:$remoteScriptDir/"
            if [ $? -ne 0 ]; then
                echo "Failed to copy $file"
                return 1
            fi
        done

        # Set execute permissions and list directory with password
        sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$NODE_USER@$address" "chmod +x $remoteScriptDir/* ; ls -ltr $remoteScriptDir"
        if [ $? -ne 0 ]; then
            echo "Failed to set permissions or list directory"
            return 1
        fi
    fi

    echo "Scripts successfully copied to $address:$remoteScriptDir"
    return 0
}

failovercount=0
# Fails a node give a list of nodes
failnodes() {
    nodelist=$*
    # Fail all the nodes
    for node in $nodelist
    do
        copyOverTestScriptsToNode "$node"

        # Kill node
        if [ $BOUNCEKUBELETSECONDS -gt 0 ]; then
            echo bouncing kubelet $node $COUNT
            if [ "$NODE_USER" == "core" ]; then
                ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /usr/tmp/karavi-resiliency-tests/bounce.kubelet --seconds $BOUNCEKUBELETSECONDS > /dev/null 2>&1 &"
            elif [ "$NODE_USER" == "root" ]; then
                sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /root/karavi-resiliency-tests/bounce.kubelet --seconds $BOUNCEKUBELETSECONDS > /dev/null 2>&1 &"
            else
                echo "Unsupported NODE_USER: $NODE_USER"
                return 1
            fi
        else
            echo bouncing ip $node $COUNT
            if [ "$NODE_USER" == "core" ]; then
                ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /usr/tmp/karavi-resiliency-tests/bounce.ip --seconds $BOUNCEIPSECONDS > /dev/null 2>&1 &"
            elif [ "$NODE_USER" == "root" ]; then
                sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /root/karavi-resiliency-tests/bounce.ip --seconds $BOUNCEIPSECONDS > /dev/null 2>&1 &"
            else
                echo "Unsupported NODE_USER: $NODE_USER"
                return 1
            fi
        fi
    done
    failovercount=$(expr $failovercount + 1)

    if [ $REBOOT != "on" ]; then return; fi

    sleep 60
    for node in $nodelist
    do
        echo rebooting node $node
        if [ "$NODE_USER" == "core" ]; then
            ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /usr/tmp/karavi-resiliency-tests/reboot.node > /dev/null 2>&1 &"
        elif [ "$NODE_USER" == "root" ]; then
            sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no "$NODE_USER@$node" "nohup sudo sh /root/karavi-resiliency-tests/reboot.node > /dev/null 2>&1 &"
        else
            echo "Unsupported NODE_USER: $NODE_USER"
            return 1
        fi
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
	echo "processing NODELIST:" $NODELIST
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
			../../tools/collect_logs.sh --ns $DRIVERNS
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
		
		# Log recovery data to CSV
		LOG_FILE="recovery_times.csv"
		if [ ! -f "$LOG_FILE" ]; then
			echo "num_instances,recovery_time_sec" > "$LOG_FILE"
		fi
		echo "$initialPodsToMove,$timesec" >> "$LOG_FILE"

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
		sleep 60
		check_running
		if [ "$WORKLOADTYPE" == "pod" ]; then
			getinitialpods > initial_pods.now
			echo "verifying initial_pods.now and initial_pods.orig to ensure there is no data loss"
			diff -b initial_pods.now initial_pods.orig
			rc=$?
		fi
		if [ $failovercount -ge $MAXITERATIONS ]; then
			echo $(date) "exiting due to failover count: " $failovercount
			python3 plot_scale_test.py || { echo "Python script failed"; exit 1; }
			echo "Plot saved as recovery_graph.png"
			exit 0
		fi
	fi
}
if [ "$WORKLOADTYPE" == "pod" ]; then
	getinitialpods > initial_pods.orig
fi
echo "falling into main loop..."
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
