<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Limitaions and Exclusions

This file contains information on Limitations and Exclusions that users should be aware of. Additionally there are driver specific limitations and exclusions that may be called out in the "Deploying Karavi Resiliency" page.

## Supported and Tested Operating Modes

The following provisioning types are supported and have been tested:

* Dynamic PVC/PVs of accessModes "ReadWriteOnce" and volumeMode "FileSystem".
* Dynamic PVC/PVs of accessModes "ReadWriteOnce" and volumeMode "Block".
* Use of the above volumes with Pods created by StatefulSets.
* Up to 12 or so protected pods on a given node.
* Failing up to 3 nodes at a time in 9 worker node clusters, or failing 1 node at a time in smaller clusters. Application recovery times are dependent on the number of pods that need to be moved as a result of the failure. See the section on "Testing and Performance" for some of the details. The scale testing was done almost exclusively on PowerFlex volumes.

## Not Tested But Assumed to Work

* Deployments with the above volume types, provided two pods from the same deployment do not reside on the same node. At the current time anti-affinity rules should be used to guarantee no two pods accessing the same volumes are scheduled to the same node.
* Multi-array support 

## Not Yet Tested or Supported

* Pods that use persistent volumes from multiple CSI drivers. This _cannot_ be supported because multiple controller-podmons (one for each driver type) would be trying to manage the failover with conflicting actions.

* ReadWriteMany volumes. This may have issues if a node has multiple pods accessing the same volumes. In any case once pod cleanup fences the volumes on a node, they will no longer be available to any pods using those volumes on that node. We will endavor to support this in the future.

* Multiple instances of the same driver type (for example two CSI driver for Dell EMC PowerFlex deployments.)



