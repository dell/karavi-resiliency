#!/bin/sh

# Important:
# Array1 is assumed to have the source RG
Array1Name=vxflexos217
Array1ID=205f819048a7000f
Array1IP=10.247.102.217
JumphostIP=10.247.102.218
Array2Name=vxflexos206
Array2ID=1102ecb40dadf70f
Array2IP=10.247.39.206
ClusterNodes="10.247.102.211 10.247.102.213 10.247.102.215"

REPLICATIONGROUPNAME=rg-17391a43-cf8b-4af5-bfd9-9eb96785303f

getSourceRG() {
	kubectl get rg | grep "^$REPLICATIONGROUPNAME"
}

getSourceRGLinkState() {
	kubectl get rg | awk "/^$REPLICATIONGROUPNAME/ { print \$5; }"
}

getTargetRG() {
	kubectl get rg | grep "^replicated-$REPLICATIONGROUPNAME"
}

getTargetRGLinkState() {
	kubectl get rg | awk "/^replicated-$REPLICATIONGROUPNAME/ { print \$5; }"
}


APPNAMESPACES=`kubectl get namespaces | awk '/pmtv/ { printf "%s ",$1; }'`
echo "Applicaion namespaces: $APPNAMESPACES"

# numberPodsRunning returns the number of pods runnint
numberPodsRunning() {
        running=0
        for ns in $APPNAMESPACES; do
                nsRunning=$(kubectl get pods -n $ns | grep -v NAME | grep Running | wc -l )
                running=$(expr  $running + $nsRunning)
        done
	echo $running
        return $running
}

echo "number running $(numberPodsRunning) "

# $1 is the number of pods that need to be running
waitOnNPodsRunning() {
        echo "Waiting on $1 pods to be running"
        nsRunning=0
        while [ $1 -ne $nsRunning ]; do
                nsRunning=$(numberPodsRunning)
                echo -n "Running $nsRunning waiting for $1 "
		date
                sleep 5
        done
}

# allPodsRunning returns the number of pods not running
allPodsRunning() {
	notRunning=0
	running=0
	for ns in $APPNAMESPACES; do
		nsNotRunning=$(kubectl get pods -n $ns | grep -v NAME | grep -v Running | wc -l )
		nsRunning=$(kubectl get pods -n $ns | grep -v NAME | grep Running | wc -l )
		notRunning=$(expr  $notRunning + $nsNotRunning)
		running=$(expr  $running + $nsRunning)
	done
	echo "pods running $running not-running $notRunning"
	return $notRunning
}

# waitOnAllPodsRunning waits until all pods are rnning
waitonAllPodsRunning() {
	echo "waiting on all pods to be running..."
	count=9999
	while [ $count -gt 0 ]
	do
		allPodsRunning
		count=$?
	done
	sleep 1
}

waitonAllPodsRunning

killArray1() {
	echo "Killing array 1"
	echo ssh "$JumphostIP" "ssh $Array1IP /bin/sh drop $ClusterNodes"
	ssh "$JumphostIP" "ssh $Array1IP /bin/sh drop $ClusterNodes"
}

killArray2() {
	echo "Killing array 2"
	echo ssh "$JumphostIP" "ssh $Array2IP /bin/sh drop $ClusterNodes"
	ssh "$JumphostIP" "ssh $Array2IP /bin/sh drop $ClusterNodes"
}

restoreArray1() {
	echo "Restoring array 1"
	echo ssh "$JumphostIP" "ssh $Array1IP /bin/sh undrop $ClusterNodes"
	ssh "$JumphostIP" "ssh $Array1IP /bin/sh undrop $ClusterNodes"
}

restoreArray2() {
	echo "Restoring array 2"
	echo ssh "$JumphostIP" "ssh $Array2IP /bin/sh undrop $ClusterNodes"
	ssh "$JumphostIP" "ssh $Array2IP /bin/sh undrop $ClusterNodes"
}


echo srcRG: $(getSourceRG)
echo targetRG: $(getTargetRG)
# getPVCNamePrefixes will return "src", "replicated", or "error" dending on if the PVC name prefixes are the same or different
# $1=namespace

getPVCNamePrefixes() {
if [ "$1" == "" ]; then
	echo "error - namespace is required"
	return
fi
kubectl get pvc -n $1 | awk '							\
BEGIN	{nsrc = 0; ntarget = 0 }					\
/[a-z].*/       {
        if (match($3, "^replicated-.")) {
                ntarget = narget+1
        } else {
                nsrc = nsrc+1
        }
}
END     {
                if (ntarget > 0 && nsrc == 0) { print "target" }
                if (nsrc > 0 && ntarget == 0) { print "src" }
                if (nsrc > 0 && ntarget > 0) { print "error" }
        }
'
}

# returns a string with the cluster failover mode, "src", "target", or "error"
getClusterMode() {
mode="unknown"
for ns in $APPNAMESPACES; do
	modex=`getPVCNamePrefixes $ns`
	#echo namespace $ns modex $modex
	if [ "$mode" == "unknown" ]; then
		mode="$modex"
	else
		if [ "$mode" != "$modex" ]; then
			mode="error"
		fi
	fi
done
echo $mode
}

# isReadyForFailover determines whether we can failover src or target
# returns 0 if ready, non zero if not ready
isReadyForFailover() { 
	# Get the current cluster mode, i.e. using the source volumes, or using the target volumes.
	export clusterMode=$(getClusterMode)
	if [ "$clusterMode" == "error" ]; then "echo cluster mode error - cannot failover"; return 2; fi
	# Get the link state of each replication group
	srcLinkState=$(getSourceRGLinkState)
	if [ "$srcLinkState" != "SYNCHRONIZED" ]; then echo "bad source LinkState $srcLinkState"; return 2;  fi
	targetLinkState=$(getTargetRGLinkState)
	if [ "$targetLinkState" != "SYNCHRONIZED" ]; then echo "bad target LinkState $targetLinkState"; return 2;  fi
	echo "ready for failover"
	return 0
}

# args: $1 number of pods that should be running
waitOnFailoverComplete() {
	npods=$1
	echo "waiting on FAILOVER link state"
	waitOnFailedOverLinkState
	echo "waiting on $npods pods running"
	waitOnNPodsRunning $npods
}

#=================================================== test iteration =============================================================
set iterationNumber=0

testIteration() {
	echo link states $(getSourceRGLinkState) $(getTargetRGLinkState)
	echo "replication-group labeled pods: "
	kubectl get pods -A -o wide -l replication-group.podmon.dellemc.com

	export clusterMode=$(getClusterMode)
	echo clusterMode $clusterMode
	isReadyForFailover
	status=$?
	echo status $status
	npodsrunning=$(numberPodsRunning)
	echo $npodsrunning pods running AAA
	# if ready for failover, kill connectivity to the appropriate array which should trigger a failover
	if [ $status -eq 0 ]; then
		echo "trying $mode"
		if [ "$clusterMode" == "src" ]; then
			killArray1
		fi
		if [ "$clusterMode" == "target" ]; then
			killArray2
		fi
	else
		# not ready for a failover, exit
		echo "iteration $iterationNumber is not ready for a failover"
		return 1
	fi 

	time waitOnFailoverComplete $npodsrunning

	# Restore connectivity to the array
	echo ping -c 1 $Array1IP
	ping -c 1 $Array1IP
	if [ $? -gt 0 ]; then
		restoreArray1
	fi
	echo ping -c $Array2IP
	ping -c 1 $Array2IP
	if [ $? -gt 0 ]; then
		restoreArray2
	fi

	# Get the RG state for our mode and initiate a reprotect
	export clusterMode=$(getClusterMode)
	if [ "$clusterMode" == "src" ]; then
		srcLinkState=$(getSourceRGLinkState)
		if [ "$srcLinkState" == "FAILEDOVER" ]; then
			echo "reprotecting source array"
			repctl reprotect --rg $REPLICATIONGROUPNAME
		fi
	else
		targetLinkState=$(getTargetRGLinkState)
		if [ "$targetLinkState" == "FAILEDOVER" ]; then
			echo "reprotecting target array"
			repctl reprotect --rg replicated-$REPLICATIONGROUPNAME
		fi
	fi

	# wait until the link status are SYNCHRONIZED again
	# Get the link state of each replication group
	waiting=1
	while [ $waiting -gt 0 ]
	do
		srcLinkState=$(getSourceRGLinkState)
		targetLinkState=$(getTargetRGLinkState)
		if [ "$srcLinkState" == "SYNCHRONIZED"  -a "$targetLinkState" == "SYNCHRONIZED" ]; then
			waiting=0
		else
			echo -n "waiting on SYNCHRONIZED: srcLinkState $srcLinkState targetLinkState $targetLinkState "
			date
			sleep 10
		fi
	done

	# print the pods and volume attachments
	kubectl get pods -A -o wide
	kubectl get rg
	kubectl get volumeattachments
	return 0
}

# Wait on either link state to indicate FAILEDOVER
waitOnFailedOverLinkState() {
	# Get the link state of each replication group
	waiting=1
	while [ $waiting -gt 0 ]
	do
		srcLinkState=$(getSourceRGLinkState)
		targetLinkState=$(getTargetRGLinkState)
		if [ "$srcLinkState" == "FAILEDOVER"  -o "$targetLinkState" == "FAILEDOVER" ]; then
			waiting=0
		else
			echo -n "waiting on FAILEDOVER: srcLinkState $srcLinkState targetLinkState $targetLinkState "
			date
			sleep 10
		fi
	done
}

# Wait on both link states to reach SYNCHRONIZED state
waitOnSynchronizedLinkState() {
	# wait until the link status are SYNCHRONIZED again
	# Get the link state of each replication group
	waiting=1
	while [ $waiting -gt 0 ]
	do
		srcLinkState=$(getSourceRGLinkState)
		targetLinkState=$(getTargetRGLinkState)
		if [ "$srcLinkState" == "SYNCHRONIZED"  -a "$targetLinkState" == "SYNCHRONIZED" ]; then
			waiting=0
		else
			echo -n "waiting on SYNCHRONIZED: srcLinkState $srcLinkState targetLinkState $targetLinkState "
			date
			sleep 10
		fi
	done
}


runIterations() {
iter=0
while [ $iter -lt 4 ]
do
	iter=$( expr $iter + 1 )
	echo "iteration $iter"
	testIteration
done
}

# $1 is the number of pods scale to be run
runScale(){
	sh ../podmontest/insv.sh --nvolumes 2 --instances $1 --storage-class rep217to206
	#sleep 60
	APPNAMESPACES=`kubectl get namespaces | awk '/pmtv/ { printf "%s ",$1; }'`
	echo "Applicaion namespaces: $APPNAMESPACES"
	waitonAllPodsRunning
	waitOnSynchronizedLinkState
	runIterations
}

runScale 2
#runScale 5
#runScale 10
#runScale 15
#runScale 20
#runScale 30
#runScale 40
#runScale 50

exit 0


