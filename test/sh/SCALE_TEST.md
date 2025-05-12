<!--
 Copyright (c) 2021-2025 Dell Inc., or its subsidiaries. All Rights Reserved.

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

# Scale Testing

This page describes a script based facility for running scalability testing. Currently, it supports PowerFlex, Unity, PowerScale, PowerStore and PowerMax.

It comprises multiple scripts that work together. The top level script is _scaleup-powerflex.sh_ / _scaleup-unity.sh_ / _scaleup-powerscale.sh_ / _scaleup-powerstore.sh_ / _scaleup-powermax.sh_.
It uses the scripts in podmontest _insv.sh_ and _uns.sh_ to deploy or terminate pods/VMs.
The number of pods/VMs deployed is configurable, and the scaleup scripts starts at a small scale
and gradually scales up the number of deployed pods/VMs from a minimal amount to the maximum number of protected
pods/VMs to be tested. While running the scaleup scripts, the caller can specify the maximum number of pods/VMs to be scaled using --maxinstances and to run VM workloads --isvirtualization should be enabled.

At each number of pods/VMs to be tested, scaleup-powerflex.sh/scaleup-unity.sh/scaleup-powerscale.sh/scaleup-powerstore.sh invokes _nway.sh_ which runs the actual testing.
Nway.sh provides up to 10 groups of nodes that are failed as a unit- so you can divide your cluster into
any number of groups between 2 and 10. These are configured in the NODELIST1... NODELIST10 variables.
The test fails each configured group in a successive iteration (empty groups are skipped).
When invoked, the caller can specify how long the interfaces are to be down using --bounceipsec argument,
the maximum number of iterations to do using --maxiterations, the timeout value for an iteration using
--timeoutseconds, the workload type to be tested either vm/pod using --workload-type, the username and password of nodes for copying scripts into them using --node-user and --password.

As mentioned above either we can specify values while invoking scripts or we can export them before running scripts as mentioned below:

    export REGISTRY_PORT='5000'
    export REGISTRY_HOST='10.247.98.98'
    export MAXINSTANCES=<max pods/VMs to be scaled>
    export ISVIRTUALIZATION=true/false
    export NODE_USER=core/root
    export PASSWORD=<password>
    
    
Each iteration is divided into three phases:

1. Evacuation of pods/VMs from nodes that are down. At the end of this phase a message is printed similar to
"movedPods: 4  evacuation time seconds:  30". This allows you to determine how long until all pods were rescheduled.

2. Waiting on the pods/VMs that were rescheduled to reach a running state. At the end of this phase a message
similar to "moving pods:  4 time for pod recovery seconds:  70" is printed. This is the time from the initiation of
node failure until all the pods/VMs were moved and reach the running state again. This is the metric generally used
for scalability, that plots on the X-axis the number of pods that were impacted, and the Y-axis the time until all 
pods were recovered.

3. Waiting on the taints to be removed from the failed nodes. At the end of this phase a message is printed
that gives the length of time after the interfaces have been restored to operational state until all the taints
have been removed (indicating the nodes are cleaned up.)

At the end of an iteration after a 60 second delay, the status of all the protected pods is displayed, and plot will be generated for Number of Instances vs. Time Taken for Recovery of pods
and if any pods are not running nway.sh exits.

To generate plot we need to install below
    yum install -y python3-pip
    pip3 install pandas matplotlib

If the script times out, the collect_logs.sh script is called to collect all the logs necessary to analyze the potential failure.

