<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Supported CSI Drivers

Dell currently develops and maintains the following CSI Drivers:

* [PowerFlex CSI Driver](https://github.com/dell/csi-vxflexos)
* [PowerScale CSI Driver](https://github.com/dell/csi-powerscale)
* [PowerStore CSI Driver](https://github.com/dell/csi-powerstore)
* [PowerMax CSI Driver](https://github.com/dell/csi-powermax)
* [Unity CSI Driver](https://github.com/dell/csi-unity)
* [Bare Metal CSI Driver](https://github.com/dell/csi-baremetal)

Currently in the initial Tech. Preview Karavil Resiliency only provides complete support for Power Flex and limited support for Unity. Additional array support in Karavil Resiliency is planned for the near future.

## PowerFlex Support

PowerFlex is a highly scalable array that is very well suited to Kubernetes deployments. The Karavil Resiliency support for PowerFlex leverages the following PowerFlex features:

* Very quick detection of Array I/O Network Connectivity status changes (generally takes 1-2 seconds for the array to detect changes)
* A roboust mechanism if Nodes are doing I/O to volumes (sampled over a 5 second period).
* Low latency REST API supports fast CSI provisioning and deprovisioning operations.
* A proprietary network protocol provided by the SDC component that can run over the same IP interface as the K8S control plane or over a separate IP interface for Array I/O.

## Unity Support

The Unity support in Karavil Resiliency has only begun recently as is not completed. Nevertheless, it is possible to test on a limited basis in the Tech. Preview:

* Initially supports NFS and iSCSI protocols. (Fibrechannel is not supported yet.)
* Array connectivity detection is implemented for iSCSI only. However detection times are slow (on the order of a minute or more.)
* The mechanism to detect I/O is in progress to volumes is not yet implemented.

