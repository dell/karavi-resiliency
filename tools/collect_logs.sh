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

# collect_logs.sh is used to collect the CSI driver logs, podmon logs, and protected pod namespace events
# for podmon failure analysis. Type "collect.logs.sh --help" for information.

ns=""
podmon_label="podmon.dellemc.com/driver"
CWD=$(pwd)
TAR=$(which tar)
CONTAINERS="podmon driver"

for param in $*; do
  case $param in
  "--ns")
    shift
    ns=$1
    shift
    ;;
  "--label")
    shift
    podmon_label=$1
    shift
    ;;
  "--help")
    shift
    echo "parameters: --ns driver_namespace [ --label podmon_label ]"
    exit
    ;;
  esac
done

[ "$ns" = "" ] && echo "Required argument: --ns driver_namespace" && exit 2
echo "Collecting logs driver namespace $ns podmon label $podmon_label"

getpods() {
	pods=$(kubectl get pods -n $ns | awk '/^NAME/ { next; }; /.*/ { print $1}')
	echo $pods
}

getprotectedpods() {
	protectedpods=$(kubectl get pods -A -l $podmon_label | awk '/^NAME/ { next; }; /.*/ { print $1}')
	echo $protectedpods
}

TEMPDIR=$(mktemp -d)
echo "Using TEMPDIR $TEMPDIR"
TIMESTAMP=$(date +%Y%m%d_%H%M)
echo $TIMESTAMP
pods=$(getpods)
echo pods $pods

# Collect the logs into the TEMPDIR
cd $TEMPDIR
kubectl get nodes -o wide >nodes.list
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints >taints.list
kubectl get pods -n $ns -o wide >driver.pods.list
kubectl get pods -A -o wide -l $podmon_label >protected.pods.list
for pod in $pods; 
do
	for container in $CONTAINERS;
	do
		echo kubectl logs -n $ns $pod $container >$ns.$pod.$container.log
		kubectl logs -n $ns $pod $container >$ns.$pod.$container.log
	done
done
# Collect the events for the protected pod namespaces
protectedpods=$(getprotectedpods)
for podns in $protectedpods;
do
	count=$(kubectl get events -n $podns | grep -v '^LAST' | wc -l)
	if [ $count -gt 0 ]; then
		kubectl get events -n $podns >$podns.events
	fi
done

DIRNAME=$(basename $TEMPDIR)
TARNAME="$CWD/driver.logs.$TIMESTAMP.tgz"
cd /tmp

# Tar up the logs using the time stamp
echo "$TAR" -c -z -v -f  $TARNAME $DIRNAME
$TAR -c -z -v -f  $TARNAME $DIRNAME

# Remove the temporary directory
rm -rf $TEMPDIR

