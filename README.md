<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Dell EMC Container Storage Modules (CSM) for Resiliency
[![Contributor Covenant](https://img.shields.io/badge/Contributor%20Covenant-v2.0%20adopted-ff69b4.svg)](https://github.com/dell/csm/blob/main/docs/CODE_OF_CONDUCT.md)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
[![Docker Pulls](https://img.shields.io/docker/pulls/dellemc/podmon)](https://hub.docker.com/r/dellemc/podmon)
[![Go version](https://img.shields.io/github/go-mod/go-version/dell/karavi-resiliency)](go.mod)
[![GitHub release (latest by date including pre-releases)](https://img.shields.io/github/v/release/dell/karavi-resiliency?include_prereleases&label=latest&style=flat-square)](https://github.com/dell/karavi-resiliency/releases/latest)
[![Releases](https://img.shields.io/badge/Releases-green.svg)](https://github.com/dell/karavi-resiliency/releases)

_CSM for Resiliency_ is a project designed to make Kubernetes Applications, including those that utilize persistent storage, more resilient to various failures. The first component of _CSM for Resiliency_ is a pod monitor that is specifically designed to protect stateful applications from various failures. It is not a standalone application, but rather is deployed as a _sidecar_ to CSI (Container Storage Interface) drivers, in both the driver's controller pods and the driver's node pods. Deploying CSM for Resiliency as a sidecar allows it to make direct requests to the driver through the Unix domain socket that Kubernetes sidecars use to make CSI requests.

Some of the methods CSM for Resiliency invokes in the driver are standard CSI methods, such as NodeUnpublishVolume, NodeUnstageVolume, and ControllerUnpublishVolume. CSM for Resiliency also uses proprietary calls that are not part of the standard CSI specification. Currently, there is only one, ValidateVolumeHostConnectivity that returns information on whether a host is connected to the storage system and/or whether any I/O activity has happened in the recent past from a list of specified volumes. This allows CSM for Resiliency to make more accurate determinations about the state of the system and its persistent volumes.

Accordingly, CSM for Resiliency is adapted to, and qualified with each Dell EMC CSI driver it is to be used with. Different storage systems have different nuances and characteristics that CSM for Resiliency must take into account.

For documentation, please visit [Container Storage Modules documentation](https://dell.github.io/csm-docs/).

# Table of Contents

- [Code of Conduct](https://github.com/dell/csm/blob/main/docs/CODE_OF_CONDUCT.md)
- [Maintainer Guide](https://github.com/dell/csm/blob/main/docs/MAINTAINER_GUIDE.md)
- [Committer Guide](https://github.com/dell/csm/blob/main/docs/COMMITTER_GUIDE.md)
- [Contributing Guide](https://github.com/dell/csm/blob/main/docs/CONTRIBUTING.md)
- [Branching Strategy](https://github.com/dell/csm/blob/main/docs/BRANCHING.md)
- [List of Adopters](https://github.com/dell/csm/blob/main/ADOPTERS.md)
- [Maintainers](https://github.com/dell/csm/blob/main/docs/MAINTAINERS.md)
- [Support](https://github.com/dell/csm/blob/main/docs/SUPPORT.md)
- [Security](https://github.com/dell/csm/blob/main/docs/SECURITY.md)
- [About](#about)

## Support

For all your support needs or to follow the latest ongoing discussions and updates, join our Slack group. Click [Here](http://del.ly/Slack_request) to request your invite.

You can also interact with us on [GitHub](https://github.com/dell/csm) by creating a [GitHub Issue](https://github.com/dell/csm/issues).

## Versioning

This project is adhering to [Semantic Versioning](https://semver.org/).

## About

Dell EMC Container Storage Modules (CSM) is 100% open source and community-driven. All components are available
under [Apache 2 License](https://www.apache.org/licenses/LICENSE-2.0.html) on
GitHub.
