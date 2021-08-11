<!--
Copyright (c) 2021 Dell Inc., or its subsidiaries. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
-->

# Getting Started Guidelines
This document steps through the deployment and configuration of the new project

## Prerequisites


## Building New Project
To build the source using the Makefile in the root directory, run:
```
make [all]
```

Or you can `cd` to the `cmd/podman` directory and use the Makefile there for more granular builds.

### Notes/Tips

Before building, you should:

1. Set the GOPROXY environment variable
```
export GOPROXY=http://repository:port/artifactory/devcon-go-gocenter,direct
```
This will ensure that you use the local mirrors for go libraries and docker hub images.


2. Add to insecure registries list in your docker config:
```
vi /etc/docker/daemon.json
``` 
Add the repository to the insecure-registries list:
```
  "insecure-registries" : [
    "registry:port"
    ]
```
Restart docker service:
```
service docker restart
```

## Deploying New Project
CSM for Resiliency is deployed as a side-car to DellEMC CSI Drivers. A reference to the CSM for Resiliency image 
to use should be specified in the DellEMC CSI Driver values.yaml file. An example of that specification:

```yaml
podmon:
  enabled: false
  image: your.registry.hostname:port/podmon:vX.Y.Z
  controller:
    args:
      - "--csisock=unix:/var/run/csi/csi.sock"
      - "--labelvalue=csi-driver"
      - "--driverPath=csi-driver.dellemc.com"
      - "--mode=controller"
      - "--driver-config-params=/csi-driver-config-params/driver-config-params.yaml"
  node:
    args:
      - "--csisock=unix:/var/lib/kubelet/plugins/driver.emc.dell.com/csi_sock"
      - "--labelvalue=csi-driver"
      - "--driverPath=csi-driver.dellemc.com"
      - "--mode=node"
      - "--leaderelection=false"
      - "--driver-config-params=/csi-driver-config-params/driver-config-params.yaml"
```

_NB: The above is generic example. The parameters are not necessarily correct for running with a real DellEMC CSI driver._
_See a CSM for Resiliency supported DellEMC CSI Driver for a better example._

### Dynamic parameters

CSM for Resiliency has configuration parameters that can be updated dynamically, such as the logging level and format. This can be 
done by editing the DellEMC CSI Driver's parameters ConfigMap. The ConfigMap can be queried using kubectl. 
For example, the DellEMC Powerflex CSI Driver ConfigMaps can be found like so: `kubectl get -n vxflexos configmap`. 
The ConfigMap to edit will have this pattern: <storage>-config-params (e.g., `vxflexos-config-params`).

To update or add parameters, you can use the `kubectl edit` command. For example, `kubectl edit -n vxflexos configmap vxflexos-config-params`.

This is a list of parameters that can be adjusted for CSM for Resiliency:

| Parameter | Type | Default | Description |
| --------- | ---- | ------- | ----------- |
| PODMON_CONTROLLER_LOG_FORMAT | String | "TEXT" |Logging format output for the controller podmon sidecar. Should be "text" or "json" |
| PODMON_CONTROLLER_LOG_LEVEL | String | "debug" |Logging level for the controller podmon sidecar. Standard values: 'info', 'error', 'warning', 'debug', 'trace' |
| PODMON_NODE_LOG_FORMAT | String | "TEXT" |Logging format output for the node podmon sidecar. Should be "text" or "json" |
| PODMON_NODE_LOG_LEVEL | String | "debug" |Logging level for the node podmon sidecar. Standard values: 'info', 'error', 'warning', 'debug', 'trace' |
| PODMON_ARRAY_CONNECTIVITY_POLL_RATE | Integer (>0) | 15 |An interval in seconds to poll the underlying array | 
| PODMON_ARRAY_CONNECTIVITY_CONNECTION_LOSS_THRESHOLD | Integer (>0) | 3 |A value representing the number of failed connection poll intervals before marking the array connectivity as lost |
| PODMON_SKIP_ARRAY_CONNECTION_VALIDATION | Boolean | false |Flag to disable the array connectivity check |

Before building the image, set up some environmental variables:
```shell
export REGISTRY_HOST=your.registry.hostname
export REGISTRY_PORT=port
export VERSION=vX.Y.Z
```

Use make to build and push the image to a repo:
```shell
cd cmd/podmon
make docker 
make push
```

## Testing New Project
Clone the source:
```shell
git clone https://github.com/dell/karavi-resiliency.git
```

Change dir to cmd/podmon
```shell
cd cmd/podmon
```

Run test using make
```shell
make godog
```