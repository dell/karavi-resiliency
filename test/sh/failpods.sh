#!/bin/sh
# Copyright (c) 2022 Dell Inc., or its subsidiaries. All Rights Reserved.
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

namespace=""
maxPods=10

for param in $*; do
  case $param in
  "--ns")
    shift
    namespace=$1
    shift
    ;;
  "--timeoutseconds")
    shift
    timeoutseconds=$1
    shift
    ;;
  "--help")
    shift
    echo "parameters: --ns driver_namespace [--timeoutseconds value]"
    exit
    ;;
  esac
done

[ "$namespace" = "" ] && echo "Required argument: --ns driver_namespace --timeoutseconds value" && exit 2

# nodeList returns a list of nodes
nodeList() {
	kubectl get nodes -A | grep -v -E 'mast.r|control-plane'  | grep -v NAME | awk '{ print $1 }'
}

# getNumOfPods returns the number of pods on a node $1 for a specific namespace
getNumOfPods() {
	# Add an extra space on match string to differentiate worker-1 from worker-10
	kubectl get pods -A -o wide | grep "$1 " | grep $namespace | wc -l 
}

# getWorker returns the first worker node that we are targeting
getWorker(){
    kubectl get nodes -A | grep -v -E 'mast.r|control-plane'  | grep -v NAME | awk 'NR==1{ print $1 }'
}

# getRunningPods returns the names of the running pods that are on the targeted worker node for a specific namespace
getRunningPods() {
    node=$(getWorker)
    # kubectl get pods -A -o wide | grep $node | grep $namespace | grep Running | awk '{ print $2 }'
    kubectl get pods -A -o wide | grep $node | grep $namespace | grep -v 'controller' | awk '{ print $2 }'
}

# getDriverImage returns the initial driver image before its patched
getDriverImage() {
    ns=$namespace
    pods=$(getRunningPods)
    for pod in $pods; do
        kubectl get pod $pod -n $ns -o custom-columns=IMAGE:.spec.containers[1].image | awk 'FNR == 2 {print}'
    done
}

# failPodsInNS will fail the pods for a specific namespace by patching it with an unknown driver image
failPodsInNS() {
    ns=$namespace
    pods=$(getRunningPods)
    for pod in $pods; do
        echo "Failing pods: $pods "
        kubectl patch pod $pod -n $ns --patch '{"spec": {"containers": [{"name": "driver", "image": "podmontest"}]}}'
    done
}

process_pods() {
    echo "Failing CSI driver pod for a single worker node..."
    initialImage=$(getDriverImage)

    # returns a list of nodes
	nodes=$(nodeList)
	echo Nodes: $nodes

    # returns # of pods on a node
	for n in $nodes; do
        pods=$(getNumOfPods $n)
        ns=$namespace
	    echo node $n has $pods pods in namespace $ns
	done

    namespaces=$namespace
    echo "Begin failing pods in namespace $ns"
	for ns in $namespaces; do
        failPodsInNS $ns
	done

    echo "Fail time in seconds:" $timeoutseconds
    sleep $timeoutseconds

    echo "Begin patching pods in namespace $ns"
    for ns in $namespaces; do
        echo "Patching pods: $pods "
        pod=$(getRunningPods)
        for pod in $pods; do
            kubectl patch pod $pod -n $ns --patch '{"spec": {"containers": [{"name": "driver", "image": "'${initialImage}'"}]}}'
        done
    done 

    echo "Waiting for $pods to come back"  
    for ns in $namespaces; do 
        node=$(getWorker)
        ns=$namespace
        podStatus=$(kubectl get pods -n $ns -o wide | grep $node | grep Running | grep -v NAME | wc -l)
        if [ $podStatus -gt 1 ]; then
		    sleep 60
        fi
    done
    echo "Fail test complete"
}

process_pods
