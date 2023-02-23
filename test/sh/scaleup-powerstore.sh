#!/bin/sh
# Copyright (c) 2023 Dell Inc., or its subsidiaries. All Rights Reserved.
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
STORAGECLASS=powerstore-nfs
MAXPODS=100

# checks that all labeled pods are running, exits if not
wait_on_running() {
	non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	while [ $non_running_pods -gt 0 ]; do
		echo "Waiting on " $non_running_pods " pod to reach Running state"
		sleep 30
		non_running_pods=$(kubectl get pods -l podmon.dellemc.com/driver -A -o wide | grep -v NAMESPACE | grep -v Running | wc -l)
	done
}

date

BOUNCEIPTIME=240
instances="4"
if [ $instances -eq $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
fi

instances="18"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
fi

instances="36"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 600
fi

BOUNCEIPTIME=480
instances="54"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 900
fi

BOUNCEIPTIME=720
instances="72"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
fi

BOUNCEIPTIME=850
instances="81"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
fi

BOUNCEIPTIME=1000
instances="90"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12 --timeoutseconds 1300
fi

BOUNCEIPTIME=1200
instances="99"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
fi

BOUNCEIPTIME=1200
instances="108"
if [ $instances -le $MAXPODS ]; then
cd ../podmontest; sh insps.sh --instances "$instances" --nvolumes $NVOLUMES --storage-class $STORAGECLASS; cd $CWD
wait_on_running
sh ../sh/nway.sh --ns powerstore --bounceipseconds $BOUNCEIPTIME --maxiterations 12  --timeoutseconds 1500
fi

date
