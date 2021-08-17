<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Dell EMC Container Storage Module (CSM) for Resiliency
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Releases](https://img.shields.io/badge/Releases-green.svg)](https://github.com/dell/karavi-resiliency/releases)

User applications can have problems if you want their Pods to be resilient to node failure. This is especially true of those deployed with StatefulSets that use PersistentVolumeClaims. Kubernetes guarantees that there will never be two copies of the same StatefulSet Pod running at the same time and accessing storage. Therefore, it does not clean up StatefulSet Pods if the node executing them fails.
 
For the complete discussion and rationale, go to https://github.com/kubernetes/community and search for the pod-safety.md file (path: contributors/design-proposals/storage/pod-safety.md).
For more background on forced deletion of Pods in a StatefulSet, please visit [Force Delete StatefulSet Pods](https://kubernetes.io/docs/tasks/run-application/force-delete-stateful-set-pod/#:~:text=In%20normal%20operation%20of%20a,1%20are%20alive%20and%20ready).

# CSM for Resiliency High Level Description

_CSM for Resiliency_ is a project designed to make Kubernetes Applications, including those that utilize persistent storage, more resilient to various failures. The first component of _CSM for Resiliency_ is a pod monitor that is specifically designed to protect stateful applications from various failures. It is not a standalone application, but rather is deployed as a _sidecar_ to CSI (Container Storage Interface) drivers, in both the driver's controller pods and the driver's node pods. Deploying CSM for Resiliency as a sidecar allows it to make direct requests to the driver through the Unix domain socket that Kubernetes sidecars use to make CSI requests.

Some of the methods CSM for Resiliency invokes in the driver are standard CSI methods, such as NodeUnpublishVolume, NodeUnstageVolume, and ControllerUnpublishVolume. CSM for Resiliency also uses proprietary calls that are not part of the standard CSI specification. Currently, there is only one, ValidateVolumeHostConnectivity that returns information on whether a host is connected to the storage system and/or whether any I/O activity has happened in the recent past from a list of specified volumes. This allows CSM for Resiliency to make more accurate determinations about the state of the system and its persistent volumes.

Accordingly, CSM for Resiliency is adapted to, and qualified with each CSI driver it is to be used with. Different storage systems have different nuances and characteristics that CSM for Resiliency must take into account.

CSM for Resiliency is currently in a _Technical Preview Phase_, and should be considered _alpha_ software. We are actively seeking feedback from users about its features, effectiveness, and reliability. Please provide feedback using the karavi@dell.com email alias. We will take that input, along with our own results from doing extensive testing, and incrementally improve the software. We do ***not*** recommend or support it for production use at this time.

# Table of Contents

## [Use Cases](docs/USE_CASES.md) 
Contains descriptions of the types of Kubernetes system failures that _CSM for Resiliency_ was designed to assist with. 

## [Supported Drivers, Access Protocols, and Driver Features](docs/SUPPORTED_DRIVERS.md)
Contains details about what drivers are supported, and what features are available with each driver.

## [CSM for Resiliency Design and How It Works](docs/DESIGN.md)
Contains the details about the design you need to need to know. 

## [Limitations and Exclusions](docs/LIMITATIONS.md)
Contains information on limitations. Please read this- especially for the _alpha_ stage, as not all features are implemented during _alpha_.

## [Deploying CSM for Resiliency](docs/DEPLOYING.md)
Contains information on how to deploy _CSM for Resiliency_ as part of the driver installation process.

## [Deploying and Managing Applications Protected By CSM for Resiliency](docs/APPLICATIONS.md)
Contains information on how to deploy protected applications and how to know they are protected.

## [Recovering from Failures](docs/RECOVERY.md)
Contains important information about how to recover when failures cannot be resolved automatically.

## [Reporting Problems](docs/PROBLEMS.md)
This section explains what information we need to diagnose the cause of problems with the _CSM for Resiliency_ protection systems. This information should be submitted in any issues if possible.

## [Testing Methodology and Results](docs/TESTING.md)
This section contains information how we tested _CSM for Resiliency_ and the results we achieved.

## Information for Project Contributors
- [Code of Conduct](./docs/CODE_OF_CONDUCT.md)
- Guides
    - [Maintainer Guide](./docs/MAINTAINER_GUIDE.md)
    - [Committer Guide](./docs/COMMITTER_GUIDE.md)
    - [Contributing Guide](./docs/CONTRIBUTING.md)
    - [Getting Started Guide](./docs/GETTING_STARTED_GUIDE.md)
- [List of Adopters](./ADOPTERS.md)
- [Support](#support)
- [About](#about)

## Support

Don’t hesitate to ask! Contact the team and community on the [Support Page](./docs/SUPPORT.md) if you need any help.
Open an issue if you found a bug on [Github Issues](https://github.com/dell/karavi-resiliency/issues).

## Versioning

This project is adhering to [Semantic Versioning](https://semver.org/).

## About

This project is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
