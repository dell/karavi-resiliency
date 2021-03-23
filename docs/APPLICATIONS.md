<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Deploying and managing applications protected By Karavi Resiliency

 The first thing to remember about _Karavi Resiliency_ is that it only takes action on pods configured with the designated label. Both the key and the value have to match what is in the podmon helm configuration. Karavi Resiliency emits a log message at startup with the label key and value it is using to monitor pods:

 ```
 labelSelector: {map[podmon.dellemc.com/driver:csi-vxflexos]
 ```
 The above message indicates the key is: podmon.dellemc.com/driver and the label value is csi-vxflexos. To search for the pods that would be monitored, try this:
 ```
[root@lglbx209 podmontest]# kubectl get pods -A -l podmon.dellemc.com/driver=csi-vxflexos
NAMESPACE   NAME           READY   STATUS    RESTARTS   AGE
pmtu1       podmontest-0   1/1     Running   0          3m7s
pmtu2       podmontest-0   1/1     Running   0          3m8s
pmtu3       podmontest-0   1/1     Running   0          3m6s
 ```

 If Karavi Resiliency detects a problem with a pod caused by a node or other failure that it can initiate remediation for, it will add an event to that pod's events:
 ```
 kubectl get events -n pmtu1
 ...
 61s         Warning   NodeFailure              pod/podmontest-0              podmon cleaning pod [7520ba2a-cec5-4dff-8537-20c9bdafbe26 lglbx215.lss.emc.com] with force delete
...
 ```

 Karavi Resiliency may also generate events if it is unable to cleanup a pod for some reason. For example, it may not clean up a pod because the pod is still doing I/O to the array.

 ### Important
 Before putting an application into production that relies on Karavi Resiliency monitoring, it is important to do a few test failovers first. To do this take the node that is running the pod offline for at least 2-3 minutes. Verify that there is an event message similar to the one above is logged, and that the pod recovers and restarts normally with no loss of data. (Note that if the node is running many Karavi Resiliency protected pods, the node may need to be down longer for Karavi Resiliency to have time to evacuate all the protected pods.)

 ## Application Recommendations

 1. It is recommended that pods that will be monitored by Karavi Resiliency be configured to exit if they receive any I/O errors. That should help achieve the recovery as quickly as possible.

 2. Karavi Resiliency does not directly monitor application health. However if standard Kubernetes health checks are configured, that may help reduce pod recovery time in the event of node failure, as Karavi Resiliency should receive an event that the application is Not Ready. Note that a Not Ready pod is not sufficient to trigger Karavi Resiliency action unless there is also some condition indicating a Node failure or problem, such as the Node is tainted, or the array has lost connectivity to the node.

 3. As noted previously in the Limitations and Exclusions section, Karavi Resiliency has not yet been verified to work with ReadWriteMany or ReadOnlyMany volumes. Also it has not been verified to work with pod controllers other than StatefulSet.
