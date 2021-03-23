<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Supported CSI Drivers

Dell currently develops and maintains the following CSI Drivers:

* [PowerFlex CSI Driver](https://github.com/dell/csi-powerflex)
* [PowerScale CSI Driver](https://github.com/dell/csi-powerscale)
* [PowerStore CSI Driver](https://github.com/dell/csi-powerstore)
* [PowerMax CSI Driver](https://github.com/dell/csi-powermax)
* [Unity CSI Driver](https://github.com/dell/csi-unity)
* [Bare Metal CSI Driver](https://github.com/dell/csi-baremetal)

Currently, in the initial Technical Preview, Karavi Resiliency only provides complete support for PowerFlex. Additional array support in Karavi Resiliency is planned for the near future.

## PowerFlex Support

PowerFlex is a highly scalable array that is very well suited to Kubernetes deployments. The Karavi Resiliency support for PowerFlex leverages the following PowerFlex features:

* Very quick detection of Array I/O Network Connectivity status changes (generally takes 1-2 seconds for the array to detect changes)
* A roboust mechanism if Nodes are doing I/O to volumes (sampled over a 5 second period).
* Low latency REST API supports fast CSI provisioning and deprovisioning operations.
* A proprietary network protocol provided by the SDC component that can run over the same IP interface as the K8S control plane or over a separate IP interface for Array I/O.
