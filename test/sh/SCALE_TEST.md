# Scale Testing

This page describes a script based facility for running scalability testing. Currently it supports PowerFlex.

It is comprised of multiple scripts that work together. The top level script is _scaleup-powerflex.sh_.
It uses the scripts in podmontest _insv.sh_ and _uns.sh_ to deploy or terminate podmontest pods.
The number of pods deployed is configurable, and the scaleup-powerflex.sh script starts at a small scale
and gradually scales up the number of deployed pods from a minimal amount to the maximum number of protected
pods to be tested.

At each number of pods to be tested, scaleup-powerflex.sh invokes _nway.sh_ which runs the actual testing.
Nway.sh provides up to 10 groups of nodes that are failed as a unit- so you can divide your cluster into
any number of groups between 2 and 10. These are configured in the NODELIST1... NODELIST10 variables.
The test fails each configured group in a successive iteration (empty groups are skipped).
When invoked, the caller can specify how long the interfaces are to be down using --bounceipsec argument,
the maximum number of iterations to do using --maxiterations, and the timeout value for an iteration using
--timeoutseconds.

Each iteration is divided into three phases:

1. Evacuation of pods from nodes that are down. At the end of this phase a message is printed similar to
"movedPods: 4  evacuation time seconds:  30". This allows you to determine how long until all pods were rescheduled.

2. Waiting on the pods that were rescheduled to reach a running state. At the end of this phase a message
similar to "moving pods:  4 time for pod recovery seconds:  70" is printed. This is the time from the initiation of
node failure until all the pods were moved and reach the running state again. This is the metric generally used
for scalability, that plots on X-axis the number of pods that were impacted, and the Y-axis the time until all 
pods were recovered.

3. Waiting on the taints to be removed from the failed nodes. At the end of this phase a message is printed
that gives the length of time after the interfaces have been restored to operational state until all the taints
have been removed (indicating the nodes are cleaned up.)

At the end of an iteration after a 60 second delay, the status of all the protected pods is displayed,
and if any pods are not running nway.sh exits.

If the script times out, the collect_logs.sh script is called to collect all the logs necessary to analyze the potential failure.

