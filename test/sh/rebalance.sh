#!/bin/sh

# Rebalance pods by 
# 1. tainting overloaded nodes,
# 2. removing pods from partially populated namespaces,
# 3. Waiting 10 minutes for the pods to get recreated,
# 4. Removing the taints that were applied

maxPods=90

# nodelist returns a list of nodes(
nodelist() {
	kubectl get nodes -A | grep -v master  | grep -v NAME | awk '{ print $1 }'
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
        echo "Rebalancing pods to nodes..."
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
		sleep 60
	done
	for n in $cordonedNodes; do
		uncordon $n
	done
	echo "Rebalancing complete"
}

rebalance
