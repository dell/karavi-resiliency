<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Karavi Resiliency General Description

_Karavi Resiliency_ is a project designed to make Kubernetes Applications, including those that utilize persistent storage, more resilient to various failures. The first component of _Karavi Resiliency_ is a pod monitor that is specifically designed to protect stateful applications from various failures. It is not a standalone application, but rather is deployed as a _sidecar_ to CSI (Container Storage Interface) drivers, in both the driver's controller pods and the driver's node pods. Deploying Karavi Resiliency as a sidecar allows it to make direct requests to the driver through the Unix domain socket that Kubernetes sidecars use to make CSI requests.

The pod monitor sidecar is currently named "podmon", but it might be renamed to "resiliency" or "karavi-resiliency" in future releases. When used in this document, podmon refers specifically to the sidecar container.

Some of the methods Karavi Resiliency invokes in the driver are standard CSI methods, such as NodeUnpublishVolume, NodeUnstageVolume, and ControllerUnpublishVolume. Karavi Resiliency also uses proprietary calls that are not part of the standard CSI specification. Currently there is only one, ValidateVolumeHostConnectivity that returns information on whether a host is connected to the storage system and/or whether any I/O activity has happened in the recent past from a list of specified volumes. This allows Karavi Resiliency to make more accurate determinations about the state of the system and its persistent volumes.

Accordingly Karavi Resiliency is adapted to, and qualified with each CSI driver it is to be used with. Different storage systems have different nuances and characteristics that Karavi Resiliency must take into account.

Karavi Resiliency is currently in a _Technical Preview Phase_, and should be considered _alpha_ software. We are actively seeking feedback from users about its features, effectiveness, and reliability. We will take that input, along with our own results from doing extensive testing, and incrementally improve the software. We do ***not*** recommend or support it for production use yet.

The rest of the documentation is organized as follows:

## I. [Use Cases](USE_CASES.md) 
Contains descriptions of the types of Kubernetes system failures that _Karavi Resiliency_ was designed to assist with. 

## II. [Supported Drivers, Access Protocols, and Driver Features](SUPPORTED_DRIVERS.md)
Contains details about what drivers are supported, and what features are available with each driver.

## III. [Karavi Resiliency Design and How It Works](DESIGN.md)
Contains the details about the design you need to need to know. 

## IV. [Limitations and Exclusions](LIMITATIONS.md)
Contains information on limitations. Please read this- especially for the _alpha_ stage, as not all features are implemented during _alpha_.

## V. [Deploying Karavi Resiliency](DEPLOYING.md)
Contains information on how to deploy _Karavi Resiliency_ as part of the driver installation process.

## VI. [Deploying and Managing Applications Protected By Karavi Resiliency](APPLICATIONS.md)
Contains information on how to deploy protected applications and how to know they are protected.

## VII. [Recovering from Failures](RECOVERY.md)
Contains important information about how to recover when failures cannot be resolved automatically.

## VIII. [Reporting Problems](PROBLEMS.md)
This section explains what information we need to diagnose the cause of problems with the _Karavi Resiliency_ protection systems. This information should be submitted in any issues if possible.

## IX. [Testing Methodology and Results](TESTING.md)
This section contains information how we tested _Karavi Resiliency_ and the results we achieved.




