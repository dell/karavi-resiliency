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

Currently, in the initial Technical Preview, CSM for Resiliency only provides complete support for PowerFlex. Additional array support in CSM for Resiliency is planned for the near future.

## PowerFlex Support

PowerFlex is a highly scalable array that is very well suited to Kubernetes deployments. The CSM for Resiliency support for PowerFlex leverages the following PowerFlex features:

* Very quick detection of Array I/O Network Connectivity status changes (generally takes 1-2 seconds for the array to detect changes)
* A robust mechanism if Nodes are doing I/O to volumes (sampled over a 5-second period).
* Low latency REST API supports fast CSI provisioning and de-provisioning operations.
* A proprietary network protocol provided by the SDC component that can run over the same IP interface as the K8S control plane or over a separate IP interface for Array I/O.

## Unity Support

Dell EMC Unity is targeted for midsized deployments, remote or branch offices, and cost-sensitive mixed workloads. Unity systems are designed for all-Flash, deliver the best value in the market, and are available in purpose-built (all Flash or hybrid Flash), converged deployment options (through VxBlock), and a software-defined virtual edition. 

* Unity (purpose built): A modern midrange storage solution, engineered from the groundup to meet market demands for Flash, affordability and incredible simplicity. The Unity Family is available in 12 All Flash models and 12 Hybrid models.
* VxBlock (converged): Unity storage options are also available in Dell EMC VxBlock System 1000.
* UnityVSA (virtual): The Unity Virtual Storage Appliance (VSA) allows the advanced unified storage and data management features of the Unity family to be easily deployed on VMware ESXi servers, for a ‘software defined’ approach. UnityVSA is available in two editions:
  * Community Edition is a free downloadable 4 TB solution recommended for nonproduction use.
  * Professional Edition is a licensed subscription-based offering available at capacity levels of 10 TB, 25 TB, and 50 TB. The subscription includes access to online support resources, EMC Secure Remote Services (ESRS), and on-call software- and systems-related support.

All three deployment options, i.e. Unity, UnityVSA, and Unity-based VxBlock, enjoy one architecture, one interface with consistent features and rich data services.