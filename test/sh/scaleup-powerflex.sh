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
STORAGECLASS=${STORAGECLASS:-vxflexos}
MAXINSTANCES=${MAXINSTANCES:-100} # Define the maximum number of instances (pods or VMs), Default to 100
ISVIRTUALIZATION=${ISVIRTUALIZATION:-false} # Default to false

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
    non_running_vms=$(kubectl get vms -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
    while [ $non_running_vms -gt 0 ]; do
        echo "Waiting on " $non_running_vms " VM to reach Running state"
        sleep 30
        non_running_vms=$(kubectl get vms -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
    done

    non_connected_vms=$(kubectl get vmis -A -o jsonpath='{.items[?(@.status.conditions[?(@.type=="AgentConnected")].status!="True")].metadata.name}' | wc -l)
    while [ $non_connected_vms -gt 0 ]; do
        echo "Waiting on " $non_connected_vms " VM to have AgentConnected condition"
        sleep 30
        non_connected_vms=$(kubectl get vmis -A -o jsonpath='{.items[?(@.status.conditions[?(@.type=="AgentConnected")].status!="True")].metadata.name}' | wc -l)
    done
}

date

if [ "$ISVIRTUALIZATION" = true ]; then
	echo "virtualization enabled: " $ISVIRTUALIZATION
	BOUNCEIPTIME=240
	instances="4"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	instances="18"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	instances="36"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	BOUNCEIPTIME=480
	instances="54"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900
	fi

	BOUNCEIPTIME=720
	instances="72"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=850
	instances="81"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=1000
	instances="90"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=1200
	instances="99"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
	fi

	BOUNCEIPTIME=1200
	instances="108"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS --workload-type vm; cd $CWD
	wait_on_running_vms
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
	fi
else
	BOUNCEIPTIME=240
	instances="4"
	if [ $instances -eq $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	instances="18"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	instances="36"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
	fi

	BOUNCEIPTIME=480
	instances="54"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900
	fi

	BOUNCEIPTIME=720
	instances="72"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=850
	instances="81"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=1000
	instances="90"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
	fi

	BOUNCEIPTIME=1200
	instances="99"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
	fi

	BOUNCEIPTIME=1200
	instances="108"
	if [ $instances -le $MAXINSTANCES ]; then
	cd ../podmontest; sh insv.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
	wait_on_running
	sh ../sh/nway.sh --ns vxflexos --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
	fi
fi

date
