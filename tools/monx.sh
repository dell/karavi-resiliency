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
alias k=kubectl
once=$1
while true;
do
	date
	k get nodes -o wide
	k get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints
	k get pods -l podmon.dellemc.com/driver -A -o wide | awk  '
BEGIN	{
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
	if [ "$once" = "--once" ]; then exit 0; fi
	sleep 5
done
