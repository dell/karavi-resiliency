#!/bin
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

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
instances=${instances:-0}
ndevices=${ndevices:-0}
nvolumes=${nvolumes:-0}
zone=${zone:-""}
storageClassName=${storageClassName:-vxflexos-retain}
PODMONTEST_REGISTRY="$REGISTRY_HOST"
if [ -n "$REGISTRY_PORT" ]; then
   PODMONTEST_REGISTRY="$PODMONTEST_REGISTRY:$REGISTRY_PORT"
fi
image="$PODMONTEST_REGISTRY/podmontest:v0.0.58"
prefix="pmtv"
replicas=1
deploymentType="statefulset"
driverLabel="csi-vxflexos"
podAffinity="false"
unreachableTolerationSeconds=300
workloadType=${workloadType:-"pod"}

if [ "$DEBUG"x != "x" ]; then
   DEBUG="--dry-run --debug"
fi

for param in $*
do
    case $param in
       "--instances")
          shift
          instances=$1
          shift
          ;;
       "--ndevices")
          shift
          ndevices=$1
          shift
          ;;
       "--nvolumes")
          shift
          nvolumes=$1
          shift
          ;;
       "--prefix")
          shift
          prefix=$1
          shift
          ;;
       "--storage-class")
          shift
          storageClassName=$1
          shift
          ;;
       "--replicas")
          shift
	  replicas=$1
	  shift
	  ;;
       "--podAffinity")
          podAffinity="true"
          shift
	  ;;
       "--deployment")
          deploymentType="deployment"
          shift
          ;;
       "--unreachableTolerationSeconds")
          shift
          unreachableTolerationSeconds=$1
          shift
          ;;
       "--label")
          shift
          driverLabel=$1
          shift
          ;;

      "--workload-type")
         shift
         workloadType=$1
         shift
         ;;
    esac

done

cd "$SCRIPTDIR"

i=1
while [ $i -le $instances ]; do
  echo $i
  kubectl create namespace ${prefix}$i
if [ "$workloadType" == "pod" ]; then
  helm install -n "${prefix}$i" "${prefix}$i" deploy \
              ${DEBUG} \
              --values deploy/values-vxflex.yaml \
              --set podmonTest.namespace="${prefix}$i" \
              --set podmonTest.storageClassName="$storageClassName" \
              --set podmonTest.ndevices=$ndevices \
              --set podmonTest.nvolumes=$nvolumes \
              --set podmonTest.deploymentType=$deploymentType \
              --set podmonTest.replicas=$replicas \
              --set podmonTest.podAffinity=$podAffinity \
              --set podmonTest.unreachableTolerationSeconds=$unreachableTolerationSeconds \
              --set podmonTest.image="$image" \
              --set podmonTest.zone="$zone" \
              --set podmonTest.driverLabel="$driverLabel"
else
   helm install -n "${prefix}${i}" "${prefix}${i}" "${SCRIPTDIR}"/deploy \
     ${DEBUG} \
     --values deploy/values-vm.yaml \
     --set vmConfig.namespace="${prefix}${i}" \
     --set vmConfig.storageClassName="$storageClassName" \
     --set vmConfig.ndevices=$ndevices \
     --set vmConfig.nvolumes=$nvolumes \
     --set vmConfig.instances=$instances \
     --set vmConfig.podAffinity=$podAffinity \
     --set vmConfig.unreachableTolerationSeconds=$unreachableTolerationSeconds \
     --set vmConfig.zone="$zone" \
     --set vmConfig.driverLabel="$driverLabel"
 fi
 i=$((i + 1))

done
