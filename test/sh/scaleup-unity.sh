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

CWD=$(pwd)
NVOLUMES=2
STORAGECLASS=unity-nfs
MAXINSTANCES=${MAXINSTANCES:-100} # Define the maximum number of instances (pods or VMs), Default to 100
ISVIRTUALIZATION=${ISVIRTUALIZATION:-false} # Default to false
NODE_USER=${NODE_USER:-root}
PASSWORD=${PASSWORD:-""}

for param in $*
do
    case $param in
       "--maxinstances")
          shift
          MAXINSTANCES=$1
          shift
          ;;
       "--isvirtualization")
          shift
          ISVIRTUALIZATION=$1
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
    esac
done

# checks that all labeled pods are running, exits if not
wait_on_running() {
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	while [ $non_running_pods -gt 0 ]; do
		echo "Waiting on " $non_running_pods " pod to reach Running state"
		sleep 30
		non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	done
}

# checks that all VM instances are running, exits if not
wait_on_running_vms() {
    non_running_vms=$(kubectl get vms -A -o json | jq '[.items[] | select(.spec.template.metadata.labels["podmon.dellemc.com/driver"] and .status.printableStatus != "Running")] | length')
    while [ $non_running_vms -gt 0 ]; do
        echo "Waiting on " $non_running_vms " VM to reach Running state"
        sleep 30
        non_running_vms=$(kubectl get vms -A -o json | jq '[.items[] | select(.spec.template.metadata.labels["podmon.dellemc.com/driver"] and .status.printableStatus != "Running")] | length')
    done
    echo "VMs reached Running state"
}

date

if [ "$ISVIRTUALIZATION" = true ]; then
	echo "virtualization enabled: " $ISVIRTUALIZATION
	BOUNCEIPTIME=240
	instances="10"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600 --workload-type vm --node-user core
	fi

	BOUNCEIPTIME=300
	instances="20"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900 --workload-type vm --node-user core
	fi

	BOUNCEIPTIME=360
	instances="30"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1200 --workload-type vm --node-user core
	fi

	BOUNCEIPTIME=480
	instances="40"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1500 --workload-type vm --node-user core
	fi

	BOUNCEIPTIME=600
	instances="50"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1800 --workload-type vm --node-user core
	fi

	instances="60"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 2100 --workload-type vm --node-user core
	fi

	BOUNCEIPTIME=720
	instances="70"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 2300 --workload-type vm --node-user core
	fi

	instances="80"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 2500 --workload-type vm --node-user core
	fi

	instances="90"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 2800 --workload-type vm --node-user core
	fi
else
	BOUNCEIPTIME=240
	instances="9"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600 --node-user $NODE_USER --password $PASSWORD
	fi

	instances="18"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600 --node-user $NODE_USER --password $PASSWORD
	fi

	BOUNCEIPTIME=360
	instances="27"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600 --node-user $NODE_USER --password $PASSWORD
	fi

	instances="36"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600 --node-user $NODE_USER --password $PASSWORD
	fi

	BOUNCEIPTIME=480
	instances="54"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900 --node-user $NODE_USER --password $PASSWORD
	fi

	instances="72"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300 --node-user $NODE_USER --password $PASSWORD
	fi

	BOUNCEIPTIME=720
	instances="81"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300 --node-user $NODE_USER --password $PASSWORD
	fi

	instances="90"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300 --node-user $NODE_USER --password $PASSWORD
	fi

	instances="99"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insu.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns unity --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500 --node-user $NODE_USER --password $PASSWORD
	fi
fi

date
