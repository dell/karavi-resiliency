<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Karavi Resiliency
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Releases](https://img.shields.io/badge/Releases-green.svg)](https://github.com/dell/karavi-resiliency/releases)

User applications can have problems if you want their Pods to be resilient to node failure. This is especially true of those deployed with StatefulSets that use PersistentVolumeClaims. Kubernetes guarantees that there will never be two copies of the same StatefulSet Pod running at the same time and accessing storage. Therefore, it does not clean up StatefulSet Pods. 
 
For the complete discussion and rationale, please visit [Pod Safety, Consistency Guarantees, and Storage Implications](https://github.com/kubernetes/community/blob/master/contributors/design-proposals/storage/pod-safety.md).

For more background on forced deletion of Pods in a StatefulSet, please visit [Force Delete StatefulSet Pods](https://kubernetes.io/docs/tasks/run-application/force-delete-stateful-set-pod/#:~:text=In%20normal%20operation%20of%20a,1%20are%20alive%20and%20ready).

Nevertheless, customers are asking that Pods created as part of StatefulSets using PersistentVolumes can be "restarted" on a different node. 
This would have to be accomplished within a reasonable time (a few minutes) if the node they are executing on fails. Since a Pod is never migrated from one node to another, 
the solution is for the old Pod to be terminated, and a replacement pod using the same volumes, to be scheduled on a functioning node.


## Table of Content
- [Code of Conduct](./docs/CODE_OF_CONDUCT.md)
- Guides
    - [Maintainer Guide](./docs/MAINTAINER_GUIDE.md)
    - [Committer Guide](./docs/COMMITTER_GUIDE.md)
    - [Contributing Guide](./docs/CONTRIBUTING.md)
    - [Getting Started Guide](./docs/GETTING_STARTED_GUIDE.md)
- [List of Adopters](./ADOPTERS.md)
- [Release Notes](./docs/RELEASE_NOTES.md)
- [Support](#support)
- [About](#about)

## Support

Don’t hesitate to ask! Contact the team and community on the [Support Page](./docs/SUPPORT.md) if you need any help.
Open an issue if you found a bug on [Github Issues](https://github.com/dell/karavi-resiliency/issues).

## About

This project is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
