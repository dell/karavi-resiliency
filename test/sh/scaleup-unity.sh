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

CWD=$(pwd)
NVOLUMES=2
STORAGECLASS=unity-nfs
MAXPODS=81

# checks that all labeled pods are running, exits if not
wait_on_running() {
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	non_running_vms=$(kubectl get vm -o wide | grep -v AGE | grep -v Running | wc -l)
	non_running_vmis=$(kubectl get vmi -o wide | grep -v AGE | grep -v Running | wc -l)
	non_running_vm_controller=$(kubectl get pod -o wide | grep -v AGE | grep -i virt-launcher | grep -v Running | wc -l)

	while [ $non_running_pods -gt 0 ]; do
		echo "Waiting on " $non_running_pods " pod to reach Running state"
		sleep 30
		non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	done
	
	while [ $non_running_vms -gt 0 ]; do
		echo "Waiting on " $non_running_vms " vm to reach Running state"
		sleep 30
		non_running_vms=$(kubectl get vm -o wide | grep -v AGE | grep -v Running | wc -l)
	done

	while [ $non_running_vmis -gt 0 ]; do
		echo "Waiting on " $non_running_vmis " vmi to reach Running state"
		sleep 30
		non_running_vmis=$(kubectl get vmi -o wide | grep -v AGE | grep -v Running | wc -l)
	done

	while [ $non_running_vm_controller -gt 0 ]; do
		echo "Waiting on " $non_running_vm_controller " vm controller pod to reach Running state"
		sleep 30
		non_running_vm_controller=$(kubectl get pod -o wide | grep -v AGE | grep -i virt-launcher | grep -v Running | wc -l)
	done
}

date


BOUNCEIPTIME=240
instances="1"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
fi


# BOUNCEIPTIME=240
# instances="9"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
# fi

# instances="18"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
# fi

# BOUNCEIPTIME=360
# instances="27"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
# fi

# instances="36"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
# fi

# BOUNCEIPTIME=480
# instances="54"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900
# fi

# instances="72"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
# fi

# BOUNCEIPTIME=720
# instances="81"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
# fi

# instances="90"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
# fi

# instances="99"
# if [ $instances -le $MAXPODS ]; then
# cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
# wait_on_running
# sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
# fi

date
