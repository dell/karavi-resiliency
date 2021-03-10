<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Recovering from Failures

Normally Karavi Resiliency should be able to move pods that have been impacted by Node Failures to a healthy node, and after the failed nodes have come back on line, clean them up (especially any potential zombie pods) and then automatically remove the Karavi Resiliency node taint that prevents pods from being scheduled to the failed node(s). There are a few cases where this cannot be fully automated and operator intervention is required, including:

1. Karavi Resiliency expects that when a node faiure occurs, all Karavi Resiliency labeled pods are evacuated and reschedule on other nodes. This process may not complete however if the node comes back online before Karavi Resiliency has had time to evacuate all the labeled pods. The remaining pods may not restart correctly, going to "Error" or "CrashLoopBackoff". We are considering some possible remediations for this condition but have not implemented them yet.

    If this happens, try deleting the pod with "kubectl delete pod ...". In our experience this normally will cause the pod to be restarted and transition to the "Running" state.

2. Podmon-node is responsible for cleaning up failed nodes after the nodes communication has been restored. The algorithmm checks to see that all the monitored pods have terminated and their volumes and mounts have been cleaned up.

    If some of the monitored pods are still executing, node-podmon will emit the following log message at the end of a cleanup cycle (and retry the cleanup after a delay):

    ```
    pods skipped for cleanup because still present: <pod-list>
    ```
    If this happens, __DO NOT__ manually remove the the Karavi Resiliency node taint. Doing so could possibly cause data corruption if volumes were not cleaned up and a pod using those volumes was subsequently scheduled to that node.

    The correct course of action in this case is to reboot the failed node(s) that have not removed their taints in a reasonable time (5-10 minutes after the node is online again.) The operator can delay executing this reboot until it is convenient, but new pods will not be scheduled to it in the interim. This reboot will kill any potential zombie pods. After the reboot, node-podmon should automatically remove the node taint after a short time.
